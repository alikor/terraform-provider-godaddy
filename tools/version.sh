#!/usr/bin/env sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
base_version=$(tr -d '[:space:]' < "$repo_root/VERSION")

if [ -z "$base_version" ]; then
  echo "VERSION file is empty" >&2
  exit 1
fi

if [ "${GITHUB_REF_TYPE:-}" = "tag" ] && [ -n "${GITHUB_REF_NAME:-}" ]; then
  printf '%s\n' "${GITHUB_REF_NAME#v}"
  exit 0
fi

if [ -n "${GITHUB_RUN_NUMBER:-}" ]; then
  printf '%s-build.%s\n' "$base_version" "$GITHUB_RUN_NUMBER"
  exit 0
fi

printf '%s-dev\n' "$base_version"
