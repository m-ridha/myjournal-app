const output = typeof print === 'function' ? print : console.log;
const assert = (ok, message) => { if (!ok) throw new Error(message); };
function run({ path = '/', cookie = '', blocked = false, protocol = 'http:', editor = true } = {}) {
  let value = cookie; let written = ''; let ready; const redirects = [];
  globalThis.document = {
    get cookie() { return value; }, set cookie(v) { written = v; if (!blocked) value = v.split(';')[0]; },
    addEventListener(name, fn) { ready = fn; },
    querySelector() { return editor ? { dataset: { entryDate: '2026-10-08' } } : null; },
  };
  globalThis.window = { location: { pathname: path, protocol, href: `http://localhost${path}`, replace(v) { redirects.push(v); } } };
  globalThis.Intl = { DateTimeFormat() { return { resolvedOptions() { return { timeZone: 'Asia/Jakarta' }; } }; } };
  if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync('assets/static/js/timezone.js', 'utf8'));
  else load('assets/static/js/timezone.js');
  if (ready) ready();
  return { redirects, written, redirecting: window.mjDateRedirecting };
}
let f = run({ path: '/entries/2026-10-07' });
assert(f.redirects[0] === 'http://localhost/entries/2026-10-07', 'timezone correction must preserve selected date');
assert(f.redirecting, 'editor must pause during correction');
f = run({ cookie: 'mj_tz=Asia%2FJakarta' });
assert(f.redirects[0] === '/entries/2026-10-08', 'today URL must become a fixed selected date');
f = run({ blocked: true, path: '/calendar' });
assert(f.redirects.length === 0, 'blocked cookies must not cause reload loop');
f = run({ protocol: 'https:', path: '/settings' });
assert(f.written.includes('; Secure') && f.redirects.length === 0, 'HTTPS timezone cookie must be secure');
f = run({ cookie: 'mj_tz=Asia%2FJakarta', editor: false });
assert(f.redirects.length === 0, 'signed-out landing must not navigate to editor');
output('Timezone browser regressions passed');
