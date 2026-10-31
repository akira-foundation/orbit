#!/bin/bash
# Formats the Swift sources with swift-format against .swift-format.
#
#   ./Tools/format.sh           rewrite the sources in place
#   ./Tools/format.sh --check   fail when a source is not formatted

set -euo pipefail
cd "$(dirname "$0")/.."

swift_format_command() {
  if command -v xcrun >/dev/null 2>&1; then
    swift_format=(xcrun swift-format)
    return 0
  fi
  if command -v swift-format >/dev/null 2>&1; then
    swift_format=(swift-format)
    return 0
  fi
  echo "swift-format is not available. It ships with Xcode and the Swift toolchain."
  exit 1
}

swift_format_command

sources=(Sources Tests Package.swift)

if [ "${1:-}" = "--check" ]; then
  "${swift_format[@]}" lint --strict --recursive --parallel "${sources[@]}"
  echo "Formatting passes."
else
  "${swift_format[@]}" format --in-place --recursive --parallel "${sources[@]}"
fi
