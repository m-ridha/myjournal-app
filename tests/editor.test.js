// Dependency-free interaction regressions. Run from the repository root with
// `node tests/editor.test.js`, or macOS JavaScriptCore's `jsc tests/editor.test.js`.
const output = typeof print === 'function' ? print : console.log;
const assert = (condition, message) => { if (!condition) throw new Error(message); };
const tick = async () => { for (let i = 0; i < 16; i += 1) await Promise.resolve(); };
const deferred = () => { let resolve; let reject; const promise = new Promise((a, b) => { resolve = a; reject = b; }); return { promise, resolve, reject }; };
const response = (text, version = 'v1', status = 200) => ({ ok: status === 200, status, json: async () => ({ text, version, exists: true }) });
const date = '2020-01-01';
const key = `mj-draft:v3:account-a:${date}`;

function element() {
  const listeners = {};
  const classes = new Set();
  return { value: '', textContent: '', disabled: false, hidden: false,
    addEventListener: (name, fn) => { listeners[name] = fn; },
    fire: (name, event = {}) => listeners[name]?.(event),
    setAttribute() {},
    classList: { add: (x) => classes.add(x), remove: (...xs) => xs.forEach((x) => classes.delete(x)), contains: (x) => classes.has(x) },
  };
}

function setup({ draft = null, initial = response('server'), storageThrows = false, signOut = false, legacy = null, sessionValues = new Map(), localValues = new Map() } = {}) {
  const panel = element(); const mine = element(); const other = element(); const both = element();
  const text = element(); const status = element(); const save = element(); const retry = element(); const old = element(); const button = element();
  const fields = { '[data-entry-conflict]': panel, '[data-conflict-mine]': mine, '[data-conflict-other]': other, '[data-conflict-both]': both, '[data-entry-text]': text, '[data-entry-status]': status, '[data-entry-save]': save,
    '[data-entry-load-retry]': retry, '[data-entry-legacy-draft]': old };
  const editor = element(); editor.dataset = { entryDate: date }; editor.querySelector = (selector) => fields[selector] || null;
  const values = sessionValues; if (draft !== null) values.set(key, JSON.stringify(draft));
  if (legacy !== null) localValues.set(`mj-draft:${date}`, legacy);
  const guard = () => { if (storageThrows) throw new Error('storage denied'); };
  const store = (map) => ({ getItem: (k) => { guard(); return map.has(k) ? map.get(k) : null; },
    setItem: (k, v) => { guard(); map.set(k, v); }, removeItem: (k) => { guard(); map.delete(k); },
    key: (i) => { guard(); return [...map.keys()][i] ?? null; }, get length() { guard(); return map.size; } });
  globalThis.localStorage = store(localValues);
  globalThis.sessionStorage = store(values);
  const docEvents = {}; const winEvents = {};
  globalThis.document = { body: { dataset: { messages: '{}', draftScope: 'account-a', today: '2026-10-08' } },
    querySelector: (s) => s === '[data-entry-editor]' ? (signOut ? null : editor) : s === '[data-sign-out]' ? (signOut ? button : null) : s === '[data-sign-out-status]' ? status : null,
    addEventListener: (name, fn) => { docEvents[name] = fn; } };
  globalThis.window = { location: { href: '' }, confirm: () => true, addEventListener: (name, fn) => { winEvents[name] = fn; } };
  const calls = []; const queue = [initial];
  globalThis.fetch = async (url, options) => { calls.push({ url, options }); if (!queue.length) throw new Error('unplanned request'); return queue.shift(); };
  if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync('assets/static/js/app.js', 'utf8'));
  else load('assets/static/js/app.js');
  docEvents.DOMContentLoaded();
  return { localValues, panel, mine, other, both, text, status, save, retry, old, button, editor, values, calls, queue, winEvents,
    input(value) { text.value = value; text.fire('input'); } };
}

async function run() {
  let f = setup({ draft: { text: 'recovered', baseVersion: 'old', versionKnown: true } }); await tick();
  assert(f.text.value === 'recovered' && f.status.textContent === 'editor.conflict', 'server text must not replace draft');
  let warned = false; f.winEvents.beforeunload({ preventDefault() { warned = true; } }); assert(warned, 'restored draft must be dirty');
  f.queue.push(response('', '', 409)); f.save.fire('click'); await tick();
  assert(JSON.parse(f.calls[1].options.body).baseVersion === 'old', 'recovered draft must retain stale version');
  assert(f.text.value === 'recovered' && f.values.has(key) && !f.save.disabled, 'conflict must retain text and enable retry');

  f = setup(); await tick(); f.input('sent'); const pendingSave = deferred(); f.queue.push(pendingSave.promise);
  f.save.fire('click'); f.input('later'); f.save.fire('click');
  assert(f.save.disabled && f.calls.length === 2, 'typing must not permit overlapping saves');
  pendingSave.resolve(response('sent', 'v2')); await tick();
  assert(f.text.value === 'later' && f.status.textContent === 'editor.unsaved' && !f.editor.classList.contains('is-saved'), 'unsent text must not be marked saved');
  assert(JSON.parse(f.values.get(key)).text === 'later' && JSON.parse(f.values.get(key)).baseVersion === 'v2', 'later draft needs latest acknowledged version');
  f.queue.push(response('later', 'v3')); f.save.fire('click'); await tick();
  assert(!f.values.has(key) && f.editor.classList.contains('is-saved'), 'matching saved snapshot clears draft');

  f.input(''); assert(JSON.parse(f.values.get(key)).text === '', 'intentional empty draft must survive');
  f.queue.push(Promise.reject(new Error('offline'))); f.save.fire('click'); await tick();
  assert(!f.save.disabled && f.text.value === '' && f.values.has(key), 'network failure must preserve empty edits and enable Save');
  f.queue.push(response('', '', 401)); f.save.fire('click'); await tick();
  assert(f.status.textContent.includes('editor.sessionExpired') && f.values.has(key), 'expired session must retain draft');

  f = setup(); await tick(); f.input('sent'); const lostResponse = deferred(); f.queue.push(lostResponse.promise); f.save.fire('click');
  f.input('server'); assert(f.values.has(key), 'reverting to old baseline during save still needs a backup');
  lostResponse.reject(new Error('response lost')); await tick();
  let uncertainWarned = false; f.winEvents.beforeunload({ preventDefault() { uncertainWarned = true; } });
  assert(uncertainWarned && f.values.has(key) && !f.save.disabled, 'ambiguous save failure must retain rollback text as dirty');

  f = setup({ storageThrows: true }); await tick(); f.input('keep me'); f.queue.push(Promise.reject(new Error('offline'))); f.save.fire('click'); await tick();
  assert(!f.save.disabled && f.text.value === 'keep me' && f.status.textContent.includes('editor.backupUnavailable') && !f.status.textContent.includes('editor.draftKept'), 'storage errors must not claim backup or disable saving');

  const loading = deferred(); f = setup({ initial: loading.promise }); f.input('typed during load');
  assert(f.save.disabled && f.text.readOnly, 'cannot edit or save before base version is known'); loading.resolve(response('server')); await tick();
  assert(f.text.value === 'typed during load' && f.values.has(key) && !f.text.readOnly, 'load must preserve programmatic input and unlock after success');

  const recovery = deferred(); f = setup({ initial: recovery.promise, draft: { text: 'old text', baseVersion: 'old', versionKnown: true } });
  f.input('edited recovery'); assert(JSON.parse(f.values.get(key)).versionKnown, 'typing while loading must retain recovered version knowledge');
  recovery.resolve(response('new server', 'new')); await tick(); f.queue.push(response('', '', 409)); f.save.fire('click'); await tick();
  assert(JSON.parse(f.calls[1].options.body).baseVersion === 'old', 'typing during recovery must not silently adopt new server version');

  f = setup({ initial: response('', '', 503), draft: { text: '', baseVersion: 'old', versionKnown: true } }); await tick();
  assert(f.retry.hidden === false && f.save.disabled && f.text.readOnly, 'failed load offers retry while preserving read-only text');
  f.queue.push(response('server', 'new')); f.retry.fire('click'); await tick();
  assert(f.text.value === '' && !f.save.disabled && f.values.has(key), 'retry must preserve recovered empty draft');

  f = setup({ legacy: 'legacy' }); await tick(); assert(f.text.value === 'server' && !f.old.hidden, 'unscoped draft requires explicit recovery');
  f.old.fire('click'); assert(f.text.value === 'legacy' && f.values.has(key) && !f.localValues.has(`mj-draft:${date}`), 'explicit recovery migrates only after backup succeeds');

  f = setup({ signOut: true }); f.values.set(key, 'draft'); f.localValues.set('mj-draft:v2:account-b:2020-01-01', 'other'); f.localValues.set(`mj-draft:${date}`, 'legacy'); f.localValues.set('mj-theme', 'dark');
  f.queue.shift(); f.queue.push({ ok: true }); f.button.fire('click'); await tick();
  assert(!f.values.has(key) && !f.localValues.has(`mj-draft:${date}`) && f.localValues.has('mj-draft:v2:account-b:2020-01-01') && f.localValues.has('mj-theme'), 'signout must clear current account and legacy drafts only');
  assert(window.location.href === '/', 'successful logout returns home');
  f = setup({ signOut: true, storageThrows: true }); f.button.fire('click'); await tick();
  assert(f.calls.length === 0 && f.status.textContent === 'settings.draftCleanupFailed', 'cleanup failure must be visible before signout');

  f = setup(); await tick(); f.input('open tab'); f.winEvents.storage({ key: 'mj-signout:account-a' }); f.input('later input');
  assert(f.save.disabled && !f.values.has(key), 'signed-out tabs must stop rewriting drafts');
  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  assert(!f.panel.hidden, 'stale recovered backup must offer conflict actions');
  f.queue.push(response('other', 'latest'), response('mine', 'saved'));
  f.mine.fire('click'); await tick();
  assert(JSON.parse(f.calls[2].options.body).baseVersion === 'latest' && f.panel.hidden && !f.values.has(key), 'keep mine must save against fetched version');

  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  window.confirm = () => false; f.other.fire('click'); await tick();
  assert(f.text.value === 'mine' && f.calls.length === 1, 'cancel must preserve mine without a request');
  window.confirm = () => true; f.queue.push(response('other', 'latest')); f.other.fire('click'); await tick();
  assert(f.text.value === 'other' && !f.values.has(key) && f.panel.hidden, 'load other must accept saved text and clear this tab backup');

  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  f.queue.push(response('other', 'latest')); f.both.fire('click'); await tick();
  assert(f.text.value === 'mine\n\nother' && f.calls.length === 2 && f.values.has(key), 'keep both must combine without auto-saving');
  f.queue.push(response('', '', 409)); f.save.fire('click'); await tick();
  assert(!f.panel.hidden && f.text.value === 'mine\n\nother', 'a second concurrent writer must conflict without losing combined text');

  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  f.queue.push(response('', '', 503)); f.mine.fire('click'); await tick();
  assert(f.text.value === 'mine' && !f.panel.hidden && !f.mine.disabled && f.status.textContent.includes('editor.resolveFailed'), 'failed resolution must preserve draft and allow retry');

  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  f.queue.push(response('', '')); const mineSave = deferred(); f.queue.push(mineSave.promise);
  f.mine.fire('click'); await tick();
  assert(JSON.parse(f.calls[2].options.body).baseVersion === '', 'keep mine can restore a remotely deleted entry against empty version');
  f.input('newer text'); mineSave.resolve(response('mine', 'restored')); await tick();
  assert(f.text.value === 'newer text' && f.status.textContent === 'editor.unsaved', 'keep mine acknowledgement must preserve newer typing');

  const sharedLocal = new Map(); const firstTab = new Map(); const secondTab = new Map();
  f = setup({ sessionValues: firstTab, localValues: sharedLocal }); await tick(); f.input('first tab');
  f = setup({ sessionValues: secondTab, localValues: sharedLocal }); await tick(); f.input('second tab');
  assert(JSON.parse(firstTab.get(key)).text === 'first tab' && JSON.parse(secondTab.get(key)).text === 'second tab', 'same-date tabs must keep independent backups');
  f = setup({ sessionValues: firstTab, localValues: sharedLocal }); await tick();
  assert(f.text.value === 'first tab', 'reload must recover only this tab text');
  f.queue.push(response('first tab', 'v2')); f.save.fire('click'); await tick();
  assert(!firstTab.has(key) && secondTab.has(key), 'saving one tab must not clear the other tab backup');

  const oldScoped = new Map([[`mj-draft:v2:account-a:${date}`, JSON.stringify({text:'old scoped',baseVersion:'old',versionKnown:true})]]);
  f = setup({ localValues: oldScoped }); await tick(); f.old.fire('click');
  assert(JSON.parse(f.values.get(key)).baseVersion === 'old' && !oldScoped.has(`mj-draft:v2:account-a:${date}`), 'migration must retain the old scoped version');
  f.queue.push(response('', '', 409)); f.save.fire('click'); await tick();
  assert(!f.panel.hidden, 'migrated stale backup must still conflict');

  output('PASS editor interaction regressions');
}

run().catch((error) => { output(error.stack || String(error)); if (typeof quit === 'function') quit(1); else process.exitCode = 1; });
