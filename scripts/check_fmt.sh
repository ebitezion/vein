#!/usr/bin/env bash
set -euo pipefail

unformatted=$(gofmt -l .)
if [[ -n "$unformatted" ]]; then
  echo "gofmt found unformatted files:"
  echo "$unformatted"
  exit 1
fi

echo "gofmt check passed"
