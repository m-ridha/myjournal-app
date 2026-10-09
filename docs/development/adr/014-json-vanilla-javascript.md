# 014: Keep the JSON API and vanilla JavaScript

Status: accepted by the user on 2026-10-09.

## Decision

Keep Go html/template pages, the versioned JSON API, and vanilla JavaScript for browser interactions. We will not use HTMX for this project. This supersedes the original HTMX plan; HTML fragments and Vary: HX-Request are not acceptance requirements for the chosen architecture.

## Reason

The current implementation already supports tested draft recovery, independent tab backups, conditional saves, conflict actions, and safe autosave. HTMX could move browsing markup into server templates, but would still require custom editor state and acknowledgement handling. Migration would add fragment endpoints, templates, dependency/configuration work, and renewed browser regression coverage without advancing Google authentication or Drive storage.

## Consequences

Preserve the JSON OpenAPI contract and existing browser controllers. Do not add HTMX, fragment routes, or a frontend framework as routine implementation work. Keep accessibility, localization, CSP, and text-safety acceptance requirements independent of this choice.

The [HTMX migration assessment](../htmx-migration-assessment.md) retains the investigated scope, examples, estimates, and sources as historical research, not an implementation plan.
