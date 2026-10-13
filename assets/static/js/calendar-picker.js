(function () {
  window.mjInitMonthPicker = function (calendar, { month, max, text, onSelect }) {
    const trigger = calendar.querySelector('[data-month-trigger]');
    const panel = calendar.querySelector('[data-month-picker]');
    const heading = calendar.querySelector('[data-month-year]');
    const previous = calendar.querySelector('[data-month-prev]');
    const next = calendar.querySelector('[data-month-next]');
    const closeButton = calendar.querySelector('[data-month-close]');
    const items = calendar.querySelector('[data-month-options]');
    const maxYear = Number(max.slice(0, 4));
    const rangeSize = 12;
    let year = Number(month.slice(0, 4));
    let yearsView = false;
    let buttons = [];

    function close(restoreFocus = false) {
      panel.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
      if (restoreFocus) trigger.focus();
    }

    function focusItem(index) {
      buttons.forEach(button => { button.tabIndex = -1; });
      buttons[index].tabIndex = 0;
      buttons[index].focus();
    }

    function render(focus = true) {
      items.replaceChildren();
      buttons = [];
      const start = Math.floor((year - 1) / rangeSize) * rangeSize + 1;
      heading.textContent = yearsView ? `${start} – ${start + rangeSize - 1}` : String(year);
      heading.setAttribute('aria-label', text(yearsView ? 'calendar.showMonths' : 'calendar.chooseYear', { year }));
      previous.disabled = yearsView ? start === 1 : year === 1;
      next.disabled = yearsView ? start + rangeSize > maxYear : year >= maxYear;
      previous.setAttribute('aria-label', text(yearsView ? 'calendar.previousYears' : 'calendar.previousYear'));
      next.setAttribute('aria-label', text(yearsView ? 'calendar.nextYears' : 'calendar.nextYear'));
      items.setAttribute('aria-label', text(yearsView ? 'calendar.chooseYear' : 'calendar.chooseMonth'));
      let selectedIndex = -1;
      for (let i = 0; i < rangeSize; i += 1) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'mj-month-option';
        const value = yearsView ? start + i : `${String(year).padStart(4, '0')}-${String(i + 1).padStart(2, '0')}`;
        const selected = yearsView ? value === Number(month.slice(0, 4)) : value === month;
        button.disabled = yearsView ? value > maxYear : value > max;
        button.textContent = yearsView ? String(value) : new Date(`${value}-01T00:00:00Z`).toLocaleDateString(document.documentElement.lang, { month: 'short', timeZone: 'UTC' });
        button.setAttribute('aria-pressed', String(selected));
        if (selected && !button.disabled) selectedIndex = i;
        if (!yearsView) button.setAttribute('aria-label', new Date(`${value}-01T00:00:00Z`).toLocaleDateString(document.documentElement.lang, { month: 'long', year: 'numeric', timeZone: 'UTC' }));
        button.tabIndex = -1;
        button.addEventListener('click', () => {
          if (button.disabled) return;
          if (yearsView) {
            year = value;
            yearsView = false;
            render();
          } else {
            month = value;
            close(true);
            onSelect(value);
          }
        });
        buttons.push(button);
        items.appendChild(button);
      }
      const index = selectedIndex >= 0 ? selectedIndex : buttons.findIndex(button => !button.disabled);
      if (focus) focusItem(index);
      else buttons[index].tabIndex = 0;
    }

    trigger.addEventListener('click', () => {
      if (!panel.hidden) { close(); return; }
      year = Number(month.slice(0, 4));
      yearsView = false;
      panel.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
      render();
    });
    heading.addEventListener('click', () => { yearsView = !yearsView; render(); });
    previous.addEventListener('click', () => {
      if (previous.disabled) return;
      year = Math.max(1, year - (yearsView ? rangeSize : 1));
      render(false);
      if (previous.disabled) heading.focus();
    });
    next.addEventListener('click', () => {
      if (next.disabled) return;
      year = Math.min(maxYear, year + (yearsView ? rangeSize : 1));
      render(false);
      if (next.disabled) heading.focus();
    });
    closeButton.addEventListener('click', () => close(true));
    panel.addEventListener('keydown', event => {
      if (event.key === 'Escape') { event.preventDefault(); close(true); return; }
      const index = buttons.indexOf(document.activeElement);
      if (index < 0) return;
      const last = buttons.findLastIndex(button => !button.disabled);
      let target;
      switch (event.key) {
        case 'ArrowLeft': target = index - 1; break;
        case 'ArrowRight': target = index + 1; break;
        case 'ArrowUp': target = index - 3; break;
        case 'ArrowDown': target = index + 3; break;
        case 'Home': target = 0; break;
        case 'End': target = last; break;
        default: return;
      }
      event.preventDefault();
      if (target >= 0 && target < buttons.length && !buttons[target].disabled) focusItem(target);
    });
    panel.addEventListener('focusout', event => {
      if (event.relatedTarget) {
        if (!panel.contains(event.relatedTarget) && event.relatedTarget !== trigger) close();
      } else {
        // Rendering a new grid can briefly focus the document body.
        Promise.resolve().then(() => {
          if (!panel.contains(document.activeElement) && document.activeElement !== trigger) close();
        });
      }
    });
    document.addEventListener('click', event => {
      // A year selection replaces its own button before this event bubbles.
      const inside = event.composedPath?.().includes(panel) || panel.contains(event.target) || trigger.contains(event.target);
      if (!panel.hidden && !inside) close();
    });
    return { setMonth(value) { month = value; close(); } };
  };
})();
