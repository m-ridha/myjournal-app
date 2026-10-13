#!/bin/sh
# Prove depguard rejects forbidden production imports, then remove each probe.
set -eu
cd "$(dirname "$0")/../.."
canary=
# Dedicated nested packages keep probes acyclic and verify recursive matching.
probe_dirs="internal/driven/cache/journal/architectureprobe internal/core/journal/architectureprobe internal/core/connection/architectureprobe internal/driver/web/architectureprobe internal/driven/provider/local/architectureprobe internal/driven/cache/local/lru/architectureprobe internal/config/architectureprobe"
created_dirs=
cleanup() {
  if [ -n "$canary" ]; then rm -f "$canary"; fi
  for dir in $created_dirs; do rmdir "$dir"; done
}
trap cleanup EXIT HUP INT TERM
for dir in $probe_dirs; do
  mkdir "$dir"
  created_dirs="$dir $created_dirs"
done
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
internal/core/journal/architecture_canary.go journal net/http
internal/core/journal/storetest/architecture_canary.go storetest net/http/httptest
internal/driver/web/architecture_canary.go web github/mridha/myjournal-app/internal/driven/provider/local
internal/driver/web/handler/architecture_canary.go handler github/mridha/myjournal-app/internal/driven/clock
internal/driven/session/architecture_canary.go session github/mridha/myjournal-app/internal/driven/cache/journal
internal/driven/provider/local/architecture_canary.go local github/mridha/myjournal-app/internal/driver/web/api
internal/driven/provider/google/architecture_canary.go google github/mridha/myjournal-app/internal/driver/web/api
internal/core/journal/architecture_canary.go journal github.com/allegro/bigcache
internal/core/journal/storetest/architecture_canary.go storetest github/mridha/myjournal-app/internal/driven/cache/local/bigcache
internal/driver/web/handler/architecture_canary.go handler github/mridha/myjournal-app/internal/driven/cache/local/bigcache
internal/driven/session/architecture_canary.go session github.com/allegro/bigcache
internal/driven/cache/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/cache/local/bigcache
internal/driven/cache/journal/architecture_canary.go cachestore github.com/allegro/bigcache
internal/driven/cache/local/bigcache/architecture_canary.go bigcache github/mridha/myjournal-app/internal/driver/web/api
internal/driven/cache/local/lru/architecture_canary.go lru github/mridha/myjournal-app/internal/driver/web/api
internal/core/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/core/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/core/connection
internal/core/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driver/web/api
internal/core/connection/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/session
internal/core/connection/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/core/connection/architectureprobe/architecture_canary.go architectureprobe net/http
internal/driver/web/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/driver/web/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/cache/local/lru
internal/driven/cache/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/driven/cache/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driver/web/api
internal/driven/provider/local/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/driven/provider/local/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/session
internal/driven/cache/local/lru/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driver/web/api
internal/driven/cache/local/lru/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/config
internal/core/journal/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/testutil
internal/config/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/provider/local
internal/driver/web/authsession/architecture_canary.go authsession github/mridha/myjournal-app/internal/driven/session
internal/driver/web/handler/architecture_canary.go handler github/mridha/myjournal-app/internal/driven/session
internal/driver/web/architectureprobe/architecture_canary.go architectureprobe github/mridha/myjournal-app/internal/driven/session
internal/core/connection/architectureprobe/architecture_canary.go architectureprobe github.com/stretchr/testify/assert
internal/core/connection/architectureprobe/architecture_canary_test.go architectureprobe github/mridha/myjournal-app/internal/driven/session
CASES
