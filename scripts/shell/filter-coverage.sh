#!/bin/sh
# Remove explicitly listed packages and files from a Go coverage profile.
set -eu

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <ignore-file> <coverage-profile>" >&2
    exit 1
fi

ignore_file=$1
profile=$2
temporary=$(mktemp "${profile}.XXXXXX")
trap 'rm -f "$temporary"' EXIT HUP INT TERM

awk '
    FILENAME == ARGV[1] {
        line = $0
        sub(/<!--.*-->/, "", line)
        sub(/#.*/, "", line)
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
        if (line != "") {
            excluded[line] = 1
        }
        next
    }
    /^mode:/ {
        print
        next
    }
    {
        file = $1
        sub(/:.*/, "", file)
        package = file
        sub(/\/[^\/]*$/, "", package)
        if (!(package in excluded) && !(file in excluded)) {
            print
        }
    }
' "$ignore_file" "$profile" > "$temporary"

mv "$temporary" "$profile"
