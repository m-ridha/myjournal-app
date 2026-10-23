(function () {
  window.addEventListener('storage', (event) => {
    if (!event.key?.startsWith('mj-signout:')) return;
    const scope = event.key.slice('mj-signout:'.length);
    try {
      for (let i = sessionStorage.length - 1; i >= 0; i--) {
        const key = sessionStorage.key(i);
        if (key?.startsWith(`mj-draft:v3:${scope}:`)) sessionStorage.removeItem(key);
      }
    } catch (error) { /* Cookie sign-out still prevents writes to the server. */ }
  });
})();
