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

    // Use the request's resolved date; the timezone bootstrap reloads safely
    // before editing if the browser reports a changed IANA location.
    const today = document.body.dataset.today || formatDate(new Date());
    const isFutureDate = date > today;
    const isTodayDate = date === today;

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

    const scope = document.body.dataset.draftScope;
    if (!scope) {
      saveBtn.disabled = true;
      setStatus(t('editor.loadFailed'));
      return;
    }
    const draftKey = `mj-draft:v3:${scope}:${date}`;
    const legacyKey = `mj-draft:${date}`;
    const retryBtn = editor.querySelector('[data-entry-load-retry]');
    const legacyBtn = editor.querySelector('[data-entry-legacy-draft]');
    const conflictPanel = editor.querySelector('[data-entry-conflict]');
    const conflictMine = editor.querySelector('[data-conflict-mine]');
    const conflictOther = editor.querySelector('[data-conflict-other]');
    const conflictBoth = editor.querySelector('[data-conflict-both]');
    let conflicted = false;
    let resolving = false;
    let baseVersion = '';
    let lastSavedText = '';
    let loaded = false;
    let loading = false;
    let saving = false;
    let edited = false;
    let signedOut = false;
    let unverifiedDraft = false;
    let backupAvailable = true;
    let legacyDraft = null;

    function setStatus(text) {
      status.textContent = text;
    }

    function storage(action) {
      try {
        const result = action(sessionStorage);
        backupAvailable = true;
        return result;
      } catch (error) {
        backupAvailable = false;
        return null;
      }
    }

    function notice(text) {
      setStatus(text + (backupAvailable ? '' : t('editor.backupUnavailable')));
    }

    function refreshControls() {
      textarea.readOnly = !loaded || signedOut || resolving;
      textarea.setAttribute('aria-busy', loading || resolving ? 'true' : 'false');
      saveBtn.disabled = !loaded || saving || signedOut || resolving;
      saveBtn.textContent = t(saving ? 'editor.saving' : 'editor.save');
      if (conflictPanel) conflictPanel.hidden = !conflicted || signedOut;
      [conflictMine, conflictOther, conflictBoth].forEach((button) => { if (button) button.disabled = saving || resolving || !loaded || signedOut; });
      if (legacyBtn) legacyBtn.disabled = !loaded || saving || isDirty();
    }

    function setSaveState(state) {
      editor.classList.remove('is-saving', 'is-saved');
      if (state) editor.classList.add(`is-${state}`);
      refreshControls();
    }

    function persistDraft() {
      if (signedOut) return false;
      const result = storage((store) => {
        if (isDirty()) {
          store.setItem(draftKey, JSON.stringify({ text: textarea.value, baseVersion, versionKnown: loaded || draft?.versionKnown === true }));
        } else {
          store.removeItem(draftKey);
        }
        return true;
      });
      return result === true;
    }

    function isDirty() {
      return saving || unverifiedDraft || textarea.value !== lastSavedText;
    }

    const rawDraft = storage((store) => store.getItem(draftKey));
    let draft = null;
    try {
      const parsed = rawDraft === null ? null : JSON.parse(rawDraft);
      if (parsed && typeof parsed.text === 'string' && typeof parsed.baseVersion === 'string') draft = parsed;
    } catch (error) {
      // Preserve malformed backups; never remove them during a failed recovery.
    }
    if (draft) {
      textarea.value = draft.text;
      baseVersion = draft.baseVersion;
      unverifiedDraft = true;
    }
    // Old local backups remain available for explicit recovery. New drafts
    // live in sessionStorage, isolated by browser tab and preserved on reload.
    let legacySource = legacyKey;
    let legacyVersion = '';
    let legacyVersionKnown = false;
    try {
      const previousKey = `mj-draft:v2:${scope}:${date}`;
      const previous = localStorage.getItem(previousKey);
      if (previous !== null) {
        const parsed = JSON.parse(previous);
        if (typeof parsed.text === 'string' && typeof parsed.baseVersion === 'string') {
          legacyDraft = parsed.text;
          legacySource = previousKey;
          legacyVersion = parsed.baseVersion;
          legacyVersionKnown = parsed.versionKnown === true;
        }
      }
      if (legacyDraft === null) legacyDraft = localStorage.getItem(legacyKey);
    } catch (error) { /* Leave old backups intact if unavailable or malformed. */ }
    if (legacyBtn) legacyBtn.hidden = legacyDraft === null || draft !== null;
    refreshControls();

    async function loadEntry() {
      if (loading || signedOut) return;
      loading = true;
      refreshControls();
      if (retryBtn) retryBtn.hidden = true;
      notice(t('editor.loading'));
      try {
        const response = await fetch(`/api/v1/entries/${date}`);
        if (!response.ok) throw new Error('load failed');
        const entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid entry');
        if (signedOut) return;
        lastSavedText = entry.text;
        // Never overwrite input typed while loading. A recovered backup keeps
        // its original base version so changed server content yields a conflict.
        if (!edited && !draft) textarea.value = entry.text;
        if (!draft || draft.versionKnown !== true || textarea.value === entry.text) baseVersion = entry.version;
        loaded = true;
        conflicted = !!draft && draft.versionKnown === true && baseVersion !== entry.version && textarea.value !== entry.text;
        unverifiedDraft = false;
        persistDraft();
        setSaveState('');
        notice(t(conflicted ? 'editor.conflict' : isDirty() ? (draft && !edited ? 'editor.draftRestored' : 'editor.unsaved') : (entry.exists ? 'editor.saved' : 'editor.blank')));
      } catch (error) {
        if (retryBtn) retryBtn.hidden = false;
        notice(t(isDirty() ? 'editor.loadFailedDraft' : 'editor.loadFailed'));
      } finally {
        loading = false;
        refreshControls();
      }
    }

    async function saveEntry() {
      if (!loaded || saving || signedOut || resolving) return;
      const sentText = textarea.value;
      const payload = { text: sentText, baseVersion };
      persistDraft();
      saving = true;
      setSaveState('saving');
      notice(t('editor.saving'));
      try {
        const response = await fetch(`/api/v1/entries/${date}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        if (!response.ok) {
          unverifiedDraft = true;
          conflicted = response.status === 409;
          const key = response.status === 409 ? 'editor.conflict'
            : response.status === 401 ? 'editor.sessionExpired'
            : response.status === 413 ? 'editor.saveTooLarge' : 'editor.saveFailed';
          const kept = persistDraft();
          setSaveState('');
          notice(t(key) + (kept ? t('editor.draftKept') : ''));
          return;
        }
        const entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid save response');
        if (signedOut) return;
        saving = false;
        conflicted = false;
        unverifiedDraft = false;
        baseVersion = entry.version;
        lastSavedText = entry.text;
        // Only normalize acknowledged text when the user has not typed since
        // the request snapshot. Later text remains dirty and backed up.
        if (textarea.value === sentText) textarea.value = entry.text;
        persistDraft();
        setSaveState(isDirty() ? '' : 'saved');
        notice(t(isDirty() ? 'editor.unsaved' : (entry.exists ? 'editor.saved' : 'editor.blank')));
      } catch (error) {
        unverifiedDraft = true;
        const kept = persistDraft();
        setSaveState('');
        notice(t('editor.saveFailed') + (kept ? t('editor.draftKept') : ''));
      } finally {
        saving = false;
        refreshControls();
      }
    }

    async function resolveConflict(choice) {
      if (!conflicted || saving || resolving || signedOut) return;
      if (choice === 'other' && !window.confirm(t('editor.loadOtherConfirm'))) return;
      resolving = true;
      refreshControls();
      notice(t('editor.loading'));
      let shouldSave = false;
      try {
        const response = await fetch(`/api/v1/entries/${date}`);
        if (!response.ok) throw new Error('load failed');
        const entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid entry');
        if (signedOut) return;
        const mine = textarea.value;
        lastSavedText = entry.text;
        baseVersion = entry.version;
        // Rebase explicitly on the just-fetched version. A subsequent writer
        // still causes a normal conflict; no unconditional overwrite is used.
        if (choice === 'other') textarea.value = entry.text;
        if (choice === 'both') textarea.value = mine + (mine && entry.text ? '\n\n' : '') + entry.text;
        draft = null;
        edited = choice !== 'other';
        unverifiedDraft = false;
        conflicted = false;
        persistDraft();
        setSaveState('');
        notice(t(isDirty() ? 'editor.unsaved' : entry.exists ? 'editor.saved' : 'editor.blank'));
        shouldSave = choice === 'mine';
      } catch (error) {
        const kept = persistDraft();
        notice(t('editor.resolveFailed') + (kept ? t('editor.draftKept') : ''));
      } finally {
        resolving = false;
        refreshControls();
      }
      if (shouldSave && !signedOut) await saveEntry();
    }
    conflictMine?.addEventListener('click', () => resolveConflict('mine'));
    conflictOther?.addEventListener('click', () => resolveConflict('other'));
    conflictBoth?.addEventListener('click', () => resolveConflict('both'));

    textarea.addEventListener('input', () => {
      if (signedOut) {
        setStatus(t('editor.sessionExpired'));
        return;
      }
      edited = true;
      if (!loaded) unverifiedDraft = true;
      persistDraft();
      setSaveState(saving ? 'saving' : '');
      notice(t(saving ? 'editor.saving' : isDirty() ? 'editor.unsaved' : 'editor.saved'));
    });
    saveBtn.addEventListener('click', saveEntry);
    retryBtn?.addEventListener('click', loadEntry);
    legacyBtn?.addEventListener('click', () => {
      if (!loaded || saving || isDirty() || legacyDraft === null) return;
      if (!window.confirm(t('editor.legacyDraftConfirm'))) return;
      textarea.value = legacyDraft;
      edited = true;
      if (legacyVersionKnown) { baseVersion = legacyVersion; draft = { versionKnown: true }; }
      if (persistDraft()) {
        try { localStorage.removeItem(legacySource); } catch (error) { /* Safe to retain the old copy. */ }
      }
      legacyBtn.hidden = true;
      setSaveState('');
      notice(t('editor.draftRestored'));
    });

    window.addEventListener('beforeunload', (event) => {
      if (isDirty() && !signedOut) {
        persistDraft();
        event.preventDefault();
        event.returnValue = '';
      }
    });
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden' && isDirty()) persistDraft();
    });
    window.addEventListener('storage', (event) => {
      if (event.key === `mj-signout:${scope}`) {
        signedOut = true;
        try {
          for (let i = sessionStorage.length - 1; i >= 0; i--) {
            const key = sessionStorage.key(i);
            if (key?.startsWith(`mj-draft:v3:${scope}:`)) sessionStorage.removeItem(key);
          }
        } catch (error) { /* Stop future writes even if cleanup is unavailable. */ }
        setSaveState('');
        setStatus(t('editor.sessionExpired'));
      }
    });

    prevBtn?.addEventListener('click', () => {
      window.location.href = `/entries/${addDay(date, -1)}`;
    });
    nextBtn?.addEventListener('click', () => {
      if (nextBtn.disabled) return;
      const nextDate = addDay(date, 1);
      if (nextDate > today) return;
      window.location.href = `/entries/${nextDate}`;
    });
    loadEntry();
  }

  function initTabDraftCleanup() {
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
  }

  function initSignOut() {
    const button = document.querySelector('[data-sign-out]');
    const status = document.querySelector('[data-sign-out-status]');
    const scope = document.body.dataset.draftScope;
    if (!button || !status || !scope) return;
    let loggedOut = false;
    button.addEventListener('click', async () => {
      if (button.disabled) return;
      button.disabled = true;
      if (!loggedOut) {
        try {
          // Check storage access without deleting drafts or announcing logout.
          // A failed request must leave every tab's recovery path intact.
          const probe = `mj-cleanup-probe:${scope}:${Date.now()}:${Math.random()}`;
          [sessionStorage, localStorage].forEach((store) => {
            store.setItem(probe, '1');
            store.removeItem(probe);
          });
        } catch (error) {
          status.textContent = t('settings.draftCleanupFailed');
          button.disabled = false;
          return;
        }
        try {
          const response = await fetch('/api/v1/logout', { method: 'POST' });
          if (!response.ok) throw new Error('sign out failed');
          loggedOut = true;
        } catch (error) {
          status.textContent = t('settings.signOutFailed');
          button.disabled = false;
          return;
        }
      }
      try {
        // Only confirmed logout makes cleanup final. Broadcast before clearing
        // so other editors stop recreating drafts, including pending saves.
        localStorage.setItem(`mj-signout:${scope}`, `${Date.now()}:${Math.random()}`);
        for (let i = sessionStorage.length - 1; i >= 0; i--) {
          const key = sessionStorage.key(i);
          if (key?.startsWith(`mj-draft:v3:${scope}:`)) sessionStorage.removeItem(key);
        }
        const keys = [];
        for (let i = 0; i < localStorage.length; i += 1) keys.push(localStorage.key(i));
        keys.forEach((key) => {
          if (key?.startsWith(`mj-draft:v2:${scope}:`) || /^mj-draft:\d{4}-\d{2}-\d{2}$/.test(key)) localStorage.removeItem(key);
        });
        window.location.href = '/';
      } catch (error) {
        status.textContent = t('settings.signedOutCleanupFailed');
        button.textContent = t('settings.retryDraftCleanup');
        button.disabled = false;
      }
    });
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
    return (document.body.dataset.today || formatDate(new Date())).slice(0, 7);
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
    if (window.mjDateRedirecting) return;
    initTabDraftCleanup();
    initEditor();
    initSignOut();
    initEntriesList();
    initCalendar();
  });
})();
