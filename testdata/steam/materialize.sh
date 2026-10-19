#!/usr/bin/env bash
# Materializes the fixture templates into a temporary Steam-like tree.
# Usage: bash testdata/steam/materialize.sh [destination]
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$HERE/fixtures"
DEST="${1:-$(mktemp -d /tmp/better-rpt-fixtures.XXXXXX)}"

if [[ ! -d "$SRC" ]]; then
	echo "fixtures not generated; run: bash testdata/steam/generate.sh" >&2
	exit 1
fi

mkdir -p "$DEST"
cp -a "$SRC/." "$DEST/"

sed \
	-e "s|{{PRIMARY}}|$DEST/primary|g" \
	-e "s|{{SECONDARY}}|$DEST/secondary|g" \
	"$DEST/primary/steamapps/libraryfolders.vdf.tmpl" \
	> "$DEST/primary/steamapps/libraryfolders.vdf"

sed \
	-e "s|{{LEGACY}}|$DEST/legacy|g" \
	"$DEST/legacy/steamapps/libraryfolders.vdf.tmpl" \
	> "$DEST/legacy/steamapps/libraryfolders.vdf"

echo "$DEST"
