(function () {
  let messages = {};
  try {
    messages = JSON.parse(document.body.dataset.messages || '{}');
  } catch (error) {
    messages = {};
  }

  function t(key, values = {}) {
    return Object.entries(values).reduce(
      (text, [name, value]) => text.replaceAll(`{${name}}`, value),
      messages[key] || key,
    );
  }

  function formatDate(date) {
    const year = date.getUTCFullYear();
    const month = String(date.getUTCMonth() + 1).padStart(2, '0');
    const day = String(date.getUTCDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function addDay(dateString, delta) {
    const date = new Date(`${dateString}T00:00:00Z`);
    date.setUTCDate(date.getUTCDate() + delta);
    return formatDate(date);
  }

  function escapeHTML(value) {
    return String(value ?? '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function initEditor() {
    const editor = document.querySelector('[data-entry-editor]');
    if (!editor) return;

    const date = editor.dataset.entryDate;
    const textarea = editor.querySelector('[data-entry-text]');
    const status = editor.querySelector('[data-entry-status]');
    const saveBtn = editor.querySelector('[data-entry-save]');
    const prevBtn = editor.querySelector('[data-entry-prev]');
    const nextBtn = editor.querySelector('[data-entry-next]');

    if (!date || !textarea || !status || !saveBtn) return;

    const entryDate = new Date(`${date}T00:00:00Z`);
    const today = new Date();
    today.setUTCHours(0, 0, 0, 0);
    const isFutureDate = entryDate > today;
    const isTodayDate = entryDate.getTime() === today.getTime();

    if (isFutureDate) {
      saveBtn.disabled = true;
      nextBtn && (nextBtn.disabled = true);
      setStatus(t('editor.futureNotAllowed'));
      textarea.disabled = true;
      return;
    }

    if (isTodayDate && nextBtn) {
      nextBtn.disabled = true;
      nextBtn.setAttribute('aria-disabled', 'true');
    }

    let baseVersion = '';
    let lastSavedText = '';
    const draftKey = `mj-draft:${date}`;

    function setStatus(text) {
      status.textContent = text;
    }

    function setSaveState(state) {
      editor.classList.remove('is-saving', 'is-saved');
      if (state) {
        editor.classList.add(`is-${state}`);
      }
      if (state === 'saving') {
        saveBtn.textContent = t('editor.saving');
        saveBtn.disabled = true;
      } else {
        saveBtn.textContent = t('editor.save');
        saveBtn.disabled = false;
      }
    }

    function persistDraft() {
      if (!textarea.value.trim()) {
        localStorage.removeItem(draftKey);
        return;
      }
      localStorage.setItem(draftKey, textarea.value);
    }

    function clearDraft() {
      localStorage.removeItem(draftKey);
    }

    function isDirty() {
      return textarea.value !== lastSavedText;
    }

    function handleBeforeUnload(event) {
      if (isDirty()) {
        event.preventDefault();
        event.returnValue = '';
      }
    }

    async function loadEntry() {
      setStatus(t('editor.loading'));
      const response = await fetch(`/api/v1/entries/${date}`);
      if (!response.ok) {
        setStatus(t('editor.loadFailed'));
        return;
      }
      const entry = await response.json();
      const storedDraft = localStorage.getItem(draftKey);
      textarea.value = entry.text || storedDraft || '';
      lastSavedText = textarea.value;
      baseVersion = entry.version || '';
      if (entry.exists && entry.text) {
        clearDraft();
      } else if (storedDraft && !entry.text) {
        setStatus(t('editor.draftRestored'));
      } else {
        setStatus(entry.exists ? t('editor.saved') : t('editor.blank'));
      }
      if (textarea.value && isDirty()) {
        persistDraft();
      }
    }

    async function saveEntry() {
      const payload = { text: textarea.value };
      if (baseVersion) {
        payload.baseVersion = baseVersion;
      }

      setSaveState('saving');
      setStatus(t('editor.saving'));
      const response = await fetch(`/api/v1/entries/${date}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const message = response.status === 409 ? t('editor.conflict') : t('editor.saveFailed');
        persistDraft();
        setSaveState('');
        setStatus(`${message}${t('editor.draftKept')}`);
        return;
      }

      const entry = await response.json();
      baseVersion = entry.version || '';
      lastSavedText = textarea.value;
      clearDraft();
      setSaveState('saved');
      setStatus(entry.exists ? t('editor.saved') : t('editor.blank'));
    }

    textarea.addEventListener('input', () => {
      if (isDirty()) {
        setSaveState('');
        persistDraft();
        setStatus(t('editor.unsaved'));
      } else {
        clearDraft();
        setSaveState('');
      }
    });

    saveBtn.addEventListener('click', () => {
      saveEntry().catch(() => {
        persistDraft();
        setStatus(t('editor.saveFailedDraftKept'));
      });
    });

    window.addEventListener('beforeunload', handleBeforeUnload);

    prevBtn?.addEventListener('click', () => {
      window.location.href = `/entries/${addDay(date, -1)}`;
    });

    nextBtn?.addEventListener('click', () => {
      const nextDate = addDay(date, 1);
      const nextDateValue = new Date(`${nextDate}T00:00:00Z`);
      if (nextDateValue > today) {
        return;
      }
      window.location.href = `/entries/${nextDate}`;
    });

    loadEntry().catch(() => setStatus(t('editor.loadFailed')));
  }

  function initEntriesList() {
    const listRoot = document.querySelector('[data-entry-list]');
    if (!listRoot) return;

    fetch('/api/v1/entries?page=1&size=20')
      .then(async (response) => {
        if (!response.ok) {
          throw new Error(t('entries.unavailable'));
        }
        return response.json();
      })
      .then((data) => {
        listRoot.innerHTML = '';
        if (!data.items || data.items.length === 0) {
          const empty = document.createElement('p');
          empty.className = 'mj-empty-state';
          empty.textContent = t('entries.empty');
          listRoot.appendChild(empty);
          return;
        }

        const years = new Map();
        data.items.forEach((item) => {
          const date = new Date(`${item.date}T00:00:00Z`);
          const year = String(date.getUTCFullYear());
          if (!years.has(year)) {
            years.set(year, []);
          }
          years.get(year).push(item);
        });

        const yearsInOrder = [...years.keys()].sort((a, b) => Number(b) - Number(a));
        yearsInOrder.forEach((year) => {
          const yearSection = document.createElement('section');
          yearSection.className = 'mj-entry-year';

          const heading = document.createElement('h2');
          heading.textContent = year;
          yearSection.appendChild(heading);

          const items = years.get(year).sort((a, b) => b.date.localeCompare(a.date));
          items.forEach((item) => {
            const article = document.createElement('article');
            article.className = 'mj-entry-row';

            const time = document.createElement('time');
            time.dateTime = item.date;
            const date = new Date(`${item.date}T00:00:00Z`);
            time.textContent = date.toLocaleDateString(document.documentElement.lang, { weekday: 'short', day: '2-digit', month: 'short', timeZone: 'UTC' });

            const body = document.createElement('div');
            const title = document.createElement('h3');
            title.innerHTML = escapeHTML(item.preview || t('entries.previewUnavailable'));
            body.appendChild(title);

            const link = document.createElement('a');
            link.href = `/entries/${item.date}`;
            link.className = 'mj-entry-link';
            link.textContent = t('entries.open');
            body.appendChild(link);

            article.appendChild(time);
            article.appendChild(body);
            yearSection.appendChild(article);
          });

          listRoot.appendChild(yearSection);
        });
      })
      .catch(() => {
        const error = document.createElement('p');
        error.className = 'mj-empty-state';
        error.textContent = t('entries.unavailable');
        listRoot.replaceChildren(error);
      });
  }

  function currentMonth() {
    const today = new Date();
    return `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`;
  }

  function shiftMonth(month, delta) {
    const date = new Date(`${month}-01T00:00:00Z`);
    date.setUTCMonth(date.getUTCMonth() + delta);
    return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
  }

  function initCalendar() {
    const calendar = document.querySelector('[data-calendar]');
    if (!calendar) return;

    const monthLabel = calendar.querySelector('[data-calendar-month]');
    const grid = calendar.querySelector('[data-calendar-grid]');
    const prevBtn = calendar.querySelector('[data-calendar-prev]');
    const nextBtn = calendar.querySelector('[data-calendar-next]');
    const todayBtn = calendar.querySelector('[data-calendar-today]');
    let month = currentMonth();

    function renderDay(day) {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'mj-day';
      const date = day.date;
      const parsedDate = new Date(`${date}T00:00:00Z`);
      const label = parsedDate.toLocaleDateString(document.documentElement.lang, {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        timeZone: 'UTC',
      });

      if (!day.inMonth) {
        button.classList.add('is-muted');
        button.disabled = true;
        button.setAttribute('aria-label', t('calendar.outsideMonth'));
        return button;
      }

      button.textContent = String(parsedDate.getUTCDate());
      button.setAttribute('aria-label', t('calendar.dayLabel', {
        date: label,
        hasEntry: day.hasEntry ? t('calendar.hasEntry') : '',
        today: day.isToday ? t('calendar.isToday') : '',
        future: day.isFuture ? t('calendar.isFuture') : '',
      }));
      if (day.hasEntry) button.classList.add('has-entry');
      if (day.isToday) button.classList.add('is-today');
      if (day.isFuture) {
        button.classList.add('is-muted');
        button.disabled = true;
        return button;
      }
      button.addEventListener('click', () => {
        window.location.href = `/entries/${date}`;
      });
      return button;
    }

    async function loadCalendar() {
      monthLabel.textContent = t('calendar.loading');
      grid.innerHTML = '';
      const response = await fetch(`/api/v1/calendar?month=${month}`);
      if (!response.ok) throw new Error(t('calendar.unavailable'));
      const data = await response.json();
      monthLabel.textContent = new Date(Date.UTC(data.year, data.month - 1, 1)).toLocaleDateString(document.documentElement.lang, {
        month: 'long',
        year: 'numeric',
        timeZone: 'UTC',
      });

      Array.from({ length: 7 }, (_, offset) => {
        const weekday = new Date(Date.UTC(2024, 0, 1 + offset));
        return weekday.toLocaleDateString(document.documentElement.lang, { weekday: 'short', timeZone: 'UTC' });
      }).forEach((name) => {
        const heading = document.createElement('span');
        heading.textContent = name;
        grid.appendChild(heading);
      });
      data.weeks.flat().forEach((day) => grid.appendChild(renderDay(day)));
    }

    prevBtn.addEventListener('click', () => {
      month = shiftMonth(month, -1);
      loadCalendar().catch(() => { monthLabel.textContent = t('calendar.unavailable'); });
    });
    nextBtn.addEventListener('click', () => {
      month = shiftMonth(month, 1);
      loadCalendar().catch(() => { monthLabel.textContent = t('calendar.unavailable'); });
    });
    todayBtn.addEventListener('click', () => {
      month = currentMonth();
      loadCalendar().catch(() => { monthLabel.textContent = t('calendar.unavailable'); });
    });
    loadCalendar().catch(() => { monthLabel.textContent = t('calendar.unavailable'); });
  }

  document.addEventListener('DOMContentLoaded', () => {
    initEditor();
    initEntriesList();
    initCalendar();
  });
})();
