const output = typeof print === 'function' ? print : console.log;
const assert = (ok, message) => { if (!ok) throw new Error(message); };
function run(saved = null, blocked = false) {
  const attrs = {}; const events = {}; let systemChange; let dark = true; const writes = [];
  const media = { get matches() { return dark; }, addEventListener(name, fn) { systemChange = fn; } };
  globalThis.document = {
    documentElement: { setAttribute(k, v) { attrs[k] = v; }, getAttribute(k) { return attrs[k]; } },
    body: { dataset: { messages: JSON.stringify({ 'theme.switchToLight': 'Light', 'theme.switchToDark': 'Dark' }) } },
    querySelector() { return { setAttribute(k, v) { attrs[k] = v; }, addEventListener(k, fn) { events[k] = fn; } }; },
  };
  globalThis.localStorage = { getItem() { if (blocked) throw new Error('blocked'); return saved; }, setItem(k, v) { if (blocked) throw new Error('blocked'); writes.push(v); } };
  globalThis.window = { matchMedia() { return media; } };
  function loadScript(name) {
    if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync(`assets/static/js/${name}.js`, 'utf8'));
    else load(`assets/static/js/${name}.js`);
  }
  loadScript('theme-init'); const initial = attrs['data-theme']; loadScript('theme');
  return { attrs, initial, writes, click() { events.click(); }, change(value) { dark = value; if (systemChange) systemChange(media); } };
}
let f = run();
assert(f.initial === 'dark' && f.attrs['data-theme'] === 'dark', 'initial system dark theme must match bootstrap');
assert(f.writes.length === 0, 'system detection must not persist an explicit preference');
f.change(false); assert(f.attrs['data-theme'] === 'light' && f.attrs['aria-label'] === 'Dark', 'unselected theme follows device changes with accessible label');
f.click(); assert(f.writes[0] === 'dark' && f.attrs['aria-pressed'] === 'true', 'toggle persists explicit choice');
f.change(false); assert(f.attrs['data-theme'] === 'dark', 'device changes cannot override explicit choice');
f = run('light'); assert(f.initial === 'light' && f.attrs['data-theme'] === 'light', 'saved light overrides dark device');
f.change(true); assert(f.attrs['data-theme'] === 'light', 'saved preference stays fixed');
f = run('invalid'); assert(f.initial === 'dark' && f.attrs['data-theme'] === 'dark', 'invalid storage values fall back to device');
f = run(null, true); assert(f.initial === 'dark', 'blocked storage must still apply device theme before CSS');
f.change(false); assert(f.attrs['data-theme'] === 'light', 'blocked storage must not prevent device following');
f.click(); f.change(false); assert(f.attrs['data-theme'] === 'dark', 'manual choice remains effective in memory when storage is blocked');
output('PASS theme browser regressions');
