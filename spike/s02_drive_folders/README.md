# S02: Google Drive folder behavior

This throwaway spike validates private `appProperties` lookups and how `drive.file` sees app-created versus manually created Drive items. Use a dedicated test account. The spike creates a uniquely named test root and child folders; it does not delete anything automatically.

The OAuth redirect is `http://localhost:3030/auth/google/callback`, matching S01. Stop any other local server using port 3030 before starting it.

## Start

From the repository root, load `.env` and start create mode:

```bash
set -a
source ./.env
set +a
go run ./spike/s02_drive_folders -mode=create
```

Open `http://localhost:3030/login`. After Google redirects back, create mode creates:

- `myjournal-s2-<run-id>` at My Drive root with `appProperties.role=root`
- `entries` below it with `appProperties.role=entries`
- `2026` below `entries` with `appProperties.role=year` and `appProperties.year=2026`

The callback checks that each folder can be found by its app properties and expected parent, then prints the run ID, folder IDs, and generated root name. These IDs are Drive resource identifiers, not credentials. Keep them locally for the next commands.

## Verify rename behavior

Rename the generated root folder in Drive's UI. Stop the spike and restart it in verify mode using the values from create mode:

```bash
go run ./spike/s02_drive_folders \
  -mode=verify \
  -run-id '<RUN_ID>' \
  -root-id '<ROOT_ID>' \
  -entries-id '<ENTRIES_ID>' \
  -year-id '<YEAR_ID>'
```

Open `http://localhost:3030/login` and complete sign-in. The callback should report `metadata_lookup_ids_match: true` and the renamed root's current name. That confirms the role lookup found the same IDs after rename.

## Verify trash detection

Complete the manual-item visibility check below before trashing `entries`, then return here. The earlier unguarded recreation attempt may have left two `entries` children; see the cleanup note below before changing either one.
In Drive's UI, move the current test root's `entries` child folder to trash (renaming the root alone does not trash `entries`). Restart with `-mode=check-trash` and the same IDs/run ID, then sign in. Expected output:

```text
entries_absent_from_active_query: true
entries_found_in_trashed_query: true
```

This checks detection only; the production store's recreate-on-write behavior is not implemented by this spike. Restore the folder from Drive trash if you want to continue using this test tree, or trash the test root and its contents when finished.

If the terminal reports active entries folders were found, do not run recreation yet: inspect the test root and trash the intended `entries` child first. An earlier version of this harness could create a duplicate during a premature `recreate-entries` attempt. If you now see two `entries` children, inspect their `2026` children and only trash the confirmed empty duplicate created by that failed attempt; preserve any folder containing test data.

To verify the Drive API can recreate the missing path, restart with `-mode=recreate-entries` and the same IDs/run ID. It creates a replacement `entries` folder and `2026` year folder under the existing root, then looks both up by their app properties. Expected output:

```text
metadata_lookup_ids_match: true
entries_recreated_with_new_id: true
year_recreated_with_new_id: true
```

This validates the folder operation needed for recreate-on-write; the real Google Store must still implement that behavior.

## Verify manual-item visibility

Run this check before the trash test above. Create a **manual duplicate folder** in My Drive using the root's current name (including your rename suffix). Also create a text file manually inside the original generated `2026` folder, named `s2-manual-<RUN_ID>.txt`.

Restart with `-mode=check-visibility`, passing the same IDs and run ID plus the exact manual filename:

```bash
go run ./spike/s02_drive_folders \
  -mode=check-visibility \
  -run-id '<RUN_ID>' \
  -root-id '<ROOT_ID>' \
  -entries-id '<ENTRIES_ID>' \
  -year-id '<YEAR_ID>' \
  -manual-file-name 's2-manual-<RUN_ID>.txt'
```

After signing in, expected output is `visible_root_matches_app_created_root_only: true` and `manual_file_invisible_to_app: true`. If the manual folder or file appears in the app's query, record that result instead; it falsifies the visibility assumption and should change the help text/design.

The earlier `recreate-entries` attempt may have created a second `entries` folder before failing. In Drive, inspect the children of the test root. Compare each folder's ID (shown in the Drive URL when opened) to the original `entries_id` and inspect the contained year folder against the original `year_id`. Preserve the original folders and any test data; move only the confirmed empty duplicate from that failed attempt to Drive trash.

## Verify moved year folder (optional)

Move the generated `2026` folder to another parent in Drive, then run `-mode=check-moved-year` with the same IDs/run ID. The metadata-only query should still locate the same year folder ID. This mode reports lookup behavior; it does not claim the moved folder still belongs to the journal's intended hierarchy.

## Record findings

Record each result and the Google API error/status if a query fails in `docs/development/findings.md`. Do not record authorization URLs, OAuth codes, tokens, file contents, or secrets. A zero-file result is not itself failure; judge each query by the expected folder IDs and properties.

The experiment listens only on IPv4 loopback (127.0.0.1:3030). Pending OAuth logins expire after ten minutes and are capped at 128; a full store returns HTTP 429 until a callback consumes a login or transactions expire.
