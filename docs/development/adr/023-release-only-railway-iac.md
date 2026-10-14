# ADR 023: Go Railway IaC applied only on published releases

- Status: Accepted
- Date: 2026-10-15

## Decision

Replace per-deployment `railway.json` with `.railway/railway.go`, using an
isolated Go module and pinned beta SDK v0.3.0. Pin Railway CLI 5.68.0 to evaluate
the graph and its explicit `variables.managed=false` policy. All Railway variable
values remain externally managed, including secrets. No application dependency
on the SDK is introduced.

Infrastructure apply and application upload both run only in the existing
serialized production workflow for published, non-prerelease GitHub Releases.
Retain tag validation, readiness, main ancestry, code checks, exact source
archives/version metadata, deployment-ID SUCCESS polling and public health
verification. Tag pushes, PRs and merges do not deploy. Do not adopt the Railway
example's apply-on-merge workflow.

Read the origin/project/environment/service target from the release job’s GitHub
variables, with no hardcoded origin or IDs in deployment code. Compare the IaC
CLI context against the supplied project/environment IDs and require the
production environment name. Require an HTTPS origin for public health checks.

Target the existing production service with one replica in its existing region,
Dockerfile builds, `/healthz`, and the previous timeout/restart policy. Remove
its GitHub source attachment on the first release-time apply, leaving CLI source
uploads as the only application source. Apply a validated pinned plan; reject
resource creation/deletion, variable changes and unrelated settings. Verify
configuration convergence before uploading the release archive.

## Context and consequences

Railway documents Config as Code as deprecated with a December 1, 2026 cutoff.
IaC is evaluated explicitly by the CLI, rather than read during source upload.
The Go SDK is beta and does not expose a project variable-policy helper in
v0.3.0; the authoring function adds that supported graph field directly.
Offline regressions cover the emitted graph and release/plan restrictions.

Configuration apply may trigger deployment work inside the release job. One
replica remains best effort across revisions and is not a distributed lock;
ADR 016's overlap and Drive concurrency limitations remain accepted. An
ambiguous apply/upload must be inspected before retrying. Older source rollback
does not automatically revert IaC or variables; retain compatible production
settings and stable secrets.

## References

- [Railway IaC](https://docs.railway.com/infrastructure-as-code)
- [Railway CLI config](https://docs.railway.com/cli/config)
- [Authoring and operational notes](../../../.railway/README.md)
