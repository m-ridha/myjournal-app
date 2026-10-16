const output = typeof print === 'function' ? print : console.log;
const assert = (ok, message) => { if (!ok) throw new Error(message); };
globalThis.window = {};
globalThis.location = { hostname: 'localhost' };
globalThis.document = { querySelector() { return null; } };
globalThis.performance = { now() { return 0; } };
globalThis.setTimeout = fn => fn();
globalThis.TextEncoder = class { encode(text) { return { length: unescape(encodeURIComponent(text)).length }; } };
if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync('scripts/javascript/benchmark-browser.js', 'utf8'));
else load('scripts/javascript/benchmark-browser.js');
function fixture(existing, failSave = false) {
  let entry = { exists: existing, text: existing ? 'private existing text' : '', version: existing ? 'old' : '' };
  let writes = 0; let deletes = 0;
  globalThis.fetch = async (path, init = {}) => {
    let status = 200; let body;
    if (path === '/api/v1/settings') body = { provider: 'Google' };
    else if (init.method === 'PUT') {
      writes++;
      if (failSave) { status = 503; body = {}; }
      else { const saved = JSON.parse(init.body); entry = { exists: true, text: saved.text, version: 'v' + writes }; body = entry; }
    } else if (init.method === 'DELETE') { deletes++; entry = { exists: false, text: '', version: '' }; status = 204; }
    else if (path.includes('?')) body = {};
    else body = entry;
    const serialized = body ? JSON.stringify(body) : '';
    return { ok: status < 400, status, async json() { return JSON.parse(serialized); }, async text() { return serialized; } };
  };
  return { get writes() { return writes; }, get deletes() { return deletes; } };
}
(async () => {
  let release;
  globalThis.fetch = () => new Promise(resolve => { release = () => resolve({ ok: false }); });
  const pending = window.mjBenchmark('2020-01-01').catch(() => { });
  let blocked = false;
  try { await window.mjBenchmark('2020-01-02'); } catch (error) { blocked = error.message.includes('already running'); }
  assert(blocked, 'overlapping probes must be rejected');
  release(); await pending;
  assert(!window.mjBenchmarkRunning, 'setup failures must release the run guard');
  let f = fixture(true);
  let result = await window.mjBenchmark('2020-01-01');
  assert(result.failed && f.writes === 0 && f.deletes === 0, 'existing entries must remain untouched');
  assert(!JSON.stringify(result).includes('private existing text'), 'reports must not expose text');
  f = fixture(false, true);
  result = await window.mjBenchmark('2020-01-01');
  assert(result.failed && f.writes === 1 && f.deletes === 0, 'ambiguous failed save must stop without retry or deletion');
  f = fixture(false);
  result = await window.mjBenchmark('2020-01-01');
  assert(result.cleanupVerified && f.deletes === 1 && result.samples.length === 60, 'successful probe must measure all scenarios and verify trash');
  assert(!JSON.stringify(result).includes('Synthetic text'), 'reports must contain no fixture bodies');
  f = fixture(false);
  result = await window.mjBenchmark('2020-01-02', { scenarios: ['changed-save', 'entries-repeat', 'entries-after-save'] });
  assert(result.cleanupVerified && result.samples.length === 30 && result.summary['changed-save'].n === 10, 'targeted comparison must run only selected scenarios');
  if (typeof require === 'function') (0, eval)(require('node:fs').readFileSync('scripts/javascript/inspect-benchmark-fixture.js', 'utf8'));
  else load('scripts/javascript/inspect-benchmark-fixture.js');
  let reads = 0;
  const synthetic = 'myjournal synthetic benchmark\n' + 'Synthetic text 🙂\n'.repeat(110);
  globalThis.fetch = async (path, init) => {
    assert(path === '/api/v1/entries/2020-01-03' && !init.method, 'inspection must only GET retained date');
    reads++;
    return { ok: true, async json() { return { exists: true, text: synthetic + 'B', version: 'private-version' }; } };
  };
  const inspected = await window.mjInspectBenchmarkFixture();
  assert(reads === 2 && inspected.matchesLastAttempt && inspected.stableAcrossReads && !inspected.cleanupPerformed, 'inspection must recognize attempted content without cleanup');
  assert(!JSON.stringify(inspected).includes('Synthetic text') && !JSON.stringify(inspected).includes('private-version'), 'inspection must sanitize its result');
  globalThis.fetch = async () => ({ ok: true, async json() { return { exists: true, text: 'unrelated private content', version: String(reads++) }; } });
  const changed = await window.mjInspectBenchmarkFixture();
  assert(!changed.stableAcrossReads && !changed.matchesLastAttempt && !changed.matchesPreviousAcknowledged, 'inspection must not classify unrelated or changing content as the fixture');
  output('PASS benchmark browser regressions');
})().catch(error => { output(error.stack || String(error)); if (typeof process !== 'undefined') process.exitCode = 1; else throw error; });
