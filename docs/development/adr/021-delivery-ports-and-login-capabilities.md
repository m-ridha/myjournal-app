# ADR 021: Inject delivery ports and declare login-flow capabilities

Status: accepted

Date: 2026-10-13

## Context

The architecture audit identified direct web imports of the concrete cookie codec,
a concrete connection use-case dependency, and login-flow selection by LocalID.
The user authorized all recommended refinements while preserving behavior.

## Decision

Web authsession owns the Codec contract for session and pending-login payloads.
The driven session.Codec binds the startup-validated secret to the existing
AES-GCM implementation and structurally implements that contract without importing
web. Only cmd/app constructs the concrete codec. Delivery receives the interface,
keeps cookie names/attributes/size limits and HTTP mapping, and receives no raw key.
Existing ciphertext, key derivation and purpose separation remain compatible.

HTTP delivery consumes handler.ConnectionService for Begin, Complete, Disconnect
and Settings, alongside its existing JournalService interface. Core constructors
continue returning concrete services. Wire binds connection.Service to the inbound
contract and Registry to core Catalog; web cannot construct journal storage via
its provider catalog. Tests can inject codec/use-case failures independently while
retaining real-adapter integration coverage.

Gateway.RequiresLoginFlow explicitly declares whether a pending browser
transaction is required. Google returns true and local development returns false.
Core validates state, provider binding, lifetime and required code before exchange;
web applies the same capability when parsing the callback cookie. Capabilities do
not bypass startup provider/origin checks or introduce additional providers.

Production web may no longer import any driven package. Depguard removes the
session-codec exception, and three additional negative-import probes cover direct
and nested imports. There are now 34 architecture probes.

Google Store stays one adapter package. Construction, discovery, reads/previews,
mutations and retry/error translation are separated into focused files without
changing their function declarations or external behavior. Caching, warming and
conflict/recovery protocols remain independent of this file organization.

## Consequences

Replacing the encryption implementation or substituting connection use cases in
HTTP tests no longer requires concrete adapter imports in delivery. Adding a new
Gateway implementation requires explicitly declaring its login-flow requirement.
The configured provider implementations remain trusted startup dependencies.

This supersedes ADR 018's concrete codec exception, not its cookie compatibility
requirements. It does not add a session database, new providers, distributed
coordination or new Google/production acceptance claims.
