# S03: Concurrent folder creation

This separate experiment launches five concurrent list-then-create operations for each of two fresh, uniquely tagged test trees: naive, then locked. It creates folders only. Existing S02 roots and journal content are not touched. The callback counts folders per role and checks whether every worker returned the same root/entries/year path.

Stop your own prior spike, then run from the repository root:

```sh
set -a
source ./.env
set +a
export APP_BASE_URL=http://localhost:3030
go run ./spike/s03_folder_concurrency
```

Open `http://localhost:3030/login` and sign in with the test account. The callback remains `http://localhost:3030/auth/google/callback`; no Console changes or new scopes are required.

Expected locked-stage result:

```text
locked_root_count: 1
locked_entries_count: 1
locked_year_count: 1
locked_workers_same_path: true
locked_one_folder_per_role: true
S3_SINGLE_PROCESS_CONCURRENCY: passed
```

Naive counts may exceed one; record actual results. If the naive stage happens to produce one folder per role, that does not establish it is concurrency-safe. Every new sign-in creates fresh test trees. On failure, preserve the displayed run ID locally and inspect the corresponding trees rather than assuming nothing was created. The callback has a three-minute time limit, so wait for the result before retrying.

The locked stage serializes the full ensure-path sequence within this process. When a query already returns duplicates, it chooses the oldest creation timestamp, with ID as a deterministic tie-breaker. Duplicates are never automatically trashed or merged. This is an experimental policy, not a production adapter implementation or proof of cross-instance locking/eventual list consistency. A production per-account lock must be shared across session-scoped stores; multiple server instances still need convergence behavior and preservation of data in duplicate folders.

Share counts and boolean flags, without callback URLs, account details, or tokens. Record live evidence in `docs/development/findings.md`. You may inspect and manually trash only these empty, clearly identified test trees when finished; no automatic cleanup is performed.
