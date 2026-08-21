#!/usr/bin/env bash
set -euo pipefail

while IFS= read -r file; do
  perl -0pi -e 's/\n+\z/\n/' "$file"
done < <(find gen -type f \( -name '*.go' -o -name '*.ts' \) -print | sort)
