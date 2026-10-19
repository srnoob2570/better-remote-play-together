#!/bin/sh
# Non-interactive warning only. The package never edits Steam configuration.
if command -v pgrep >/dev/null 2>&1 && pgrep -x better-rpt >/dev/null 2>&1; then
    echo "Better RPT está en ejecución. Ciérralo y restaura SpaceWar antes de desinstalar. Si preparaste una sesión y ya no está instalado, reinstala esta misma versión para restaurar." >&2
fi
exit 0
