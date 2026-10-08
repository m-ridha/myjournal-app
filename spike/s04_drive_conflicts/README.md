# S04: Drive conflict metadata

S02 remains the completed folder experiment. S04 creates a separate synthetic plain-text file in My Drive and compares `version`, `modifiedTime`, `md5Checksum`, and `headRevisionId`. No journal files are read or changed, and nothing is deleted automatically. Use the same dedicated test account and OAuth client as S01/S02.

Google documents these fields in the [Files resource reference](https://developers.google.com/workspace/drive/api/reference/rest/v3/files). Documentation informs the candidates; live results determine the choice. ADR 005 stays undecided until those results are reviewed.

## Automated experiment

Stop your own previous spike with Ctrl+C. From the repository root:

```sh
set -a
source ./.env
set +a
export APP_BASE_URL=http://localhost:3030

go run ./spike/s04_drive_conflicts \
  -mode=run \
  -baseline /tmp/myjournal-s4-baseline.json
```

The baseline path must not already exist. For another run, choose a new path. It stores resource identifiers and candidate metadata only, with mode 0600; it contains no credentials or file contents. Keep it local.

Open `http://localhost:3030/login` and sign in. The existing Console redirect remains `http://localhost:3030/auth/google/callback`. The spike uses only `openid`, `email`, `profile`, and `drive.file`.

The callback:

1. Creates `myjournal-s4-<run-id>.txt` with synthetic content A.
2. Loads client A's metadata, then writes different content B as client B.
3. Reads current metadata before A's attempted save and reports whether each candidate would reject A's stale save. A's stale content is never uploaded.
4. Re-reads to check that B's content checksum remains current.
5. Renames the file through the API and reports which candidates changed.
6. Saves the metadata baseline for the manual rename check.

The callback reports booleans, one metadata-read duration in milliseconds, and the test file ID. `metadata_check_ms` is one sample, not a latency percentile. Missing fields report `available: false` and cannot qualify as candidates.

Look for `content_edit_<field>_changed: true`, `stale_save_<field>_rejected: true`, and `api_rename_<field>_changed: false` for a usable candidate. `client_b_content_preserved` should be true. A completed experiment reports `S4_API_EXPERIMENT: completed`; completion does not mean all candidates passed. An incomplete run reports its failed stage and Google HTTP status when available, without raw external error bodies. Save the test file ID even on partial failure so you can inspect it later.

## Manual Drive rename

In Drive, locate the synthetic file using its ID or generated name. Rename it again without editing its contents. Stop the spike and restart using the same baseline:

```sh
go run ./spike/s04_drive_conflicts \
  -mode=check-rename \
  -baseline /tmp/myjournal-s4-baseline.json
```

Open `http://localhost:3030/login` and sign in again. This mode only reads the tagged test file. A suitable candidate remains available and reports `manual_rename_<field>_changed: false`. Confirm that you actually renamed the file: the harness does not prove the manual action happened.

Share the boolean results and timing, without the callback URL, authorization codes, tokens, baseline contents, or file contents. Record human-observed evidence in `docs/development/findings.md`; update ADR 005 only after reviewing it. You may move the synthetic test file to Drive trash manually after both stages.

## Limits and next steps

The two clients are sequential snapshots, not separate devices. This checks a pre-save comparison; it does not prove atomic conditional writes or eliminate a race between checking metadata and updating content. A checksum also cannot detect an A→B→A history when the final bytes match. The initial experiment covers one content change and API/manual renames; repeat runs add evidence, and sharing changes remain untested.

S7 needs a separate encrypted-cookie/token lifecycle harness covering refresh, restart, rotation, tampering, revocation, cookie size/attributes, and browser behavior. S04 does not enable the production Google provider.
