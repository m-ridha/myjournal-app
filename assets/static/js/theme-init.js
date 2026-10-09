// Apply the theme before loading CSS to avoid a flash of the wrong appearance.
(function () {
  try {
    const saved = localStorage.getItem('mj-theme');
    const preferred = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', saved || preferred);
  } catch (error) {}
})();
