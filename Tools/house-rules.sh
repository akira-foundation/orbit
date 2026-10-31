#!/bin/bash
# Checks the conventions no off the shelf linter knows about.
#
#   ./Tools/house-rules.sh               check this repository
#   ./Tools/house-rules.sh --self-test   check the rules against planted files
#
# Prints every finding as `file:line: rule` and exits 1 when there is any.

set -euo pipefail
export LC_ALL=C

em_dash="$(printf '\342\200\224')"
en_dash="$(printf '\342\200\223')"
old_stack="Wa""ils"

swift_files() {
  local root="$1"
  shift
  local dir
  for dir in "$@"; do
    [ -d "$root/$dir" ] || continue
    find "$root/$dir" -type f -name '*.swift' -not -path '*/.build/*' -print0
  done
}

prose_files() {
  local root="$1"
  local dir
  for dir in Sources Tests Tools docs .github; do
    [ -d "$root/$dir" ] || continue
    find "$root/$dir" -path "$root/docs/superpowers" -prune -o -path '*/.build' -prune -o -type f -print0
  done
  find "$root" -maxdepth 1 -type f \( -name '*.md' -o -name 'Makefile' -o -name 'Package.swift' \
    -o -name '*.yml' -o -name '*.toml' \) -print0
}

relative() {
  local root="$1"
  sed "s|^$root/||"
}

findings_as() {
  local rule="$1"
  cut -d: -f1,2 | sed "s|\$|: $rule|"
}

rule_ui_import() {
  local root="$1"
  swift_files "$root" Sources Tests \
    | xargs -0 grep -nHE '^[[:space:]]*(@[A-Za-z]+[[:space:]]+)*import[[:space:]]+([a-z]+[[:space:]]+)?(SwiftUI|AppKit)([^A-Za-z0-9_]|$)' /dev/null \
    | relative "$root" \
    | grep -vE '^(Sources/(Orbit|OrbitDesign)|Tests/OrbitDesignTests)/' \
    | findings_as ui-import || true
}

rule_dashes() {
  local root="$1"
  prose_files "$root" | xargs -0 grep -nHI -e "$em_dash" -e "$en_dash" /dev/null \
    | relative "$root" | findings_as dashes || true
}

rule_banned_folders() {
  local root="$1"
  local dir
  for dir in Sources Tests; do
    [ -d "$root/$dir" ] || continue
    find "$root/$dir" -type f \( -path '*/Helpers/*' -o -path '*/Extensions/*' \)
  done | relative "$root" | sed 's|$|:1: banned-folders|'
}

rule_view_subprocess() {
  local root="$1"
  local file
  swift_files "$root" Sources | while IFS= read -r -d '' file; do
    grep -qE '^[[:space:]]*([a-z]+[[:space:]]+)*(struct|class|extension)[[:space:]][^{]*:[^{]*(^|[^A-Za-z0-9_])View([^A-Za-z0-9_]|$)' "$file" || continue
    grep -nHE '(^|[^A-Za-z0-9_])Process(\(|\.run)' "$file" | relative "$root" | findings_as view-subprocess || true
  done
}

rule_old_stack() {
  local root="$1"
  prose_files "$root" | xargs -0 grep -nHIi -e "$old_stack" /dev/null \
    | relative "$root" | findings_as old-stack || true
}

run_rules() {
  local root="$1"
  rule_ui_import "$root"
  rule_dashes "$root"
  rule_banned_folders "$root"
  rule_view_subprocess "$root"
  rule_old_stack "$root"
}

self_test_failures=0

report_case() {
  local outcome="$1" rule="$2" path="$3" output="$4"
  if [ "$outcome" = fail ]; then
    printf '%s\n' "$output" | grep -q "^$path:[0-9]*: $rule\$" && return 0
    echo "missed: $rule should flag $path"
    return 1
  fi
  [ -z "$output" ] && return 0
  echo "false positive: $path is clean but got:"
  printf '  %s\n' "$output"
  return 1
}

expect() {
  local outcome="$1" rule="$2" path="$3" content="$4"
  local tree output
  tree="$(mktemp -d)"
  mkdir -p "$tree/$(dirname "$path")"
  printf '%s\n' "$content" > "$tree/$path"
  output="$(run_rules "$tree")"
  rm -rf "$tree"
  if report_case "$outcome" "$rule" "$path" "$output"; then
    echo "ok $outcome $rule $path"
    return 0
  fi
  self_test_failures=$((self_test_failures + 1))
}

self_test() {
  expect fail ui-import Sources/Core/X.swift 'import SwiftUI'
  expect pass ui-import Sources/Orbit/X.swift 'import SwiftUI'
  expect fail ui-import Sources/orbit-agent/X.swift 'import AppKit'
  expect pass ui-import Sources/OrbitDesign/X.swift 'import SwiftUI'
  expect fail ui-import Sources/Core/Y.swift '@preconcurrency import AppKit'
  expect fail ui-import Tests/CoreTests/X.swift 'import struct SwiftUI.Color'
  expect pass ui-import Tests/OrbitDesignTests/X.swift 'import SwiftUI'
  expect fail dashes Sources/Core/X.swift "let x = 1 // a $em_dash b"
  expect fail dashes README.md "a $en_dash b"
  expect pass dashes docs/superpowers/specs/x.md "a $em_dash b"
  expect fail banned-folders Sources/Core/Helpers/X.swift 'struct X {}'
  expect fail banned-folders Sources/Orbit/Extensions/X.swift 'struct X {}'
  expect pass banned-folders Sources/Core/Projects/X.swift 'struct X {}'
  expect fail view-subprocess Sources/Orbit/Views/X.swift "$(printf 'struct X: View {\n    let p = Process()\n}')"
  expect pass view-subprocess Sources/Core/Runtime/X.swift 'let p = Process()'
  expect fail old-stack Sources/Core/X.swift "let x = \"$old_stack\""
  expect pass old-stack docs/superpowers/specs/x.md "$old_stack"
  if [ "$self_test_failures" -gt 0 ]; then
    echo "$self_test_failures self-test cases failed"
    exit 1
  fi
  echo "Self-test passes."
}

if [ "${1:-}" = "--self-test" ]; then
  self_test
  exit 0
fi

cd "$(dirname "$0")/.."
findings="$(run_rules "$PWD")"
if [ -n "$findings" ]; then
  printf '%s\n' "$findings"
  echo "$(printf '%s\n' "$findings" | wc -l | tr -d ' ') house rule violations"
  exit 1
fi
echo "House rules pass."
