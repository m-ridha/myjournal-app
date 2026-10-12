# Google branding verification

Status: local pages and Search Console HTML tag prepared on 2026-10-11; deployment, ownership verification and Google acceptance pending.

The user selected the current Railway URL for the first ownership attempt and
<support@myjournal.ink> for public support. Confirm the mailbox receives messages
before publishing the policy. Branding verification does not itself authorize
unrestricted release or change the existing restricted test-user policy.

## Public pages

- Home: <https://myjournal-ink.up.railway.app/>
- Privacy: <https://myjournal-ink.up.railway.app/privacy>
- Terms: <https://myjournal-ink.up.railway.app/terms> (new local public route; deploy before configuring GCP)
- Consent-screen app name: myjournal (user reports corrected).

Local updates explain journal functionality and limited Drive access, and disclose
identity/tokens, journal processing, temporary memory, browser storage, sharing,
logging, retention, disconnect and deletion. The policy identifies Google and
Railway and links Google's Limited Use policy. No expanded scopes are needed.

## Ownership and retry

1. In Google Search Console, use a Google account that is an owner/editor of the
   GCP project. Add a URL-prefix property for the exact HTTPS home URL above.
2. HTML tag verification was selected in Search Console and the supplied
   google-site-verification meta tag is now in the landing page head. The
   URL-prefix property is awaiting verification; Verify has not been submitted.
   Keep it deployed after verification so Google can recheck ownership.
3. Deploy the approved page changes and verification tag through the authorized
   release process. Confirm home/privacy respond without signing in, without a
   crawler-blocking challenge or rate limit, with readable desktop/mobile layouts.
4. Click Verify in Search Console. If OAuth branding still rejects the shared
   Railway hostname, inspect Google's required top private domain. URL-prefix
   verification alone is not proof the OAuth domain requirement is satisfied;
   a user-owned custom domain may be needed. Do not claim ownership of railway.app.
5. After ownership is accepted, wait at least the 24 hours requested by Google's
   rejection notice. Ensure branding home/privacy/terms links exactly match these pages
   and the name is myjournal, then retry branding verification.
6. Record Google's result. Published page edits and Search Console success alone
   do not establish brand-verification approval.

Public curl requests from the development environment returned HTTP 429 during
initial investigation; the web retrieval tool could not load the pages. This is
not proof Google receives 429, but warrants checking the deployed public path
before resubmission. No bypass or weakening of application protections is needed.

References:

- [Google verification requirements](https://support.google.com/cloud/answer/13464321)
- [Brand verification and top private domains](https://developers.google.com/identity/protocols/oauth2/production-readiness/brand-verification)
- [Search Console ownership methods](https://support.google.com/webmasters/answer/9008080)
