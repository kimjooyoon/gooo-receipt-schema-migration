#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: measure.sh output command [args...]" >&2
  exit 2
fi

output=$1
shift
raw=$(mktemp)
trap 'rm -f "$raw"' EXIT
/usr/bin/time -f '%e %M' -o "$raw" "$@"
awk '{printf "wall_ms=%d peak_rss_kib=%d\n", ($1 * 1000 + 0.5), $2}' "$raw" > "$output"
