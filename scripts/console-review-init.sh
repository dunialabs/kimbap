#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACT_DIR="${1:-$ROOT_DIR/artifacts/console-review}"
TEMPLATE="$ROOT_DIR/docs/console-review-round-template.yaml"
LEDGER="$ARTIFACT_DIR/ledger.csv"

artifact_abs="$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$ARTIFACT_DIR")"
root_abs="$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$ROOT_DIR")"

if [ "$artifact_abs" = "/" ] || [ "$artifact_abs" = "$root_abs" ] || [ "$(basename "$artifact_abs")" != "console-review" ]; then
  printf 'Refusing unsafe artifact path: %s\n' "$artifact_abs" >&2
  exit 1
fi

if [ ! -f "$TEMPLATE" ]; then
  printf 'Template not found: %s\n' "$TEMPLATE" >&2
  exit 1
fi

if [ -e "$LEDGER" ] || [ -e "$ARTIFACT_DIR/R001/report.yaml" ]; then
  if [ "${FORCE:-0}" != "1" ]; then
    printf 'Artifacts already exist at %s. Re-run with FORCE=1 to overwrite.\n' "$ARTIFACT_DIR" >&2
    exit 1
  fi

  rm -rf "$ARTIFACT_DIR"
fi

mkdir -p "$ARTIFACT_DIR"
mkdir -p "$ARTIFACT_DIR/screens"

printf 'round,status,report_path,next_round\n' > "$LEDGER"

for n in $(seq 1 100); do
  round=$(printf 'R%03d' "$n")
  next_round=""
  if [ "$n" -eq 100 ]; then
    next_round="WRAPUP"
  else
    next_round=$(printf 'R%03d' "$((n+1))")
  fi

  round_dir="$ARTIFACT_DIR/$round"
  report_path="$round_dir/report.yaml"
  mkdir -p "$round_dir/screens"

  sed -e "s/^round: .*/round: $round/" \
      -e "s/^status: .*/status: queued/" \
      -e "s/^  id: .*/  id: $next_round/" \
      "$TEMPLATE" > "$report_path"

  printf '%s,queued,%s,%s\n' "$round" "$report_path" "$next_round" >> "$LEDGER"
done

wrapup_dir="$ARTIFACT_DIR/WRAPUP"
wrapup_report="$wrapup_dir/report.yaml"
mkdir -p "$wrapup_dir"
cat > "$wrapup_report" <<'EOF'
round: WRAPUP
status: queued
summary: ""
final_checks:
  total_rounds: 100
  completed_rounds: 0
  ledger_verified: false
  oracle_verified: false
EOF
printf '%s,%s,%s,%s\n' "WRAPUP" "queued" "$wrapup_report" "" >> "$LEDGER"

printf 'Initialized console review artifacts at %s\n' "$ARTIFACT_DIR"
printf 'Ledger: %s\n' "$LEDGER"
