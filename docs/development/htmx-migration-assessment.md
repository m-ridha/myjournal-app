# HTMX migration assessment

Status: researched and rejected by user decision on 2026-10-09. **We will not use HTMX for this project; keep the JSON API and vanilla JavaScript.** See [ADR 014](adr/014-json-vanilla-javascript.md). The scope, estimates, examples, and sources below are retained as historical research, not planned work or implemented interfaces. Estimates exclude the architecture/theme work already in the working tree.

## Investigated scope

The assessment recommended retaining the current JSON API and vanilla JavaScript as the shortest route to Google integration. The alternative investigated was starting with Entries and Calendar while retaining the editor controller. The user chose to retain the current architecture and exclude HTMX from the project.

Current app.js has 770 lines: about 443 in the editor initializer, about 101 for Entries, and about 104 for Calendar. The remainder includes account cleanup and shared helpers. Most editor code manages drafts, versions, conflicts, and acknowledgements rather than rendering. Replacing fetch alone cannot remove those responsibilities.

| Scope | Estimated files added/changed | Main work |
| --- | --- | --- |
| Entries and Calendar | 8–12 application/test files, plus documentation and vendor license | Server-rendered view models and fragments, two page templates, renderer/router, removal of browser DOM rendering, pinned library/configuration, fragment/history/order regressions |
| All interactive flows | 15–25 application/test files, plus documentation and vendor license | Above plus editor transport/status fragments, form parsing, error handling, save lifecycle hooks, trash/account actions, and recovery regression adaptation |

These are repository-based planning estimates, not a measured patch. About 200 lines of browsing DOM-rendering code could move to Go templates/view models. Total authored code need not shrink: fragment routing, error views, and tests are added, and the vendored library is additional code. A full migration would still retain substantial custom editor JavaScript.

Domain, stores, cache, sessions, provider ports, Wire, and Google implementation need no redesign for HTMX. Keep the existing JSON /api/v1 endpoints and OpenAPI contract; browser HTML handlers call the same injected journal.Service directly rather than making HTTP calls to their own API.

## Example: Entries pagination

Today the Entries shell is returned first; initEntriesList fetches JSON and builds year headings, previews, and links in the browser:

```js
const response = await fetch(`/api/v1/entries?page=${page}&size=20`);
const data = await response.json();
// Validate data, group by year, build rows, update Newer/Older and Retry.
```

With HTMX, /entries?page=2 returns a complete page for ordinary navigation and an entries-body fragment for HX-Request. The proposed template would include year-grouped rows and pagination inside the same replacement target:

```html
{{define "entries-body"}}
<div id="entries-body" hx-history="false">
  {{range .Years}}
    <section class="mj-entry-year">
      <h2>{{.Year}}</h2>
      {{range .Items}}
        <article class="mj-entry-row">
          <time datetime="{{.Date}}">{{.DisplayDate}}</time>
          <h3>{{.Preview}}</h3>
          <a href="/entries/{{.Date}}">{{t "entries.open"}}</a>
        </article>
      {{end}}
    </section>
  {{end}}
  {{if .HasNext}}
    <a href="/entries?page={{.NextPage}}"
       hx-get="/entries?page={{.NextPage}}"
       hx-target="#entries-body" hx-swap="outerHTML"
       hx-sync="#entries-body:replace" hx-push-url="true">
      {{t "entries.older"}}
    </a>
  {{end}}
</div>
{{end}}
```

The production template also needs Newer, empty/error/retry states, accessible loading feedback, and retained page bounds. Normal href navigation remains available. Go html/template escapes previews. hx-sync coordinates overlapping requests; rapid navigation and Back/Forward still require browser tests.

The relevant response selection would look like this inside a new rendering helper, after authentication, query validation, service.Page, and view-model construction:

```go
w.Header().Add("Vary", "HX-Request")
w.Header().Set("Content-Type", "text/html; charset=utf-8")
name := "base"
if r.Header.Get("HX-Request") == "true" {
    name = "entries-body"
}
// data must include the shared page fields and the new year/page view model.
if err := pageTemplates.ExecuteTemplate(w, name, data); err != nil {
    // Use the existing localized render-error handling.
}
```

This is not a replacement for the generated JSON ListEntries handler. render.go currently accepts only pageData; it must be extended deliberately for the new view model. The full-page entries template would invoke entries-body so both paths share markup. HX-Request is a rendering hint, never authentication.

## Example: Calendar

The month and date-cell markup would move out of initCalendar into a Go template. Previous/next month links would use real month URLs and the same target/synchronization pattern:

```html
<a href="/calendar?month={{.NextMonth}}"
   hx-get="/calendar?month={{.NextMonth}}"
   hx-target="#calendar-body" hx-swap="outerHTML"
   hx-sync="#calendar-body:replace" hx-push-url="true"
   aria-label="{{t "calendar.nextMonth"}}">→</a>
```

The page handler would validate month, resolve browser time zone, call service.Calendar, and build weekday/date labels. Today, entry markers, future-date disabling, stale failures, history, and keyboard focus all need preserved tests. Replacing the whole calendar after a click also requires a focus policy.

## Why the editor is harder

The current save captures a text snapshot and baseVersion, sends JSON, then reconciles acknowledgement with any newer typing. A tempting conversion is a form with hx-put and hx-trigger="input changed delay:5s". That does not preserve the accepted behavior by itself: a queued request may use a stale version, recovered text must not save before review, failures must pause subsequent automatic writes, and an older response must never replace newer typed text.

A safer full conversion would keep the textarea/controller mounted and swap only a status/conflict area. A proposed manual-save form could use:

```html
<form hx-put="/ui/entries/{{.Date}}"
      hx-target="#entry-feedback" hx-swap="innerHTML">
  <textarea id="entry-text" name="text" aria-label="{{t "editor.label"}}">{{.Text}}</textarea>
  <input type="hidden" name="baseVersion" value="{{.Version}}">
  <button type="submit">{{t "editor.save"}}</button>
  <div id="entry-feedback" role="status" aria-live="polite"></div>
</form>
```

That form is illustrative, not a safe autosave implementation. /ui/entries/{date} would be a new HTML/form endpoint because the existing API requires JSON. The controller still needs snapshot/version reconciliation, one-flight follow-up scheduling, storage/error handling, initial-load locks, all three conflict choices, and sign-out/trash coordination. Hooks must handle 401/409/413/503 and ambiguous outcomes without swapping away text. No automatic retries or acknowledgement inferred merely from sending a request. HTMX defaults do not render error responses as ordinary successful swaps; explicit error handling is needed.

## Investigated migration requirements (not planned)

Vendor a pinned HTMX release and license under static assets; no CDN, framework, bundler, or OAuth-scope change is needed. Configure allowEval=false, allowScriptTags=false, includeIndicatorStyles=false, selfRequestsOnly=true, historyCacheSize=0, and historyRestoreAsHxRequest=false. Use external listeners rather than inline hx-on or js: expressions to preserve CSP. Add request-indicator CSS locally. Disable history snapshots for journal pages: default HTMX history can otherwise put rendered journal content in localStorage outside accepted tab-scoped backups. Keep dynamic no-store headers and Vary: HX-Request for full/fragment responses.

New tests must cover full page versus fragment content, escaped user text, cookie/cross-origin protection, correct error statuses, rapid requests, Back/Forward, no sensitive history snapshots, focus after swaps, and CSP. Existing domain/store/API tests stay. The editor's dependency-free fake-DOM tests cannot establish real HTMX event/swap behavior; browser regressions would be needed in addition to preserving their state-machine checks.

Theme source/version/license provenance is a separate unresolved gate; adding HTMX does not resolve it. No missing upstream provenance has been invented.

Sources checked for this assessment: [HTMX documentation](https://htmx.org/docs/) (HTML responses, configuration, errors, caching), [hx-sync](https://htmx.org/attributes/hx-sync/) (request coordination), and [hx-history](https://htmx.org/attributes/hx-history/) (sensitive history exclusions). Estimates and the recommended migration boundaries are conclusions from this repository, not claims made by those sources.
