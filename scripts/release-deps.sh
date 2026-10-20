#!/bin/sh
# SPDX-License-Identifier: GPL-2.0-or-later
# Usage: release-deps.sh [tag]   (default v0.1.0)
# Run once at publish time, after the engine repo's tags exist: drops every
# `github.com/stealthmonkey99/arkade...` replace from each cartridge go.mod,
# pins the matching requires to <tag>, and runs go mod tidy.
set -eu
tag="${1:-v0.1.0}"
cd "$(dirname "$0")/.."

for mod in cartridges/*/go.mod; do
	dir=$(dirname "$mod")
	for path in $(awk '$1 ~ /^github\.com\/stealthmonkey99\/arkade/ && $2 == "=>" { print $1 }
		$1 == "replace" && $2 ~ /^github\.com\/stealthmonkey99\/arkade/ { print $2 }' "$mod"); do
		go -C "$dir" mod edit -dropreplace="$path" -require="$path@$tag"
	done
	go -C "$dir" mod tidy
done
