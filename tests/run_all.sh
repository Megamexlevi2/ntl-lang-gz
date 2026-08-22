#!/usr/bin/env bash

set -u

if [ $# -lt 1 ]; then
  echo "Usage: bash tests/run_all.sh ./path/to/lunex"
  echo "Error: you must provide an explicit path starting with ./ or /"
  exit 1
fi

LUNEX="$1"

case "$LUNEX" in
  ./*|/*)
    ;;
  *)
    echo "Error: invalid binary path '$LUNEX'"
    echo "You must use an explicit path like: ./lunex or /usr/local/bin/lunex"
    exit 1
    ;;
esac

if [ ! -f "$LUNEX" ]; then
  echo "Error: binary not found: $LUNEX"
  exit 1
fi

if [ ! -x "$LUNEX" ]; then
  echo "Info: adding execute permission to $LUNEX"
  chmod +x "$LUNEX"
fi

if [ ! -x "$LUNEX" ]; then
  echo "Error: binary is still not executable: $LUNEX"
  exit 1
fi

PASS=0
FAIL=0
ERRORS=""

run_test() {
  file="$1"
  name=$(basename "$file" .lx)
  output=$("$LUNEX" run "$file" 2>&1)
  exitcode=$?

  if [ $exitcode -eq 0 ] && echo "$output" | grep -q "PASS"; then
    printf "  \033[32m✓\033[0m  %s\n" "$name"
    PASS=$((PASS + 1))
  else
    printf "  \033[31m✗\033[0m  %s\n" "$name"
    FAIL=$((FAIL + 1))
    ERRORS="$ERRORS\n    $file"

    first_err=$(echo "$output" | grep -i "error\|type" | head -1)
    [ -n "$first_err" ] && printf "        \033[90m%s\033[0m\n" "$first_err"
  fi
}

echo ""
echo "\033[1mLunex Test Suite\033[0m"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

BASE="$(cd "$(dirname "$0")" && pwd)"

categories=(
  variables
  functions
  control_flow
  loops
  structs
  "stdlib/io"
  "stdlib/math"
  "stdlib/utils"
  "stdlib/json"
  "stdlib/datetime"
  "stdlib/crypto"
  "stdlib/fs"
  "stdlib/os"
  "stdlib/regex"
  "stdlib/db"
  "stdlib/http"
  "stdlib/ints"
  "stdlib/buffer"
  concurrency
  advanced
)

for category in "${categories[@]}"; do
  dir="$BASE/$category"

  if [ -d "$dir" ]; then
    echo ""
    printf "\033[1m%s\033[0m\n" "  $category"

    for f in "$dir"/*.lx; do
      [ -f "$f" ] && run_test "$f"
    done
  fi
done

TOTAL=$((PASS + FAIL))
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
printf "  passed : \033[32m%d\033[0m / %d\n" "$PASS" "$TOTAL"
printf "  failed : \033[31m%d\033[0m / %d\n" "$FAIL" "$TOTAL"
echo ""

if [ "$FAIL" -gt 0 ]; then
  printf "  \033[31mFailed tests:\033[0m$ERRORS\n"
  echo ""
  exit 1
fi

printf "  \033[32m✓ All %d tests passed.\033[0m\n" "$PASS"
echo ""