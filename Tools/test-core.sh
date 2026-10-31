#!/bin/bash
# Runs the Core suite in a mirror package with no app target, so a broken view cannot stop it.
#
#   ./Tools/test-core.sh                         run everything
#   ./Tools/test-core.sh OrbitEnvironmentTests   run the suites matching each filter given
#   ORBIT_TEST_SWIFT_ARGS="--sanitize=thread" ./Tools/test-core.sh   pass extra flags to swift test

set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
scratch="$root/.build/core-tests"
mkdir -p "$root/.build"
mirror="$(mktemp -d "$root/.build/core-mirror.XXXXXX")"
trap 'rm -rf "$mirror"' EXIT INT TERM HUP

mkdir -p "$mirror/Sources" "$mirror/Tests"
ln -s "$root/Sources/Core" "$mirror/Sources/Core"
ln -s "$root/Tests/CoreTests" "$mirror/Tests/CoreTests"

cat > "$mirror/Package.swift" <<'MANIFEST'
// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "CoreOnly",
    platforms: [.macOS("27.0")],
    targets: [
        .target(name: "Core", swiftSettings: [.swiftLanguageMode(.v6)]),
        .testTarget(name: "CoreTests", dependencies: ["Core"], swiftSettings: [.swiftLanguageMode(.v6)]),
    ]
)
MANIFEST

args=()
for filter in "$@"; do
  args+=(--filter "$filter")
done
read -r -a extra <<< "${ORBIT_TEST_SWIFT_ARGS:-}"

cd "$mirror"
swift test --scratch-path "$scratch" ${extra[@]+"${extra[@]}"} ${args[@]+"${args[@]}"}
