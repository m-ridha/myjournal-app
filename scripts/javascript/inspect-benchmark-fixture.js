/* Paste on /privacy, then run await mjInspectBenchmarkFixture().
   Read-only: two GETs; returns no text, versions, cookies or account values.
   Keep all editor/probe tabs idle. This never authorizes cleanup. */
window.mjInspectBenchmarkFixture = async function () {
  if (!['localhost', '127.0.0.1', '[::1]'].includes(location.hostname))
    throw new Error('Localhost only');
  if (document.querySelector('[data-entry-text]'))
    throw new Error('Run from Privacy to avoid editor autosave');
  const date = '2020-01-03';
  const expected = 'myjournal synthetic benchmark\n' + 'Synthetic text 🙂\n'.repeat(110);
  async function read() {
    const response = await fetch('/api/v1/entries/' + date, {cache: 'no-store'});
    if (!response.ok) throw new Error('Inspection HTTP ' + response.status);
    const entry = await response.json();
    if (typeof entry.exists !== 'boolean' || typeof entry.text !== 'string' ||
        typeof entry.version !== 'string' || (entry.exists && !entry.version))
      throw new Error('Invalid inspection response');
    return entry;
  }
  const first = await read();
  const second = await read();
  return {fixtureDate: date, readOnly: true, exists: second.exists,
    stableAcrossReads: first.exists === second.exists && first.version === second.version && first.text === second.text,
    matchesLastAttempt: second.exists && second.text === expected + 'B',
    matchesPreviousAcknowledged: second.exists && second.text === expected + 'A',
    matchesInitialFixture: second.exists && second.text === expected,
    cleanupPerformed: false};
};
