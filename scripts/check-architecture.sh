#!/bin/sh
# Prove depguard rejects forbidden production imports, then remove each probe.
set -eu
cd "$(dirname "$0")/.."
canary=
trap 'if [ -n "$canary" ]; then rm -f "$canary"; fi' EXIT HUP INT TERM
while read -r file package forbidden; do
  canary="$file"
  if [ -e "$canary" ]; then
    echo "Refusing to overwrite $canary" >&2
    canary=
    exit 1
  fi
  printf 'package %s\n\nimport _ "%s"\n' "$package" "$forbidden" > "$canary"
  gofmt -w "$canary"
  if result=$(golangci-lint run 2>&1); then
    echo "FAIL: depguard allowed $forbidden in $canary" >&2
    exit 1
  fi
  case "$result" in
    *"$canary"*"$forbidden"*"(depguard)"*) echo "PASS: $canary rejects $forbidden" ;;
    *) printf 'FAIL: probe failed for another reason:\n%s\n' "$result" >&2; exit 1 ;;
  esac
  rm "$canary"
  canary=
done <<'CASES'
internal/journal/architecture_canary.go journal net/http
internal/journal/storetest/architecture_canary.go storetest net/http/httptest
internal/web/architecture_canary.go web github/mridha/myjournal-app/internal/provider/local
internal/web/handler/architecture_canary.go handler github/mridha/myjournal-app/internal/clock
internal/session/architecture_canary.go session github/mridha/myjournal-app/internal/cachestore
internal/provider/local/architecture_canary.go local github/mridha/myjournal-app/internal/web/api
internal/provider/google/architecture_canary.go google github/mridha/myjournal-app/internal/web/api
CASES
