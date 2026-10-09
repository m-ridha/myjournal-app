// Apply the theme before loading CSS to avoid a flash of the wrong appearance.
(function () {
  let saved;
  try { saved = localStorage.getItem('mj-theme'); } catch (error) {}
  const preferred = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  document.documentElement.setAttribute('data-theme', saved === 'light' || saved === 'dark' ? saved : preferred);
})();
