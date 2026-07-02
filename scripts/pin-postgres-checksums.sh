#!/usr/bin/env bash
set -euo pipefail
for ver in 18.4.0 17.8.0 16.10.0 15.14.0; do
  for triple in aarch64-apple-darwin x86_64-apple-darwin aarch64-unknown-linux-gnu x86_64-unknown-linux-gnu; do
    url="https://github.com/theseus-rs/postgresql-binaries/releases/download/${ver}/postgresql-${ver}-${triple}.tar.gz.sha256"
    sum=$(curl -fsSL "$url" | awk '{print $1}')
    echo "${ver} ${triple} ${sum}"
  done
done
