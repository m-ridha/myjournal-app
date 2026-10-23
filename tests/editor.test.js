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
  return {
    value: '', textContent: '', disabled: false, hidden: false,
    addEventListener: (name, fn) => { listeners[name] = fn; },
    fire: (name, event = {}) => listeners[name]?.(event),
    setAttribute() { },
    classList: { add: (x) => classes.add(x), remove: (...xs) => xs.forEach((x) => classes.delete(x)), contains: (x) => classes.has(x) },
  };
}

function setup({ draft = null, initial = response('server'), storageThrows = false, signOut = false, legacy = null, sessionValues = new Map(), localValues = new Map() } = {}) {
  const auth = element(); auth.hidden = true;
  const trash = element(); const panel = element(); const mine = element(); const other = element(); const both = element();
  const text = element(); const status = element(); const save = element(); const retry = element(); const old = element(); const button = element(); const disconnect = element();
  const fields = {
    '[data-entry-delete]': trash, '[data-entry-conflict]': panel, '[data-conflict-mine]': mine, '[data-conflict-other]': other, '[data-conflict-both]': both, '[data-entry-text]': text, '[data-entry-status]': status, '[data-entry-save]': save,
    '[data-entry-load-retry]': retry, '[data-entry-legacy-draft]': old
  };
  const editor = element(); editor.dataset = { entryDate: date }; editor.querySelector = (selector) => fields[selector] || null;
  const values = sessionValues; if (draft !== null) values.set(key, JSON.stringify(draft));
  if (legacy !== null) localValues.set(`mj-draft:${date}`, legacy);
  const guard = () => { if (storageThrows) throw new Error('storage denied'); };
  const store = (map) => ({
    getItem: (k) => { guard(); return map.has(k) ? map.get(k) : null; },
    setItem: (k, v) => { guard(); map.set(k, v); }, removeItem: (k) => { guard(); map.delete(k); },
    key: (i) => { guard(); return [...map.keys()][i] ?? null; }, get length() { guard(); return map.size; }
  });
  globalThis.localStorage = store(localValues);
  globalThis.sessionStorage = store(values);
  let now = 0; let nextTimer = 0; const timers = new Map();
  globalThis.setTimeout = (fn, delay) => { const id = ++nextTimer; timers.set(id, { fn, at: now + delay }); return id; };
  globalThis.clearTimeout = (id) => timers.delete(id);
  const docEvents = {}; const winEvents = {};
  globalThis.document = {
    body: { dataset: { messages: '{}', draftScope: 'account-a', today: '2026-10-08' } },
    querySelector: (s) => s === '[data-auth-expired]' ? auth : s === '[data-entry-editor]' ? (signOut ? null : editor) : s === '[data-sign-out]' ? (signOut ? button : null) : s === '[data-disconnect]' ? (signOut ? disconnect : null) : s === '[data-sign-out-status]' ? status : null,
    addEventListener: (name, fn) => { docEvents[name] = fn; }
  };
  globalThis.window = { location: { href: '' }, confirm: () => true, addEventListener: (name, fn) => { winEvents[name] = fn; } };
  const calls = []; const queue = [initial];
  globalThis.fetch = async (url, options) => { calls.push({ url, options }); if (!queue.length) throw new Error('unplanned request'); return queue.shift(); };
  for (const script of ['draft-cleanup', 'app']) {
    if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync(`assets/static/js/${script}.js`, 'utf8'));
    else load(`assets/static/js/${script}.js`);
  }
  docEvents.DOMContentLoaded();
  return {
    auth, docEvents, async advance(ms) {
      const end = now + ms;
      while (true) {
        const due = [...timers].filter(([, timer]) => timer.at <= end).sort((a, b) => a[1].at - b[1].at)[0];
        if (!due) break;
        now = due[1].at; timers.delete(due[0]); due[1].fn(); await tick();
      }
      now = end; await tick();
    }, localValues, trash, panel, mine, other, both, text, status, save, retry, old, button, disconnect, editor, values, calls, queue, winEvents,
    input(value) { text.value = value; text.fire('input'); }
  };
}

async function run() {
  for (const exists of [false, true]) {
    const blank = setup({ initial: { ok: true, status: 200, json: async () => ({ text: '', version: exists ? 'v1' : '', exists }) } });
    await tick();
    const expected = exists ? 'editor.saved' : 'editor.blank';
    assert(blank.status.textContent === expected, 'initial empty status reflects file existence');
    blank.input('temporary draft');
    assert(blank.status.textContent === 'editor.unsaved', 'typing into an empty entry is unsaved');
    blank.input('');
    assert(blank.status.textContent === expected, 'erasing draft restores status according to file existence');
    await blank.advance(5000);
    assert(blank.calls.length === 1 && !blank.values.has(key), 'reverted empty draft does not write or retain a backup');
  }

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

  // A permanent size rejection must explain the limit without losing newer edits.
  f = setup(); await tick(); f.input('oversized snapshot');
  const oversizedSave = deferred(); f.queue.push(oversizedSave.promise); f.save.fire('click');
  f.input('newer oversized draft'); oversizedSave.resolve(response('', '', 413)); await tick();
  assert(f.status.textContent.includes('editor.saveTooLarge') && !f.status.textContent.includes('editor.saveFailed'), '413 must explain the size limit instead of a generic failure');
  assert(f.text.value === 'newer oversized draft' && JSON.parse(f.values.get(key)).text === 'newer oversized draft' && JSON.parse(f.values.get(key)).baseVersion === 'v1' && !f.save.disabled, '413 preserves current text, backup, base version, and editing');
  let sizeWarned = false; f.winEvents.beforeunload({ preventDefault() { sizeWarned = true; } });
  assert(sizeWarned && !f.editor.classList.contains('is-saved'), 'rejected text remains unsaved with a leave warning');
  const rejectedBackup = f.values;
  f = setup({ sessionValues: rejectedBackup }); await tick();
  assert(f.text.value === 'newer oversized draft' && f.status.textContent === 'editor.draftRestored', '413 backup must survive reload');
  f.input('shorter'); f.queue.push(response('shorter', 'v2')); f.save.fire('click'); await tick();
  assert(!f.values.has(key) && f.editor.classList.contains('is-saved'), 'shortening after recovery can save normally');

  f = setup({ storageThrows: true }); await tick(); f.input('keep oversized text');
  f.queue.push(response('', '', 413)); f.save.fire('click'); await tick();
  assert(f.status.textContent.includes('editor.saveTooLarge') && f.status.textContent.includes('editor.backupUnavailable') && !f.status.textContent.includes('editor.draftKept') && f.text.value === 'keep oversized text' && !f.save.disabled, '413 with unavailable backup must keep visible text and give truthful recovery guidance');

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

  const failedTab = new Map([[key, JSON.stringify({ text: 'recover me', baseVersion: 'v1', versionKnown: true })]]);
  const failedLocal = new Map([[`mj-draft:${date}`, 'older draft'], [`mj-draft:v2:account-a:${date}`, 'scoped backup']]);
  f = setup({ signOut: true, sessionValues: failedTab, localValues: failedLocal });
  const logout = deferred(); f.queue.shift(); f.queue.push(logout.promise); f.button.fire('click'); await tick();
  assert(f.values.has(key) && f.localValues.has(`mj-draft:${date}`) && !f.localValues.has('mj-signout:account-a'), 'pending logout must not erase or broadcast');
  logout.resolve({ ok: false, status: 503 }); await tick();
  assert(!f.button.disabled && f.status.textContent === 'settings.signOutFailed' && f.values.has(key) && f.localValues.has(`mj-draft:${date}`) && f.localValues.has(`mj-draft:v2:account-a:${date}`) && !f.localValues.has('mj-signout:account-a'), '503 logout must retain every backup and avoid locking editors');
  // Reload the editor using the retained tab backup: normal editing/save works.
  f = setup({ sessionValues: failedTab, localValues: failedLocal }); await tick();
  assert(f.text.value === 'recover me' && !f.text.readOnly && !f.save.disabled, 'editor can recover after failed logout');
  f.input('still editable'); f.queue.push(response('still editable', 'v2')); f.save.fire('click'); await tick();
  assert(!f.values.has(key) && f.editor.classList.contains('is-saved'), 'editor can save after failed logout');

  f = setup({ signOut: true, draft: { text: 'retain on network failure', baseVersion: 'v1', versionKnown: true }, legacy: 'retain legacy' });
  f.queue.shift(); f.queue.push(Promise.reject(new Error('network failure'))); f.button.fire('click'); await tick();
  assert(f.values.has(key) && f.localValues.has(`mj-draft:${date}`) && !f.localValues.has('mj-signout:account-a') && !f.button.disabled, 'network failure must preserve backups without broadcasting');
  f.queue.push({ ok: true }); f.button.fire('click'); await tick();
  assert(!f.values.has(key) && !f.localValues.has(`mj-draft:${date}`) && f.localValues.has('mj-signout:account-a'), 'successful retry can finalize cleanup');

  f = setup({ signOut: true, draft: { text: 'cleanup failure', baseVersion: 'v1', versionKnown: true } });
  const remove = sessionStorage.removeItem;
  sessionStorage.removeItem = (k) => { if (k === key) throw new Error('cleanup denied'); remove(k); };
  f.queue.shift(); f.queue.push({ ok: true }); f.button.fire('click'); await tick();
  assert(f.status.textContent === 'settings.signedOutCleanupFailed' && f.button.textContent === 'settings.retryDraftCleanup' && f.values.has(key), 'post-logout cleanup failure must distinguish signed-out state');
  sessionStorage.removeItem = remove; f.button.fire('click'); await tick();
  assert(!f.values.has(key) && f.calls.length === 1 && window.location.href === '/', 'cleanup retry must not repeat logout');

  f = setup(); await tick(); f.input('open tab'); f.winEvents.storage({ key: 'mj-signout:account-a' }); f.input('later input');
  assert(f.save.disabled && !f.values.has(key), 'signed-out tabs must stop rewriting drafts');
  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  assert(!f.panel.hidden, 'stale recovered backup must offer conflict actions');
  f.queue.push(response('other', 'latest'), response('mine', 'saved'));
  f.mine.fire('click'); await tick();
  assert(f.calls[1].options.headers['Cache-Control'] === 'no-store', 'conflict resolution bypasses server content cache');
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

  const oldScoped = new Map([[`mj-draft:v2:account-a:${date}`, JSON.stringify({ text: 'old scoped', baseVersion: 'old', versionKnown: true })]]);
  f = setup({ localValues: oldScoped }); await tick(); f.old.fire('click');
  assert(JSON.parse(f.values.get(key)).baseVersion === 'old' && !oldScoped.has(`mj-draft:v2:account-a:${date}`), 'migration must retain the old scoped version');
  f.queue.push(response('', '', 409)); f.save.fire('click'); await tick();
  assert(!f.panel.hidden, 'migrated stale backup must still conflict');

  // Manual retries verify uncertain saves before risking another upload.
  for (const failure of [() => response('', '', 503), () => Promise.reject(new Error('lost acknowledgement')),
  () => ({ ok: true, status: 200, json: async () => { throw new Error('truncated acknowledgement'); } })]) {
    f = setup(); await tick(); f.input('uncertain snapshot'); f.queue.push(failure()); f.save.fire('click'); await tick();
    f.queue.push(response('uncertain snapshot', 'v2')); f.save.fire('click'); await tick();
    assert(f.calls[2].options.headers['Cache-Control'] === 'no-store', 'save reconciliation bypasses server content cache');
    assert(!f.calls[2].options?.method && f.calls.length === 3, 'retry must read matching saved text without another PUT');
    assert(!f.values.has(key) && f.editor.classList.contains('is-saved'), 'matching read acknowledges the uncertain snapshot');
    f.input('next'); f.queue.push(response('next', 'v3')); await f.advance(5000);
    assert(JSON.parse(f.calls[3].options.body).baseVersion === 'v2', 'follow-up autosave uses verified version');
  }
  f = setup(); await tick(); f.input('uncertain'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  const verifying = deferred(); f.queue.push(verifying.promise); f.save.fire('click'); await tick();
  f.input('typed during verification'); f.save.fire('click');
  assert(f.calls.length === 3 && f.save.disabled && !f.text.readOnly, 'verification serializes saves while allowing typing');
  verifying.resolve(response('uncertain', 'v2')); await tick();
  assert(f.text.value === 'typed during verification' && f.status.textContent === 'editor.unsaved' && JSON.parse(f.values.get(key)).baseVersion === 'v2', 'verified snapshot must preserve later edits and their backup');

  f = setup(); await tick(); f.input('mine'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  f.queue.push(response('server', 'v1'), response('mine', 'v2')); f.save.fire('click'); await tick();
  assert(!f.calls[2].options?.method && f.calls[3].options.method === 'PUT' && JSON.parse(f.calls[3].options.body).baseVersion === 'v1', 'uncommitted retry keeps original conditional version');

  f = setup(); await tick(); f.input('mine'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  f.queue.push(response('external change', 'v9'), response('', '', 409)); f.save.fire('click'); await tick();
  assert(JSON.parse(f.calls[3].options.body).baseVersion === 'v1' && !f.panel.hidden && f.text.value === 'mine' && f.values.has(key), 'verification must never silently rebase different server text');

  for (const failedRead of [() => response('', '', 401), () => response('', '', 503),
  () => Promise.reject(new Error('offline')), () => ({ ok: true, status: 200, json: async () => ({ text: 'mine', version: '', exists: true }) })]) {
    f = setup(); await tick(); f.input('mine'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
    f.queue.push(failedRead()); f.save.fire('click'); await tick();
    assert(f.calls.length === 3 && !f.calls[2].options?.method && f.text.value === 'mine' && f.values.has(key) && !f.save.disabled, 'failed verification must retain draft without a PUT');
    assert(JSON.parse(f.values.get(key)).baseVersion === 'v1', 'failed verification retains original version');
    f.text.fire('blur'); await f.advance(10000); assert(f.calls.length === 3, 'failed verification must keep autosave paused');
  }

  f = setup(); await tick(); f.input('snapshot'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  const unchangedRead = deferred(); f.queue.push(unchangedRead.promise, response('snapshot', 'v2')); f.save.fire('click'); await tick();
  f.input('newer while reading'); unchangedRead.resolve(response('server', 'v1')); await tick();
  assert(JSON.parse(f.calls[3].options.body).text === 'snapshot' && f.text.value === 'newer while reading' && JSON.parse(f.values.get(key)).baseVersion === 'v2', 'read-before-retry uploads its snapshot and retains later typing');

  f = setup(); await tick(); f.input(''); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  f.queue.push({ ok: true, status: 200, json: async () => ({ exists: false, text: '', version: '' }) }); f.save.fire('click'); await tick();
  assert(f.calls.length === 3 && !f.values.has(key) && f.status.textContent === 'editor.blank', 'verified absence acknowledges an empty snapshot without creating a file');

  f = setup(); await tick(); f.input('uncertain'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  const signingOutRead = deferred(); f.queue.push(signingOutRead.promise); f.save.fire('click'); await tick();
  f.winEvents.storage({ key: 'mj-signout:account-a' }); signingOutRead.resolve(response('server', 'v1')); await tick();
  assert(f.calls.length === 3 && !f.values.has(key), 'signout during verification prevents a later PUT or backup recreation');

  f = setup(); await tick(); f.input('committed'); f.queue.push(response('', '', 503)); f.save.fire('click'); await tick();
  const afterFailureBackup = f.values;
  f = setup({ initial: response('committed', 'v2'), sessionValues: afterFailureBackup }); await tick();
  assert(f.calls[0].options.headers['Cache-Control'] === 'no-store', 'draft recovery bypasses server content cache');
  assert(f.text.value === 'committed' && !f.values.has(key) && f.calls.length === 1, 'reload reconciles matching recovery through its initial GET without writing');

  // Autosave uses the same acknowledged snapshot/version and recovery rules.
  f = setup(); await tick(); await f.advance(10000);
  assert(f.calls.length === 1, 'loading an unchanged entry must not write');
  f.input('first'); await f.advance(4999); assert(f.calls.length === 1, 'wait five seconds after input');
  f.input('second'); await f.advance(4999); assert(f.calls.length === 1, 'typing resets idle debounce');
  f.queue.push(response('second', 'v2')); await f.advance(1);
  assert(f.calls.length === 2 && JSON.parse(f.calls[1].options.body).text === 'second' && !f.values.has(key), 'idle autosave acknowledges current snapshot');
  f.text.fire('blur'); await f.advance(10000); assert(f.calls.length === 2, 'unchanged blur/timer must not save again');
  f.input('transient'); f.input('second'); await f.advance(5000); assert(f.calls.length === 2, 'reverting to acknowledged text suppresses saves');

  f = setup(); await tick(); f.input('snapshot'); const autoPending = deferred(); f.queue.push(autoPending.promise); await f.advance(5000);
  f.input('newer'); await f.advance(10000); assert(f.calls.length === 2, 'long saves never overlap');
  autoPending.resolve(response('snapshot', 'v2')); await tick();
  assert(f.status.textContent === 'editor.unsaved' && JSON.parse(f.values.get(key)).text === 'newer', 'later edits remain dirty after autosave');
  f.queue.push(response('newer', 'v3')); await f.advance(5000);
  assert(f.calls.length === 3 && JSON.parse(f.calls[2].options.body).baseVersion === 'v2' && !f.values.has(key), 'follow-up autosave uses returned version');

  for (const failure of [response('', '', 401), response('', '', 409), response('', '', 413), response('', '', 503), Promise.reject(new Error('offline'))]) {
    f = setup(); await tick(); f.input('preserve'); f.queue.push(failure); await f.advance(5000);
    assert(f.text.value === 'preserve' && f.values.has(key) && !f.save.disabled, 'autosave failure preserves editable backup');
    f.input('preserve later'); f.text.fire('blur'); f.winEvents.pagehide?.(); await f.advance(20000);
    assert(f.calls.length === 2, 'failed autosave must not silently retry through typing/blur/pagehide');
    f.queue.push(response('preserve later', 'v2')); f.save.fire('click'); await tick();
    f.input('resume'); f.queue.push(response('resume', 'v3')); await f.advance(5000);
    assert(f.calls.length === 4 && !f.values.has(key), 'successful manual retry resumes autosave');
  }

  f = setup({ draft: { text: 'review me', baseVersion: 'v1', versionKnown: true } }); await tick();
  f.text.fire('blur'); f.winEvents.pagehide?.(); await f.advance(10000);
  assert(f.calls.length === 1 && f.values.has(key), 'restored draft waits for review before autosave');
  f.input('reviewed'); f.queue.push(response('reviewed', 'v2')); await f.advance(5000); assert(f.calls.length === 2, 'editing a restored draft opts into autosave');

  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true } }); await tick();
  f.input('edited conflict'); await f.advance(5000); assert(f.calls.length === 1, 'typing must not bypass conflict controls');
  f.queue.push(response('other', 'v2')); f.both.fire('click'); await tick();
  f.text.fire('blur'); f.winEvents.pagehide?.(); await f.advance(10000);
  assert(f.calls.length === 2 && f.text.value === 'edited conflict\n\nother', 'keep both requires review before saving');

  f = setup(); await tick(); f.input('blur save'); f.queue.push(response('blur save', 'v2')); f.text.fire('blur'); await tick();
  assert(f.calls.length === 2 && !f.values.has(key), 'blur flushes a pending edit');
  f.input('leave snapshot'); const leaving = deferred(); f.queue.push(leaving.promise); f.winEvents.pagehide(); await tick();
  assert(f.calls.length === 3 && f.calls[2].options.keepalive === true && f.values.has(key), 'pagehide attempts keepalive but retains backup without acknowledgement');
  leaving.resolve(response('leave snapshot', 'v3')); await tick(); assert(!f.values.has(key), 'only acknowledgement clears pagehide backup');

  for (const choice of ['other', 'both']) {
    f = setup(); await tick(); f.input('mine'); f.queue.push(response('', '', 409)); await f.advance(5000);
    f.queue.push(response('theirs', 'v2')); f[choice].fire('click'); await tick(); await f.advance(10000);
    assert(f.calls.length === 3, 'conflict resolution must not autosave review output by itself');
    f.input('reviewed resolution'); f.queue.push(response('reviewed resolution', 'v3')); await f.advance(5000);
    assert(f.calls.length === 4 && JSON.parse(f.calls[3].options.body).baseVersion === 'v2', 'reviewed edits after resolution must resume autosave against latest version');
  }

  f = setup(); await tick(); f.input('manual first'); f.queue.push(response('manual first', 'v2')); f.save.fire('click'); await tick(); await f.advance(10000);
  assert(f.calls.length === 2, 'manual save cancels its queued autosave');
  f.input(''); f.queue.push(response('', 'v3')); await f.advance(5000);
  assert(JSON.parse(f.calls[2].options.body).text === '' && !f.values.has(key), 'intentional empty text autosaves safely');

  f = setup({ storageThrows: true }); await tick(); f.input('no backup'); f.queue.push(Promise.reject(new Error('offline'))); await f.advance(5000);
  assert(f.status.textContent.includes('editor.backupUnavailable') && f.status.textContent.includes('editor.autosavePaused') && !f.status.textContent.includes('editor.draftKept'), 'autosave failure truthfully reports unavailable backup and paused state');

  f = setup(); await tick(); f.input('hidden'); f.queue.push(response('hidden', 'v2')); document.visibilityState = 'hidden'; f.docEvents.visibilitychange(); await tick();
  assert(f.calls[1].options.keepalive && !f.values.has(key), 'hidden-page flush uses normal acknowledged keepalive path');

  f = setup(); await tick(); f.input('leaving while saving'); const active = deferred(); f.queue.push(active.promise); await f.advance(5000);
  f.input('newer leaving'); f.winEvents.pagehide(); assert(f.calls.length === 2 && JSON.parse(f.values.get(key)).text === 'newer leaving', 'pagehide during a save preserves newest draft without overlapping requests');
  active.resolve(response('leaving while saving', 'v2')); await tick();
  assert(JSON.parse(f.values.get(key)).baseVersion === 'v2', 'unflushed pagehide text retains acknowledged follow-up version');

  f = setup(); await tick(); f.input('signed out'); f.winEvents.storage({ key: 'mj-signout:account-a' }); await f.advance(10000);
  assert(f.calls.length === 1 && !f.values.has(key), 'sign-out cancels autosave without recreating drafts');

  f = setup(); await tick(); f.input('unsaved deletion'); f.text.fire('blur', { relatedTarget: f.trash }); await tick();
  assert(f.calls.length === 1, 'focusing delete must not autosave unsaved text before confirmation');
  f.trash.fire('pointerdown'); f.text.fire('blur', { relatedTarget: null }); await tick();
  assert(f.calls.length === 1, 'pointer activation must suppress blur saving even when browser does not focus buttons');
  f.winEvents.pointerup();
  window.confirm = () => false; f.trash.fire('click'); await tick();
  assert(f.calls.length === 1 && f.values.has(key) && f.text.value === 'unsaved deletion', 'cancelled delete preserves text and backup');
  f.values.set('mj-draft:v3:account-a:2019-01-01', 'other date');
  window.confirm = () => true; const deletePending = deferred(); f.queue.push(deletePending.promise); f.trash.fire('click'); await tick();
  assert(f.calls.length === 2 && f.calls[1].options.method === 'DELETE' && f.text.readOnly && f.save.disabled && f.trash.disabled, 'confirmed delete locks editor mutations');
  await f.advance(10000); assert(f.calls.length === 2 && f.values.has(key), 'delete cancels autosave while retaining backup until acknowledgement');
  deletePending.resolve({ ok: true, status: 204 }); await tick();
  assert(f.values.has('mj-draft:v3:account-a:2019-01-01') && f.text.value === '' && !f.values.has(key) && f.status.textContent === 'editor.deleted' && f.trash.hidden && !f.text.readOnly, 'confirmed trash clears only acknowledged text/backup and permits a fresh entry');
  f.input('recreated'); f.queue.push(response('recreated', 'new')); await f.advance(5000);
  assert(JSON.parse(f.calls[2].options.body).baseVersion === '' && !f.trash.hidden, 'writing after delete creates against empty version');

  for (const failure of [{ ok: false, status: 401 }, { ok: false, status: 503 }, { ok: false, status: 404 }, Promise.reject(new Error('offline'))]) {
    f = setup(); await tick(); f.input('retain after failed delete'); f.queue.push(failure); f.trash.fire('click'); await tick();
    assert(f.text.value === 'retain after failed delete' && f.values.has(key) && !f.save.disabled && !f.text.readOnly, 'failed delete keeps draft and unlocks editor');
    await f.advance(10000); assert(f.calls.length === 2, 'failed delete must not silently autosave instead');
    f.queue.push(response('retain after failed delete', 'v2')); f.save.fire('click'); await tick(); assert(!f.values.has(key), 'manual save can recover after failed delete');
  }
  f = setup(); await tick(); f.input('saving first'); const beforeDelete = deferred(); f.queue.push(beforeDelete.promise); f.save.fire('click'); f.trash.fire('click'); await tick();
  assert(f.calls.length === 2 && f.trash.disabled, 'delete cannot race with an active save'); beforeDelete.resolve(response('saving first', 'v2')); await tick();
  f = setup({ initial: { ok: true, status: 200, json: async () => ({ text: '', version: '', exists: false }) } }); await tick();
  assert(f.trash.hidden, 'missing entries do not offer a saved-entry delete action');

  f = setup(); await tick(); f.queue.push(Promise.reject(new Error('delete acknowledgement lost'))); f.trash.fire('click'); await tick();
  assert(f.values.has(key) && f.text.value === 'server', 'ambiguous delete must back up even previously clean text');
  f.queue.push(response('server', 'v2')); f.save.fire('click'); await tick(); assert(f.calls.length === 3 && !f.values.has(key), 'manual Save can recover previously clean text after an ambiguous delete');

  f = setup({ storageThrows: true }); await tick(); f.queue.push({ ok: true, status: 204 }); f.trash.fire('click'); await tick();
  assert(f.status.textContent === 'editor.deletedBackupRetained' && f.text.value === '', 'successful trash reports unavailable browser backup cleanup truthfully');

  for (const failure of [() => ({ ok: false, status: 503 }), () => ({ ok: false, status: 401 }), () => Promise.reject(new Error('delete acknowledgement lost'))]) {
    f = setup(); await tick(); f.input('acknowledged text'); f.queue.push(response('acknowledged text', 'v2')); f.save.fire('click'); await tick();
    assert(f.editor.classList.contains('is-saved'), 'successful save establishes green saved presentation');
    window.confirm = () => false; f.trash.fire('click'); await tick();
    assert(f.editor.classList.contains('is-saved') && f.calls.length === 2, 'cancelled deletion preserves acknowledged presentation');
    window.confirm = () => true; f.queue.push(failure()); f.trash.fire('click');
    assert(!f.editor.classList.contains('is-saved') && !f.editor.classList.contains('is-saving'), 'confirmed deletion clears prior saved presentation immediately');
    await tick(); let warned = false; f.winEvents.beforeunload({ preventDefault() { warned = true; } });
    assert(!f.editor.classList.contains('is-saved') && f.values.has(key) && warned && !f.save.disabled, 'uncertain delete stays neutral with recovery and manual retry');
    await f.advance(10000); assert(f.calls.length === 3, 'uncertain deletion must not autosave');
    f.queue.push(response('acknowledged text', 'v3')); f.save.fire('click'); await tick();
    assert(f.editor.classList.contains('is-saved') && !f.values.has(key), 'acknowledged manual recovery can restore saved presentation');
  }

  f = setup({ signOut: true, draft: { text: 'preserve on cancel', baseVersion: 'v1', versionKnown: true } }); f.queue.shift(); window.confirm = () => false; f.disconnect.fire('click'); await tick();
  assert(f.calls.length === 0 && f.values.has(key), 'cancelled disconnect leaves connection and drafts untouched');
  window.confirm = () => true; const disconnecting = deferred(); f.queue.push(disconnecting.promise); f.disconnect.fire('click'); await tick();
  assert(f.calls.length === 1 && f.calls[0].url === '/api/v1/disconnect' && f.button.disabled && f.disconnect.disabled && f.values.has(key), 'disconnect calls revoke endpoint and locks both actions without clearing backups');
  f.button.fire('click'); await tick(); assert(f.calls.length === 1, 'signout cannot overlap pending disconnect');
  disconnecting.resolve({ ok: false, status: 503 }); await tick(); assert(f.values.has(key) && !f.localValues.has('mj-signout:account-a') && !f.button.disabled && !f.disconnect.disabled && f.status.textContent === 'settings.disconnectFailed', 'failed disconnect preserves backups/notification state and permits retry');
  f.localValues.set('mj-theme', 'dark'); f.values.set('mj-draft:v3:account-b:2020-01-01', 'other account'); f.queue.push({ ok: true, status: 204 }); f.disconnect.fire('click'); await tick();
  assert(!f.values.has(key) && f.values.has('mj-draft:v3:account-b:2020-01-01') && f.localValues.has('mj-theme') && f.localValues.has('mj-signout:account-a') && window.location.href === '/', 'successful disconnect cleans current account and broadcasts only after acknowledgement');

  f = setup({ signOut: true, draft: { text: 'retry cleanup', baseVersion: 'v1', versionKnown: true } }); f.queue.shift(); f.queue.push({ ok: true, status: 204 });
  const removeAfterDisconnect = sessionStorage.removeItem; sessionStorage.removeItem = k => { if (k === key) throw new Error('cleanup denied'); removeAfterDisconnect(k); };
  f.disconnect.fire('click'); await tick(); assert(f.status.textContent === 'settings.signedOutCleanupFailed' && !f.disconnect.disabled && f.button.disabled, 'confirmed disconnect with cleanup failure permits only cleanup retry');
  sessionStorage.removeItem = removeAfterDisconnect; f.disconnect.fire('click'); await tick(); assert(f.calls.length === 1 && !f.values.has(key), 'cleanup retry never repeats revocation');
  f = setup(); await tick(); f.input('keep after expiry');
  f.queue.push(response('', '', 401)); f.save.fire('click'); await tick();
  assert(f.calls[1].options.headers['X-Myjournal-Account'] === 'account-a', 'protected editor requests carry their original account scope');
  assert(!f.auth.hidden && f.text.value === 'keep after expiry' && f.values.has(key) && window.location.href === '', '401 shows reconnect without navigating or deleting editor draft');
  const expiredCalls = f.calls.length;
  f.input('later typing after expiry'); await f.advance(10000); f.text.fire('blur'); f.winEvents.pagehide(); await tick();
  assert(f.calls.length === expiredCalls && !f.text.readOnly && JSON.parse(f.values.get(key)).text === 'later typing after expiry', '401 pauses idle/blur/pagehide writes and keeps later typing');
  f.queue.push(response('server', 'v1'), response('later typing after expiry', 'v2')); f.save.fire('click'); await tick();
  assert(f.auth.hidden && f.calls[expiredCalls].options.cache === 'no-store' && f.calls[expiredCalls + 1].options.method === 'PUT' && !f.values.has(key), 'manual retry after reconnect verifies fresh content before saving');
  f = setup({ draft: { text: 'recovered after expiry', baseVersion: 'v0', versionKnown: true }, initial: response('', '', 401) }); await tick();
  assert(!f.auth.hidden && f.text.value === 'recovered after expiry' && f.values.has(key) && !f.retry.hidden && f.save.disabled, 'initial 401 retains recovered text and requires valid load');
  f = setup(); await tick(); f.queue.push(response('', '', 401)); f.trash.fire('click'); await tick();
  assert(!f.auth.hidden && f.text.value === 'server' && f.values.has(key) && !localStorage.getItem('mj-signout:account-a'), 'DELETE 401 preserves text and does not announce logout');
  f = setup({ signOut: true, sessionValues: new Map([[key, 'recovery']]) }); f.queue.length = 0; f.queue.push(response('', '', 401)); f.disconnect.fire('click'); await tick();
  assert(!f.auth.hidden && f.values.get(key) === 'recovery' && window.location.href === '', 'disconnect 401 preserves session drafts and offers reconnect');
  f = setup({ storageThrows: true }); await tick(); f.input('copy this text'); f.queue.push(response('', '', 401)); f.save.fire('click'); await tick();
  assert(!f.auth.hidden && f.text.value === 'copy this text' && window.location.href === '' && f.status.textContent.includes('editor.backupUnavailable'), '401 with blocked storage retains text and truthful backup warning');
  f = setup({ draft: { text: 'mine', baseVersion: 'old', versionKnown: true }, initial: response('other', 'new') }); await tick(); f.queue.push(response('', '', 401)); f.mine.fire('click'); await tick();
  assert(!f.auth.hidden && f.text.value === 'mine' && f.values.has(key) && f.calls.length === 2, '401 during conflict resolution preserves mine without PUT');
  output('PASS editor interaction regressions');
}

run().catch((error) => { output(error.stack || String(error)); if (typeof quit === 'function') quit(1); else process.exitCode = 1; });

// Public pages omit app.js but must still clear only the signed-out account's drafts.
{
  const drafts = new Map([['mj-draft:v3:a:2026-10-01', 'a'], ['mj-draft:v3:b:2026-10-01', 'b'], ['mj-theme', 'dark']]);
  let cleanup;
  const originalWindow = globalThis.window;
  const originalStorage = globalThis.sessionStorage;
  globalThis.window = { addEventListener: (_, fn) => { cleanup = fn; } };
  globalThis.sessionStorage = { get length() { return drafts.size; }, key: i => [...drafts.keys()][i], removeItem: key => drafts.delete(key) };
  if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync('assets/static/js/draft-cleanup.js', 'utf8'));
  else load('assets/static/js/draft-cleanup.js');
  cleanup({ key: null });
  cleanup({ key: 'unrelated' });
  assert(drafts.size === 3, 'unrelated storage events must preserve drafts');
  cleanup({ key: 'mj-signout:a' });
  assert(!drafts.has('mj-draft:v3:a:2026-10-01') && drafts.has('mj-draft:v3:b:2026-10-01') && drafts.has('mj-theme'), 'public-page cleanup preserves other accounts and preferences');
  globalThis.sessionStorage = { get length() { throw new Error('blocked'); } };
  cleanup({ key: 'mj-signout:a' });
  globalThis.window = originalWindow;
  globalThis.sessionStorage = originalStorage;
}
