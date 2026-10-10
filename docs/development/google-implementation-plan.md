# Google integration implementation and validation

Authorized 2026-10-10. Preserve existing local/editor behavior and all uncommitted work; no commit or publication is part of this task.

1. Implement authorization-code exchange with retained PKCE, Google ID-token signature/claims/nonce validation, required Drive grant, account-bound refresh-token preservation, compact versioned credentials, bounded access-token cache, and revocation. Test success and rejected/missing claims, permissions, refresh failures, and omitted refresh tokens.
2. Implement the journal Store over Drive: private app-tagged folder discovery, deterministic duplicate selection, write-only recreation, plain UTF-8/LF content, checksum versions, shared account mutation locks, complete listing pagination, bounded preview concurrency/reads, context-aware retry of safe reads, and safe error mapping. Run shared store contracts against a mocked Drive server and failure/concurrency tests. Do not blindly retry writes.
3. Wire only an explicit localhost Google integration-test mode; retain the production guard. Add settings folder link without creating folders on reads. Keep cookie size bounded and preserve drafts on reconnect/account changes.
4. Run formatting/lint/tests, architecture probes separately, focused race tests, and local synthetic UI checks where useful. Document actual results and unresolved Google race/revision/live limitations.
5. Supply a human checklist for Console redirect/scopes/test users, integrated sign-in/refresh/storage/conflicts/revocation, and synthetic revision measurements. Human browser consent and real account evidence are required; never infer a pass from mocks.

Review focus: omitted refresh token must never borrow another account's credential; canceled/failed writes retain drafts; checksum comparison is not atomic against external writers; folder duplicates/data must survive; transient/ambiguous creates are never blindly retried. Single-process locks do not establish multi-instance safety. Production enabling remains deferred until live acceptance and race policy are settled.
