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

  let authenticationExpired = false;
  let authenticationGeneration = 0;
  let protectEditorOnExpiry = () => { };

  async function protectedFetch(url, options, isCurrent = () => true) {
    const generation = authenticationGeneration;
    const retrying = authenticationExpired;
    const scope = document.body.dataset.draftScope;
    if (scope) options = { ...options, headers: { ...options?.headers, 'X-Myjournal-Account': scope } };
    const response = await fetch(url, options);
    if (!isCurrent()) return response;
    const banner = document.querySelector('[data-auth-expired]');
    if (response.status === 401) {
      const firstExpiry = !authenticationExpired;
      authenticationExpired = true;
      authenticationGeneration += 1;
      if (banner) banner.hidden = false;
      const message = document.querySelector('[data-auth-message]');
      if (message) message.textContent = t('auth.expired');
      if (firstExpiry) banner?.scrollIntoView?.({ block: 'start' });
      protectEditorOnExpiry();
    } else if (response.ok && retrying && generation === authenticationGeneration) {
      authenticationExpired = false;
      if (banner) banner.hidden = true;
    }
    return response;
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

  function initEditor() {
    const editor = document.querySelector('[data-entry-editor]');
    if (!editor) return;

    const date = editor.dataset.entryDate;
    const textarea = editor.querySelector('[data-entry-text]');
    const status = editor.querySelector('[data-entry-status]');
    const saveBtn = editor.querySelector('[data-entry-save]');
    const deleteBtn = editor.querySelector('[data-entry-delete]');
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
    const legacyKey = `mj-draft:v2:${scope}:${date}`;
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
    let deleting = false;
    let exists = false;
    let explicitAction = false;
    let edited = false;
    let signedOut = false;
    let unverifiedDraft = false;
    let saveNeedsVerification = false;
    let backupAvailable = true;
    let legacyDraft = null;
    let autosaveTimer = null;
    let autosaveReady = false;
    let autosavePaused = false;

    protectEditorOnExpiry = () => {
      cancelAutosave();
      autosavePaused = true;
      saveNeedsVerification = true;
      if (loaded || isDirty()) {
        unverifiedDraft = true;
        persistDraft();
      }
    };

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
      textarea.readOnly = !loaded || signedOut || resolving || deleting;
      textarea.setAttribute('aria-busy', loading || resolving || deleting ? 'true' : 'false');
      saveBtn.disabled = !loaded || saving || signedOut || resolving || deleting;
      saveBtn.textContent = t(saving ? 'editor.saving' : 'editor.save');
      if (deleteBtn) {
        deleteBtn.hidden = !exists;
        deleteBtn.disabled = !loaded || saving || deleting || signedOut || resolving;
        const label = t(deleting ? 'editor.deleting' : 'editor.delete');
        deleteBtn.setAttribute('aria-label', label);
        deleteBtn.setAttribute('title', label);
      }
      if (conflictPanel) conflictPanel.hidden = !conflicted || signedOut;
      [conflictMine, conflictOther, conflictBoth].forEach((button) => { if (button) button.disabled = saving || resolving || deleting || !loaded || signedOut; });
      if (legacyBtn) legacyBtn.disabled = !loaded || saving || deleting || signedOut || isDirty();
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

    function cancelAutosave() {
      clearTimeout(autosaveTimer);
      autosaveTimer = null;
    }

    function canAutosave() {
      return loaded && autosaveReady && !autosavePaused && !conflicted && !resolving && !deleting && !signedOut;
    }

    function scheduleAutosave() {
      cancelAutosave();
      if (canAutosave() && !saving && isDirty()) {
        autosaveTimer = setTimeout(() => saveEntry({ automatic: true }), 5000);
      }
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
    // Only account-scoped v2 backups can be recovered. Date-only backups
    // have no ownership information and must never enter the editor.
    let legacyVersion = '';
    let legacyVersionKnown = false;
    try {
      const previous = localStorage.getItem(legacyKey);
      if (previous !== null) {
        const parsed = JSON.parse(previous);
        if (typeof parsed.text === 'string' && typeof parsed.baseVersion === 'string') {
          legacyDraft = parsed.text;
          legacyVersion = parsed.baseVersion;
          legacyVersionKnown = parsed.versionKnown === true;
        }
      }
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
        const recoveryRead = draft !== null || legacyDraft !== null || edited;
        const response = await protectedFetch(`/api/v1/entries/${date}`, recoveryRead
          ? { cache: 'no-store', headers: { 'Cache-Control': 'no-store' } }
          : undefined);
        if (!response.ok) throw new Error('load failed');
        const entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid entry');
        if (signedOut) return;
        exists = entry.exists;
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

    async function saveEntry({ automatic = false, keepalive = false } = {}) {
      if (!loaded || saving || deleting || signedOut || resolving || !isDirty()) return;
      if (automatic && !canAutosave()) return;
      cancelAutosave();
      const sentText = textarea.value;
      const payload = { text: sentText, baseVersion };
      persistDraft();
      saving = true;
      setSaveState('saving');
      notice(t('editor.saving'));
      try {
        let response;
        let entry;
        // A failed acknowledgement can hide a successful upload. Only an explicit
        // retry reads first; differing text never silently adopts a new version.
        if (!automatic && saveNeedsVerification) {
          response = await protectedFetch(`/api/v1/entries/${date}`, { cache: 'no-store', headers: { 'Cache-Control': 'no-store' } });
          if (response.ok) {
            const current = await response.json();
            if (typeof current.text !== 'string' || typeof current.version !== 'string' ||
              typeof current.exists !== 'boolean' || (current.exists && !current.version) ||
              (!current.exists && (current.text !== '' || current.version !== '')))
              throw new Error('invalid verification response');
            if (signedOut) return;
            if (current.text === sentText) entry = current;
            else response = null;
          }
        }
        if (!response) {
          response = await protectedFetch(`/api/v1/entries/${date}`, {
            method: 'PUT',
            keepalive,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          });
        }
        if (signedOut) return;
        if (!response.ok) {
          autosavePaused = true;
          unverifiedDraft = true;
          conflicted = response.status === 409;
          saveNeedsVerification = !conflicted;
          const key = response.status === 409 ? 'editor.conflict'
            : response.status === 401 ? 'editor.sessionExpired'
              : response.status === 413 ? 'editor.saveTooLarge' : 'editor.saveFailed';
          const kept = persistDraft();
          setSaveState('');
          notice(t(key) + (kept ? t('editor.draftKept') : '') + (automatic && !conflicted ? t('editor.autosavePaused') : ''));
          return;
        }
        if (!entry) entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid save response');
        if (signedOut) return;
        saving = false;
        conflicted = false;
        autosavePaused = false;
        autosaveReady = true;
        saveNeedsVerification = false;
        unverifiedDraft = false;
        baseVersion = entry.version;
        exists = entry.exists;
        lastSavedText = entry.text;
        // Only normalize acknowledged text when the user has not typed since
        // the request snapshot. Later text remains dirty and backed up.
        if (textarea.value === sentText) textarea.value = entry.text;
        persistDraft();
        setSaveState(isDirty() ? '' : 'saved');
        notice(t(isDirty() ? 'editor.unsaved' : (entry.exists ? 'editor.saved' : 'editor.blank')));
      } catch (error) {
        if (signedOut) return;
        autosavePaused = true;
        saveNeedsVerification = true;
        unverifiedDraft = true;
        const kept = persistDraft();
        setSaveState('');
        notice(t('editor.saveFailed') + (kept ? t('editor.draftKept') : '') + (automatic ? t('editor.autosavePaused') : ''));
      } finally {
        saving = false;
        refreshControls();
        scheduleAutosave();
      }
    }

    async function deleteEntry() {
      if (!loaded || !exists || saving || deleting || resolving || signedOut) return;
      if (!window.confirm(t('editor.deleteConfirm'))) return;
      cancelAutosave();
      deleting = true;
      setSaveState('');
      notice(t('editor.deleting'));
      try {
        const response = await protectedFetch(`/api/v1/entries/${date}`, { method: 'DELETE' });
        if (signedOut) return;
        if (response.status !== 204) {
          autosavePaused = true;
          unverifiedDraft = true;
          const kept = persistDraft();
          notice(t(response.status === 401 ? 'editor.sessionExpired' : 'editor.deleteFailed') + (kept ? t('editor.draftKept') : ''));
          return;
        }
        textarea.value = '';
        lastSavedText = '';
        baseVersion = '';
        draft = null;
        exists = false;
        edited = false;
        conflicted = false;
        saveNeedsVerification = false;
        unverifiedDraft = false;
        autosaveReady = false;
        autosavePaused = false;
        const cleared = persistDraft();
        setSaveState('');
        setStatus(t(cleared ? 'editor.deleted' : 'editor.deletedBackupRetained'));
      } catch (error) {
        if (signedOut) return;
        autosavePaused = true;
        unverifiedDraft = true;
        const kept = persistDraft();
        notice(t('editor.deleteFailed') + (kept ? t('editor.draftKept') : ''));
      } finally {
        deleting = false;
        refreshControls();
      }
    }
    deleteBtn?.addEventListener('click', deleteEntry);

    async function resolveConflict(choice) {
      if (!conflicted || saving || deleting || resolving || signedOut) return;
      if (choice === 'other' && !window.confirm(t('editor.loadOtherConfirm'))) return;
      resolving = true;
      cancelAutosave();
      autosaveReady = false;
      refreshControls();
      notice(t('editor.loading'));
      let shouldSave = false;
      try {
        const response = await protectedFetch(`/api/v1/entries/${date}`, { cache: 'no-store', headers: { 'Cache-Control': 'no-store' } });
        if (!response.ok) throw new Error('load failed');
        const entry = await response.json();
        if (typeof entry.text !== 'string' || typeof entry.version !== 'string') throw new Error('invalid entry');
        if (signedOut) return;
        const mine = textarea.value;
        exists = entry.exists;
        lastSavedText = entry.text;
        baseVersion = entry.version;
        // Rebase explicitly on the just-fetched version. A subsequent writer
        // still causes a normal conflict; no unconditional overwrite is used.
        if (choice === 'other') textarea.value = entry.text;
        if (choice === 'both') textarea.value = mine + (mine && entry.text ? '\n\n' : '') + entry.text;
        draft = null;
        edited = choice !== 'other';
        autosavePaused = false;
        saveNeedsVerification = false;
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
      autosaveReady = loaded;
      if (!loaded) unverifiedDraft = true;
      persistDraft();
      setSaveState(saving ? 'saving' : '');
      notice(t(saving ? 'editor.saving' : isDirty() ? 'editor.unsaved' : (exists ? 'editor.saved' : 'editor.blank')) + (autosavePaused && !conflicted ? t('editor.autosavePaused') : ''));
      scheduleAutosave();
    });
    saveBtn.addEventListener('click', () => saveEntry());
    // Some browsers blur the textarea without focusing a clicked button.
    [saveBtn, deleteBtn].forEach((button) => button?.addEventListener('pointerdown', () => { explicitAction = true; }));
    window.addEventListener('pointerup', () => { explicitAction = false; });
    window.addEventListener('pointercancel', () => { explicitAction = false; });
    textarea.addEventListener('blur', (event) => {
      // Let explicit actions run first; autosaving here would disable Delete
      // before its click and save text the user intended to discard.
      if (!explicitAction && event.relatedTarget !== deleteBtn && event.relatedTarget !== saveBtn) saveEntry({ automatic: true });
    });
    retryBtn?.addEventListener('click', loadEntry);
    legacyBtn?.addEventListener('click', () => {
      if (!loaded || saving || signedOut || isDirty() || legacyDraft === null) return;
      if (!window.confirm(t('editor.legacyDraftConfirm'))) return;
      cancelAutosave();
      autosaveReady = false;
      textarea.value = legacyDraft;
      edited = true;
      if (legacyVersionKnown) { baseVersion = legacyVersion; draft = { versionKnown: true }; }
      if (persistDraft()) {
        try { localStorage.removeItem(legacyKey); } catch (error) { /* Safe to retain the old copy. */ }
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
    function saveBeforeLeaving() {
      if (!isDirty() || signedOut) return;
      persistDraft();
      // Best effort only: navigation may prevent an acknowledgement. The draft
      // and leave warning remain until the normal save path confirms success.
      saveEntry({ automatic: true, keepalive: true });
    }
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') saveBeforeLeaving();
    });
    window.addEventListener('pagehide', saveBeforeLeaving);
    window.addEventListener('storage', (event) => {
      if (event.key === `mj-signout:${scope}`) {
        signedOut = true;
        legacyDraft = null;
        legacyVersion = '';
        legacyVersionKnown = false;
        if (legacyBtn) legacyBtn.hidden = true;
        cancelAutosave();
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

  function initSignOut() {
    const signOut = document.querySelector('[data-sign-out]');
    const disconnect = document.querySelector('[data-disconnect]');
    const buttons = [signOut, disconnect].filter(Boolean);
    const status = document.querySelector('[data-sign-out-status]');
    const scope = document.body.dataset.draftScope;
    if (!buttons.length || !status || !scope) return;
    let loggedOut = false;
    function unlock(active) {
      buttons.forEach((button) => { button.disabled = loggedOut && button !== active; });
    }
    buttons.forEach((button) => button.addEventListener('click', async () => {
      if (button.disabled) return;
      const revoking = button === disconnect;
      if (!loggedOut && revoking && !window.confirm(t('settings.disconnectConfirm'))) return;
      buttons.forEach((button) => { button.disabled = true; });
      if (!loggedOut) {
        status.textContent = t(revoking ? 'settings.disconnecting' : 'settings.signingOut');
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
          unlock(button);
          return;
        }
        try {
          const response = await (revoking ? protectedFetch('/api/v1/disconnect', { method: 'POST' }) : fetch('/api/v1/logout', { method: 'POST' }));
          if (revoking ? response.status !== 204 : !response.ok) throw new Error('connection not ended');
          loggedOut = true;
        } catch (error) {
          status.textContent = t(revoking ? 'settings.disconnectFailed' : 'settings.signOutFailed');
          unlock(button);
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
        unlock(button);
      }
    }));
  }

  function initEntriesList() {
    const listRoot = document.querySelector('[data-entry-list]');
    if (!listRoot) return;
    const listLimit = Number(listRoot.dataset.listLimit);
    if (!Number.isInteger(listLimit) || listLimit < 1 || listLimit > 100) return;

    const status = document.querySelector('[data-entries-status]');
    const retry = document.querySelector('[data-entries-retry]');

    function loadEntries() {
      if (retry) { retry.disabled = true; retry.hidden = true; }
      listRoot.setAttribute('aria-busy', 'true');
      if (status) status.textContent = t('entries.loading');
      return protectedFetch(`/api/v1/entries?page=1&size=${listLimit}`)
        .then(async (response) => {
          if (!response.ok) {
            throw new Error(t('entries.unavailable'));
          }
          return response.json();
        })
        .then((data) => {
          if (!Array.isArray(data.items) || data.items.length > listLimit || data.page !== 1 || data.pageSize !== listLimit) throw new Error('invalid list');
          if (status) status.textContent = '';
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
              const row = document.createElement('a');
              row.href = `/entries/${item.date}`;
              row.className = 'mj-entry-row';

              const time = document.createElement('time');
              time.dateTime = item.date;
              const date = new Date(`${item.date}T00:00:00Z`);
              time.textContent = date.toLocaleDateString(document.documentElement.lang, { weekday: 'short', day: '2-digit', month: 'short', timeZone: 'UTC' });

              const body = document.createElement('div');
              const title = document.createElement('h3');
              title.textContent = item.preview || t('entries.previewUnavailable');
              body.appendChild(title);

              const link = document.createElement('span');
              link.className = 'mj-entry-link';
              link.textContent = t('entries.open');
              body.appendChild(link);

              row.appendChild(time);
              row.appendChild(body);
              yearSection.appendChild(row);
            });

            listRoot.appendChild(yearSection);
          });
        })
        .catch(() => {
          const error = document.createElement('p');
          error.className = 'mj-empty-state';
          error.textContent = t('entries.unavailable');
          listRoot.replaceChildren(error);
          if (retry) retry.hidden = false;
          if (status) status.textContent = t('entries.unavailable');
        }).finally(() => {
          listRoot.setAttribute('aria-busy', 'false');
          if (retry) retry.disabled = false;
        });
    }
    retry?.addEventListener('click', () => { if (!retry.disabled) loadEntries(); });
    loadEntries();
  }

  function currentMonth() {
    return (document.body.dataset.today || formatDate(new Date())).slice(0, 7);
  }

  function shiftMonth(month, delta) {
    const date = new Date(`${month}-01T00:00:00Z`);
    date.setUTCMonth(date.getUTCMonth() + delta);
    return `${String(date.getUTCFullYear()).padStart(4, '0')}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
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
    let calendarRequest = 0;
    const maximumMonth = currentMonth();
    const calendarStatus = calendar.querySelector('[data-calendar-status]');
    const picker = window.mjInitMonthPicker?.(calendar, {
      month, max: maximumMonth, text: t,
      onSelect(value) {
        if (value < '0001-01' || value > maximumMonth) return;
        month = value;
        loadCalendar();
      },
    });

    function updateMonthControls() {
      monthLabel.textContent = new Date(`${month}-01T00:00:00Z`).toLocaleDateString(document.documentElement.lang, {
        month: 'long', year: 'numeric', timeZone: 'UTC',
      });
      const trigger = calendar.querySelector('[data-month-trigger]');
      if (trigger) trigger.setAttribute('aria-label', `${t('calendar.chooseMonth')}: ${monthLabel.textContent}`);
      prevBtn.disabled = month <= '0001-01';
      nextBtn.disabled = month >= maximumMonth;
      picker?.setMonth(month);
    }


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
      const request = ++calendarRequest;
      const requestedMonth = month;
      updateMonthControls();
      if (calendarStatus) calendarStatus.textContent = t('calendar.loading');
      grid.setAttribute('aria-busy', 'true');
      try {
        grid.innerHTML = '';
        const response = await protectedFetch(`/api/v1/calendar?month=${requestedMonth}`, undefined, () => request === calendarRequest);
        if (!response.ok) throw new Error(t('calendar.unavailable'));
        const data = await response.json();
        if (request !== calendarRequest) return;
        if (`${String(data.year).padStart(4, '0')}-${String(data.month).padStart(2, '0')}` !== requestedMonth || !Array.isArray(data.weeks)) throw new Error('invalid calendar');
        if (calendarStatus) calendarStatus.textContent = '';

        Array.from({ length: 7 }, (_, offset) => {
          const weekday = new Date(Date.UTC(2024, 0, 1 + offset));
          return weekday.toLocaleDateString(document.documentElement.lang, { weekday: 'short', timeZone: 'UTC' });
        }).forEach((name) => {
          const heading = document.createElement('span');
          heading.textContent = name;
          grid.appendChild(heading);
        });
        data.weeks.flat().forEach((day) => grid.appendChild(renderDay(day)));
      } catch (error) {
        if (request !== calendarRequest) return;
        grid.innerHTML = '';
        if (calendarStatus) calendarStatus.textContent = t('calendar.unavailable');
      } finally {
        if (request === calendarRequest) grid.setAttribute('aria-busy', 'false');
      }
    }

    prevBtn.addEventListener('click', () => {
      if (prevBtn.disabled) return;
      month = shiftMonth(month, -1);
      loadCalendar();
    });
    nextBtn.addEventListener('click', () => {
      if (nextBtn.disabled) return;
      month = shiftMonth(month, 1);
      loadCalendar();
    });
    todayBtn.addEventListener('click', () => {
      month = maximumMonth;
      loadCalendar();
    });
    loadCalendar();
  }

  document.addEventListener('DOMContentLoaded', () => {
    if (window.mjDateRedirecting) return;
    initEditor();
    initSignOut();
    initEntriesList();
    initCalendar();
  });
})();
