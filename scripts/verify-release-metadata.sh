#!/usr/bin/env bash
set -euo pipefail

readonly version="$(tr -d '[:space:]' < VERSION)"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "VERSION must contain a semantic version" >&2
  exit 1
}

jq -e --arg version "$version" '.["."] == $version' .release-please-manifest.json >/dev/null
grep -Fq "## [$version]" CHANGELOG.md
