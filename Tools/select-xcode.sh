#!/bin/zsh
set -euo pipefail

selected=""
for candidate in $(find /Applications -maxdepth 1 -name 'Xcode_27*.app' | sort -V); do
  target="$(readlink -f "$candidate")"
  echo "$candidate -> $target"
  case "$target:l" in
    *beta* | *release_candidate*) continue ;;
  esac
  selected="$candidate"
done

if [[ -n "$selected" ]]; then
  echo "Selecting $selected"
  sudo xcode-select -s "$selected/Contents/Developer"
fi

version="$(xcodebuild -version)"
echo "$version"
major="$(printf '%s\n' "$version" | sed -n '1s/^Xcode \([0-9]*\).*/\1/p')"
if [[ -z "$major" || "$major" -lt 27 ]]; then
  echo "::error::Xcode 27 or newer is required, this runner has: $version"
  exit 1
fi
developer="$(readlink -f "$(xcode-select -p)")"
if printf '%s\n' "$developer" | grep -qiE 'beta|release_candidate'; then
  echo "::error::Only beta or release candidate Xcode 27 builds are installed: $developer"
  exit 1
fi
