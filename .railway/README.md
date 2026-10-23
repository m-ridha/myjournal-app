# Production infrastructure

`railway.go` defines the existing myjournal production service using the beta
Go authoring SDK v0.3.0 in this isolated module. The application module has no
Railway SDK dependency. The release job supplies `APP_BASE_URL`,
`RAILWAY_PROJECT_ID`, `RAILWAY_ENVIRONMENT_ID` and `RAILWAY_SERVICE_ID` from
GitHub variables. CLI 5.70.0 evaluates the graph, including its explicit
unmanaged-variable policy; secrets and all other variable values stay on Railway.

The release path in `.github/workflows/release.yml` applies configuration after a regular
GitHub Release is published and tag/main-ancestry validation and code checks pass.
Tag pushes, PRs and merges do not deploy.
PR checks evaluate the Go graph offline without Railway credentials.

The release helper targets the configured environment via `RAILWAY_TOKEN`
and `--environment`, writes a temporary pinned plan outside the upload archive,
rejects resource creation, deletion, variable changes and unexpected targets,
applies that plan without destructive authorization, and verifies convergence
before uploading release source. Deployment-ID SUCCESS polling and public
`/healthz` verification follow. The first apply removes the GitHub source
attachment; CLI archives are the sole application source. Configuration
applies can themselves trigger deployment work within that release job; one
replica still permits temporary revision overlap.

Do not run `config apply` or `config migrate --apply` manually. For a read-only
review, export `RAILWAY_PROJECT_ID` and `RAILWAY_ENVIRONMENT_ID` matching the
GitHub variables, use the pinned CLI, link the existing production target, then run
`railway config plan`. Never request decrypted variables or include their values.
If a legacy Railway Config File setting blocks planning, stop and prepare its
removal in the release path; do not apply a separate bootstrap deployment.

The configuration deliberately targets the existing Singapore region. Changing
service identity, regions, variables or resource topology requires reviewing the
release helper's restrictions too. Keep one replica and existing secrets,
especially `SESSION_KEY`; rollback does not revert Google Drive content.
