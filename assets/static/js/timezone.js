(function () {
  // Runs before the editor is usable. A failed/blocked cookie leaves the
  // server's UTC fallback intact; do not reload endlessly when storage fails.
  function cookieValue() {
    return document.cookie.split(';').map((part) => part.trim())
      .find((part) => part.startsWith('mj_tz='))?.slice(6) || '';
  }
  try {
    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (zone) {
      const value = encodeURIComponent(zone);
      const old = cookieValue();
      document.cookie = `mj_tz=${value}; Path=/; SameSite=Lax; Max-Age=31536000${window.location.protocol === 'https:' ? '; Secure' : ''}`;
      const path = window.location.pathname;
      const usesDate = path === '/' || path === '/calendar' || path === '/entries/today' || /^\/entries\/\d{4}-\d{2}-\d{2}$/.test(path);
      if (old !== value && cookieValue() === value && usesDate) {
        window.mjDateRedirecting = true;
        // Keep explicit selected dates unchanged, including their draft keys.
        window.location.replace(window.location.href);
        return;
      }
    }
  } catch (error) {
    // The server continues using its resolved location (UTC if absent/invalid).
  }

  document.addEventListener('DOMContentLoaded', () => {
    if (window.location.pathname !== '/' && window.location.pathname !== '/entries/today') return;
    const editor = document.querySelector('[data-entry-editor]');
    if (!editor) return;
    window.mjDateRedirecting = true;
    // Canonical URL fixes the date once the user arrives; saving after midnight
    // never changes it. No editor request starts during this navigation.
    window.location.replace(`/entries/${editor.dataset.entryDate}`);
  });
})();
