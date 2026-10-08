# 002: Use encrypted signed cookie sessions

The app keeps session state in an encrypted, signed HttpOnly cookie instead of a database-backed server-side session. This matches the requirement to avoid storing journal content server-side while still carrying the provider connection and refresh token through the app lifecycle.

On 2026-10-08, the user adopted a fixed 30-day session lifetime, following the S7 lifecycle experiment. Enforce the deadline using the original `IssuedAt` as well as cookie expiry; access-token refresh must not extend it. This is the current product decision, not evidence that production age enforcement is implemented. The lifetime may be revised to one or two weeks during development. When shortening it, server-side age validation must also apply the new limit to existing cookies, even if their browser expiry is later.
