/* Paste into DevTools on the signed-in localhost app, then call:
   await mjBenchmark('2020-01-01')
   Uses one empty past date, stops on failure, trashes its fixture on success.
   Returns timings/status only; never returns text, versions, cookies or claims. */
window.mjBenchmark = async function (date, options = {}) {
  if (window.mjBenchmarkRunning) throw new Error('Benchmark already running; wait for its result');
  window.mjBenchmarkRunning = true;
  const progress = stage => {
    if (typeof console !== 'undefined' && typeof console.info === 'function')
      console.info('[myjournal benchmark]', date, stage);
  };
  try {
    progress('starting');
    if (!['localhost', '127.0.0.1', '[::1]'].includes(location.hostname))
      throw new Error('Localhost only');
    if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) throw new Error('Use YYYY-MM-DD');
    if (document.querySelector('[data-entry-text]'))
      throw new Error('Run from Privacy to avoid editor autosave');
    const allowedScenarios = ['open', 'changed-save', 'unchanged-save', 'entries-repeat', 'calendar-repeat', 'entries-after-save'];
    const scenarios = options.scenarios === undefined ? allowedScenarios : options.scenarios;
    if (!Array.isArray(scenarios) || scenarios.length === 0 ||
        new Set(scenarios).size !== scenarios.length || scenarios.some(name => !allowedScenarios.includes(name)))
      throw new Error('Choose distinct supported benchmark scenarios');
    const settings = await fetch('/api/v1/settings');
    if (!settings.ok || (await settings.json()).provider !== 'Google')
      throw new Error('Google session required');
    const path = '/api/v1/entries/' + date;
    const marker = 'myjournal synthetic benchmark\n';
    const text = marker + 'Synthetic text 🙂\n'.repeat(110);
    const samples = [];
    let current;
    async function request(scenario, endpoint, init, timed = true) {
      const start = performance.now();
      const response = await fetch(endpoint, init);
      const body = await response.text();
      if (timed) samples.push({scenario, status: response.status,
        elapsedMs: performance.now() - start});
      if (!response.ok) throw new Error('HTTP ' + response.status + ' during ' + scenario);
      return body ? JSON.parse(body) : null;
    }
    async function save(content, timed = true) {
      const next = await request(content === current.text ? 'unchanged-save' : 'changed-save', path, {
        method: 'PUT', headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({text: content, baseVersion: current.version || ''})
      }, timed);
      if (!next.exists || next.text !== content || !next.version)
        throw new Error('Invalid save acknowledgement');
      current = next;
    }
    const pause = () => new Promise(resolve => setTimeout(resolve, 300));
    try {
      current = await request('preflight', path, undefined, false);
      if (current.exists) throw new Error('Choose an empty past date; existing entry untouched');
      await save(text, false);
      // Prime auth/connection paths, not measured; browsing caches may already exist.
      const opened = await request('prime', path, undefined, false);
      if (opened.text !== current.text || opened.version !== current.version)
        throw new Error('Fixture mismatch');
      for (const scenario of scenarios) {
        progress(scenario);
        for (let i = 0; i < 10; i++) {
          if (scenario === 'open') {
            const entry = await request(scenario, path);
            if (entry.text !== current.text || entry.version !== current.version) throw new Error('Fixture changed');
          } else if (scenario === 'changed-save') await save(text + (i % 2 ? 'B' : 'A'));
          else if (scenario === 'unchanged-save') await save(current.text);
          else if (scenario === 'entries-repeat') await request(scenario, '/api/v1/entries?page=1&size=20');
          else if (scenario === 'calendar-repeat') await request(scenario, '/api/v1/calendar?month=' + date.slice(0, 7));
          else {
            await save(text + (i % 2 ? 'D' : 'C'), false);
            await request(scenario, '/api/v1/entries?page=1&size=20');
          }
          await pause();
        }
      }
      progress('cleanup');
      const check = await request('cleanup-check', path, undefined, false);
      if (check.text !== current.text || check.version !== current.version) throw new Error('Fixture changed before cleanup');
      await request('cleanup', path, {method: 'DELETE'}, false);
      const removed = await request('cleanup-verify', path, undefined, false);
      if (removed.exists) throw new Error('Cleanup not verified');
      const summary = {};
      for (const scenario of new Set(samples.map(s => s.scenario))) {
        const values = samples.filter(s => s.scenario === scenario).map(s => s.elapsedMs).sort((a,b) => a-b);
        summary[scenario] = {n: values.length, medianMs: (values[4] + values[5]) / 2,
          minMs: values[0], maxMs: values[values.length - 1]};
      }
      progress('completed');
      return {mode: 'exploratory, token primed; browsing cache state unverified', date,
        fixtureBytes: new TextEncoder().encode(text).length, cleanupVerified: true,
        samples, summary};
    } catch (error) {
      progress('stopped; inspect returned result before retrying');
      return {failed: true, reason: error.message, fixtureDate: date,
        cleanupVerified: false, samples};
    }
  } finally {
    window.mjBenchmarkRunning = false;
  }
};
