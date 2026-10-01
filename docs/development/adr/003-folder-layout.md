# 003: Store entries by date and year with Drive appProperties

Entries will live in `myjournal/entries/<year>/YYYY-MM-DD.txt` so files remain readable and portable. The app will use Drive folder properties to identify root, entries, and year folders, which keeps the app resilient to user-driven renames while respecting the `drive.file` scope.
