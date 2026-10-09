# ADR 011: Isolate draft backups by browser tab

Status: Accepted (2026-10-09)

The user explicitly chose reload recovery over recovery in a fresh tab after closing an old one. Use account/date-scoped `mj-draft:v3` keys in `sessionStorage` for new drafts. Each tab owns its backup, including its original base version. Browser session storage survives reloads and same-tab navigation. Browser-duplicated tabs may start with a copied snapshot; later changes use independent stores. Closing a tab can end recovery; browser session restoration behavior is browser-dependent.

Previously shared `mj-draft:v2` local-storage backups and older date-only backups remain available through explicit Review older draft. Preserve the version of an account-scoped backup. Remove its old local copy only after writing the new backup successfully. The user must confirm replacement of clean loaded text.

Successful saves clear only that tab's backup. Sign-out checks storage access without modifying drafts, then requests logout. Only a successful response triggers the final localStorage broadcast and clears this tab’s account drafts and old local backups. Other open editor tabs then clear account drafts and stop further writes. Failed or uncertain logout responses retain backups and do not lock editors. A cleanup failure after confirmed logout is reported separately and can retry cleanup without another logout request. Theme storage remains in localStorage. A shared listener clears tab backups on other open app pages too. Suspended tabs that miss the notification or unavailable storage may retain their backup until a later sign-out or tab-session end; account scoping prevents another account from reading it through the editor.

On conflict, fetch the latest entry/version before an explicit resolution. Keep mine retries a conditional save; a further change returns a new conflict. Load the other version requires confirmation. Keep both appends saved text below local text and remains unsaved for review. These operations preserve the selected entry date and ordinary save-snapshot behavior.
