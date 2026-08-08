#!/usr/bin/env bash

set -euo pipefail

workspace="${GITHUB_WORKSPACE:-$PWD}"
runner_temp="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"

absolute_path() {
	local path="$1"
	if [[ "$path" = /* ]]; then
		printf '%s\n' "$path"
	else
		printf '%s/%s\n' "${workspace%/}" "$path"
	fi
}

linter_directory="$(absolute_path "$INPUT_LINTER_DIRECTORY")"
working_directory="$(absolute_path "$INPUT_WORKING_DIRECTORY")"

if [[ ! -d "$linter_directory" ]]; then
	echo "Glint linter directory does not exist: $linter_directory" >&2
	exit 1
fi
if [[ ! -d "$working_directory" ]]; then
	echo "Glint working directory does not exist: $working_directory" >&2
	exit 1
fi

binary_directory="$(mktemp -d "${runner_temp%/}/glint-action.XXXXXX")"
binary="$binary_directory/glint"

go -C "$linter_directory" build -trimpath -o "$binary" "$INPUT_LINTER_PACKAGE"

arguments=()
while IFS= read -r argument; do
	if [[ -n "$argument" ]]; then
		arguments+=("$argument")
	fi
done <<< "$INPUT_ARGUMENTS"

cd "$working_directory"
"$binary" "${arguments[@]}"
