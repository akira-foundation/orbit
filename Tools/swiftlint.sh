#!/bin/bash
# Runs SwiftLint strictly over Sources and Tests against .swiftlint.yml.
#
#   ./Tools/swiftlint.sh          lint everything
#   ./Tools/swiftlint.sh --fix    apply SwiftLint's own corrections, then read the diff

set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v swiftlint >/dev/null 2>&1; then
  echo "SwiftLint is not on the PATH. Install it with: brew install swiftlint"
  exit 1
fi

echo "==> swiftlint $(swiftlint version)"

if [ "${1:-}" = "--fix" ]; then
  shift
  swiftlint lint --fix "$@"
  exit 0
fi

swiftlint lint --strict --quiet "$@"
echo "SwiftLint passes."
