#!/usr/bin/env bash
set -euo pipefail

mode="${1:-all}"

tidy_mod() {
  mod_dir="$1"
  tidy_mode="$2"

  (
    cd "$mod_dir"
    go_version="$(go mod edit -json | awk -F'"' '/"Go":/ { print $4; exit }' | tr -d '[:blank:]')"
    if [ -z "$go_version" ]; then
      echo >&2 "Go version not found in ${mod_dir}/go.mod"
      exit 1
    fi

    case "$tidy_mode" in
      relaxed)
        go mod tidy -e -compat="$go_version"
        ;;
      strict)
        go mod tidy -compat="$go_version"
        ;;
      *)
        echo >&2 "Unknown tidy mode: $tidy_mode"
        exit 1
        ;;
    esac
  )
}

work_dirs() {
  if [ -f go.work ]; then
    go work edit -json | awk -F'"' '/"DiskPath":/ { print $4 }'
  else
    find . -name go.mod -not -path "*/vendor/*" -exec dirname {} \; | sort -u
  fi
}

run_tidy_mode() {
  tidy_mode="$1"
  found_work_dir=false

  while IFS= read -r mod_dir; do
    if [ -z "$mod_dir" ]; then
      continue
    fi

    found_work_dir=true
    tidy_mod "$mod_dir" "$tidy_mode"
  done < <(work_dirs)

  if [ "$found_work_dir" = "false" ]; then
    echo >&2 "No Go modules found"
    exit 1
  fi
}

case "$mode" in
  all)
    run_tidy_mode relaxed
    run_tidy_mode strict
    ;;
  relaxed | strict)
    run_tidy_mode "$mode"
    ;;
  *)
    echo >&2 "Usage: $0 [all|relaxed|strict]"
    exit 1
    ;;
esac
