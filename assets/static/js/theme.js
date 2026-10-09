(function () {
  const root = document.documentElement;
  const toggle = document.querySelector('[data-theme-toggle]');
  let messages = {};
  try {
    messages = JSON.parse(document.body.dataset.messages || '{}');
  } catch (error) {
    messages = {};
  }

  function t(key) {
    return messages[key] || key;
  }

  const media = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)');
  let explicit = false;

  function applyTheme(theme) {
    root.setAttribute('data-theme', theme);
    if (toggle) {
      const isDark = theme === 'dark';
      toggle.setAttribute('aria-label', isDark ? t('theme.switchToLight') : t('theme.switchToDark'));
      toggle.setAttribute('aria-pressed', String(isDark));
      toggle.title = isDark ? t('theme.switchToLight') : t('theme.switchToDark');
    }
  }

  function syncInitialTheme() {
    try {
      const savedTheme = localStorage.getItem('mj-theme');
      if (savedTheme === 'light' || savedTheme === 'dark') {
        explicit = true;
        applyTheme(savedTheme);
        return;
      }
    } catch (e) {}

    const preferred = media && media.matches ? 'dark' : 'light';
    applyTheme(preferred);
  }

  syncInitialTheme();
  if (media && media.addEventListener) {
    media.addEventListener('change', () => {
      if (!explicit) applyTheme(media.matches ? 'dark' : 'light');
    });
  }

  if (toggle) {
    toggle.addEventListener('click', function () {
      const nextTheme = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      explicit = true;
      applyTheme(nextTheme);
      try { localStorage.setItem('mj-theme', nextTheme); } catch (error) {}
    });
  }
})();
