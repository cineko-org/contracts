#!/usr/bin/env bash
set -euo pipefail

if rg -n '^\s*(syntax|package)\s*=.*v[0-9]|protocol(_version)?|schema_version' proto; then
  echo 'Protocol and schema versions are forbidden in Cineko contracts.' >&2
  exit 1
fi

if rg -n '^\s*reserved\b' proto; then
  echo 'Reserved declarations are forbidden; Cineko performs one coordinated latest-contract cutover.' >&2
  exit 1
fi

if rg -n '^\s*enum\s+' proto; then
  echo 'Semantic enums are forbidden; model required alternatives with oneof messages.' >&2
  exit 1
fi

if rg -n 'google\.protobuf\.(Any|Struct|Value|ListValue)' proto; then
  echo 'Untyped protobuf payloads are forbidden at service boundaries.' >&2
  exit 1
fi

while IFS= read -r file; do
  if ! rg -q '^edition = "2024";' "$file"; then
    echo "$file must use Protobuf Edition 2024." >&2
    exit 1
  fi
done < <(rg --files proto -g '*.proto')
