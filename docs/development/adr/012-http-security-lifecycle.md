# 012: Bound HTTP lifecycle and enforce same-origin assets

On 2026-10-09, the local app adopted explicit HTTP deadlines: five seconds for headers, fifteen seconds for reading a request, thirty-five seconds for writes, and sixty seconds for idle connections. The existing thirty-second handler context deadline stays in place; the write deadline leaves room to send its response. These limits bound network operations, not arbitrary computation in handlers that ignore cancellation.

SIGINT/SIGTERM initiates a ten-second graceful shutdown. Main waits for active requests to drain before deferred adapter cleanup. If draining exceeds the deadline, the server closes remaining connections and reports failure. A lost response can leave save acknowledgement uncertain, so browser drafts must remain recoverable. Process termination and power loss cannot promise a completed save.

All app responses carry a Content Security Policy allowing only same-origin scripts, styles, images, fonts, and API connections. Inline scripts/eval, objects, base URL overrides, and framing are blocked. Forms submit only to the app origin. The theme bootstrap is an embedded external script loaded before CSS; no nonce or unsafe-inline exception is required. Keep future assets compatible with this policy. Real OAuth redirect behavior still requires validation when Google is implemented.

Responses also use nosniff, DENY framing, and no-referrer. Pages/API/auth/error responses use no-store; embedded static assets retain ordinary HTTP caching behavior. No HSTS or upgrade-insecure-requests directive is imposed on HTTP localhost; deployment HTTPS policy remains separate.

Automated checks verify active-request draining, listener failure propagation, timeout relationships, headers on normal/redirect/error/cross-origin rejection responses, and external bootstrap availability/order. Subsequent live browser automation covered enforcement and theme/editor/calendar/recovery/conflict/logout flows in Chrome 155 and native Safari 26.4; see [CSP browser QA](../csp-browser-qa.md). Safari full startup-console history remains unverified. Static/header tests alone do not establish browser results.

References: [Go Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown), [CSP header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy).
