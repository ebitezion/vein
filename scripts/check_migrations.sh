#!/usr/bin/env bash
set -euo pipefail

missing=0
for up in migrations/*.up.sql; do
  down="${up/.up.sql/.down.sql}"
  if [[ ! -f "$down" ]]; then
    echo "missing down migration for $up"
    missing=1
  fi

  if [[ ! -s "$up" ]]; then
    echo "empty migration file: $up"
    missing=1
  fi
done

if [[ $missing -ne 0 ]]; then
  exit 1
fi

echo "migration safety checks passed"
