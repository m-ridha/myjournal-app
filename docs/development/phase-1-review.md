# Phase 1 review and remaining work

Reviewed 2026-10-08. The user gave a go decision to continue after the S1/S2/S4/S7 evidence. This authorizes further implementation; it does not turn unobserved experiments into passing results or enable the incomplete Google provider.

| Experiment | Evidence or next action |
| --- | --- |
| S1 OAuth | Test-account sign-in, claim validation, refresh, and subsequent omission of refresh token recorded |
| S2 folders | Rename, visibility, trash, guarded recreation, and move observations recorded |
| S3 concurrency | Five naive workers created five paths; shared lock produced one path for all workers (human-reported 2026-10-08) |
| S4 conflict token | `md5Checksum` selected; check/update race remains unresolved |
| S5 revisions | Conservative fallback below; repeated-update measurement still deferred |
| S6 latency/rate limits | Bounded-work fallback below; production percentile measurement still deferred |
| S7 cookie lifecycle | Accepted for the tested account/browsers with Safari localhost workaround; fixed 30-day lifetime adopted |
| S11 production OAuth | Official-doc research below; Console/domain/production observations deferred until hosting |
| S12 libraries | Retain existing packages and standard-library choices below |

## S5 revision fallback

Do not assume how many Drive revisions autosave retains or expose revisions as a recovery guarantee. Preserve manual saving and browser draft recovery. When implementing autosave, skip unchanged content and use a five-second idle debounce with one save in flight; serialize any follow-up save against the returned version. Page-hide saves remain dependent on text-safety and browser acceptance. This reduces redundant writes without claiming a measured revision limit. No revision-pinning or cleanup calls are planned. Google documents that blob revisions can be purged; revision history must not substitute for accurate save state. [Drive revision guide](https://developers.google.com/workspace/drive/api/guides/manage-revisions).

This fallback is a plan, not implemented autosave or a completed 50–100-update experiment. Before enabling autosave with real Google storage, measure changed/unchanged saves using synthetic files and record counts/size/behavior.

## S6 bounded-work fallback

Keep the existing bounded, account-scoped cache and page-only preview fetches. For the Google adapter, cap preview concurrency at four, bound reads to the configured preview size, and parse every Drive listing page. HTTP range reads are an optional optimization until measured; fall back to reading only the required prefix and closing the response. Use injected bounded exponential backoff with jitter for retryable rate-limit and transient failures, respecting request cancellation. Google recommends exponential backoff for rate-limit failures. [Drive error guide](https://developers.google.com/workspace/drive/api/guides/handle-errors).

Do not retry an ambiguous file creation blindly: requery the app-tagged path and preserve duplicates. Keep retries bounded and preserve drafts on unavailable storage. These adapter behaviors still need deterministic tests and live latency measurements. The S4 349 ms metadata-read sample is not a p50/p95 result; no performance acceptance is claimed. Record the planned 20-run cold/warm measurements before making production performance promises.

## S11 production OAuth research

Google classifies `drive.file` as non-sensitive. Keep the existing four scopes; there is no reason to request restricted full-Drive scopes for journal storage. [Drive scope reference](https://developers.google.com/workspace/drive/api/guides/api-specific-auth).

Google documents seven-day refresh-token expiry for External apps in Testing, except identity-only scope sets. Because this app requests `drive.file`, that exception does not apply. A 30-day browser session cannot extend Google's token lifetime. If the current Console project is in Testing, expect re-consent as needed; no week-long live expiry test or Console publishing-state inspection was performed here. [Google OAuth token-expiration documentation](https://developers.google.com/identity/protocols/oauth2).

For friends/family use without Testing's weekly expiry, plan a separate production project with production publishing status and narrow scopes. This is a plan, not an authorization to publish now. Non-sensitive scope use avoids sensitive/restricted scope review, but brand verification remains relevant to displaying the app's identity. [OAuth verification help](https://support.google.com/cloud/answer/13463073?hl=en).

Before release, provide a public homepage explaining the app and linking a privacy policy on the same domain; verify authorized-domain ownership, align HTTPS callback URLs, and complete applicable brand verification. Development projects can remain in Testing with named test users and its tester warning. Personal-use exceptions exist but do not establish a warning-free experience. We cannot guarantee the eventual consent screen without reviewing the real production Console configuration. [Google brand verification requirements](https://developers.google.com/identity/protocols/oauth2/production-readiness/brand-verification).

The fallback while no production domain exists is to continue local/test-account development without publishing or promising persistent production access. Hosting/Console steps and a post-publishing sign-in/refresh check belong to release acceptance.

## S12 supporting-library choices

- Keep the existing standard-library AES-GCM codec; S7 provided round-trip/lifecycle evidence. Implement production age enforcement against the approved lifetime before enabling Google.
- Use standard-library `net/http.CrossOriginProtection` for unsafe browser requests. The installed Go toolchain exposes it (`go doc net/http.CrossOriginProtection`), and S7 tests reject a cross-origin POST. OAuth callbacks still require explicit state and PKCE checks. Production CSRF wiring is pending.
- Keep `html/template`, chi, `log/slog`, `cloud.google.com/go/civil`, `golang.org/x/oauth2`, and `google.golang.org/api/drive/v3`/`idtoken`. These are existing dependencies or standard packages exercised in the repository/spikes; no new dependency or maintenance ranking was introduced.
- Add focused security headers in the existing web layer during the security phase. Retain Google Wire and oapi-codegen as already decided. The vanilla-JavaScript/HTMX discrepancy still needs its own decision; these choices do not silently settle it.

## Go decision and gates

The tested Google operations support continuing with the current narrow-scope, plain-text, encrypted-cookie architecture. User go received on 2026-10-08. S3 live results confirmed one path under a shared in-process lock, compared with five paths from the naive workers. Phase 1 evidence/fallback requirements are recorded; S5/S6 mitigations remain documented plans rather than measured results. Production OAuth setup remains a release dependency.

For production folder creation, share a lock per provider/account across session-scoped stores and use stable oldest-wins selection when queries return duplicates. Preserve data in other folders; never silently delete or merge them. The live run demonstrated single-process serialization only, not cross-instance consistency or Drive list visibility guarantees.

Before Phase 4 acceptance, resolve Phase 2/3 text safety, date correctness, lifecycle/bounds, contract, and UI acceptance gaps. Provider implementation must preserve refresh tokens, enforce session age, use path-only logs, implement OAuth state/PKCE/claim checks, and protect writes from stale versions without claiming a pre-save checksum check is atomic. Google registration remains disabled until implementation and validation pass.
