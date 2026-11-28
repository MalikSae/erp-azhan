#!/usr/bin/env bash
# Bangun skema dari semua file migrations/*.sql (urut nama, nomor ganda seperti 035_* ikut urutan abjad)
# lalu isi data minimal untuk integration test. Kredensial dibaca runner dari env DB_*.
set -euo pipefail
cd "$(dirname "$0")/../.."
bin="${MIGRATE_BIN:-${RUNNER_TEMP:-/tmp}/migrate}"
go build -o "$bin" migrations/run.go
log="$(mktemp)"
for f in $(ls migrations/*.sql | LC_ALL=C sort) .github/ci/seed.sql; do
  if ! "$bin" "$f" >"$log" 2>&1; then
    echo "::error file=$f::migrasi gagal"
    tail -n 20 "$log"
    exit 1
  fi
  echo "ok  $f"
done
