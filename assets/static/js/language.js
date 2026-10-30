(function () {
  document.querySelectorAll('[data-language-picker]').forEach(initPicker);
  document.querySelectorAll('[data-public-language]').forEach(picker => {
    picker.addEventListener('keydown', event => {
      if (event.key === 'Escape' && picker.open) {
        event.preventDefault();
        picker.open = false;
        picker.querySelector('summary').focus();
      }
    });
    document.addEventListener('click', event => {
      if (!picker.contains(event.target)) picker.open = false;
    });
    picker.addEventListener('focusout', event => {
      // Safari does not focus links on click, so a null relatedTarget does not
      // prove focus left the picker. Outside clicks still close it below.
      if (event.relatedTarget && !picker.contains(event.relatedTarget)) picker.open = false;
    });
  });

  function initPicker(picker) {
    const select = picker.querySelector('select');
    const custom = picker.querySelector('[data-language-custom]');
    const trigger = picker.querySelector('[data-language-trigger]');
    const label = picker.querySelector('[data-language-value]');
    const menu = picker.querySelector('[role="listbox"]');
    const autoSubmit = picker.hasAttribute('data-language-autosubmit');
    const form = picker.closest('form');
    const returnPath = picker.querySelector('[data-language-return]');
    if (returnPath) returnPath.value = window.location.pathname;
    const options = Array.from(picker.querySelectorAll('[data-language-option]'));

    function close(restoreFocus = false) {
      menu.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
      if (restoreFocus) trigger.focus();
    }

    function open(index = options.findIndex(option => option.dataset.languageOption === select.value)) {
      menu.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
      options[index].focus();
    }

    trigger.addEventListener('click', () => menu.hidden ? open() : close());
    options.forEach(option => option.addEventListener('click', () => {
      select.value = option.dataset.languageOption;
      label.textContent = autoSubmit ? select.value.toUpperCase() : option.textContent;
      label.lang = option.lang;
      options.forEach(item => item.setAttribute('aria-selected', String(item === option)));
      close(true);
      if (autoSubmit) form.requestSubmit();
    }));

    picker.addEventListener('keydown', event => {
      const index = options.indexOf(document.activeElement);
      switch (event.key) {
        case 'Escape':
          if (!menu.hidden) { event.preventDefault(); close(true); }
          return;
        case 'Tab': close(); return;
        case 'ArrowDown':
        case 'ArrowUp':
          event.preventDefault();
          if (menu.hidden) open();
          else open((index + (event.key === 'ArrowDown' ? 1 : options.length - 1)) % options.length);
          return;
        case 'Home': event.preventDefault(); open(0); return;
        case 'End': event.preventDefault(); open(options.length - 1); return;
        default:
          if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
            const match = options.findIndex(option => option.textContent.trim().toLowerCase().startsWith(event.key.toLowerCase()));
            if (match >= 0) { event.preventDefault(); open(match); }
          }
      }
    });
    document.addEventListener('click', event => { if (!picker.contains(event.target)) close(); });
    picker.addEventListener('focusout', event => {
      // Safari does not focus options on click, so a null relatedTarget does not
      // prove focus left the picker. Outside clicks still close it above.
      if (event.relatedTarget && !picker.contains(event.relatedTarget)) close();
    });
    const fallback = picker.querySelector('[data-language-fallback]');
    if (fallback) fallback.hidden = true;
    select.hidden = true;
    custom.hidden = false;
  }
})();
