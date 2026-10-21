#!/usr/bin/env bash
# Arranca `wails3 dev` sin dejar procesos de fondo huérfanos.
#
# Wails ejecuta `wails3 task dev:frontend` como proceso de fondo en su propio
# grupo y refresh lo limpia con SIGINT/SIGTERM, pero no captura SIGHUP. Si se
# cierra el terminal, wails3 muere y Vite sigue escuchando en el puerto 9245;
# además, la app anterior conserva el bloqueo de instancia única. El siguiente
# `make dev` falla. Este envoltorio captura SIGHUP, para wails3 y libera los
# restos si algo sobrevive. Antes de arrancar también retira los procesos
# huérfanos de ejecuciones anteriores de este repositorio.
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
port=9245
mode="${1:-}"

case "$mode" in
	"" | --stop) ;;
	*)
		printf 'uso: %s [--stop]\n' "$0" >&2
		exit 2
		;;
esac

command -v ss >/dev/null 2>&1 || {
	printf 'error: se requiere "ss" (iproute2) para comprobar el puerto %s.\n' "$port" >&2
	exit 1
}

wails3=""
if [ "$mode" != --stop ]; then
	wails3="${WAILS3:-}"
	if [ -z "$wails3" ]; then
		if command -v wails3 >/dev/null 2>&1; then
			wails3="$(command -v wails3)"
		else
			wails3="$(go env GOPATH)/bin/wails3"
		fi
	fi
	if [ ! -x "$wails3" ]; then
		printf 'error: no se encontró wails3; instálalo o define WAILS3.\n' >&2
		exit 1
	fi
fi

listener_pid() {
	local pid
	pid="$(ss -H -ltnp "sport = :$port" 2>/dev/null |
		sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' | head -n 1)"
	[ -n "$pid" ] && printf '%s\n' "$pid"
	return 0
}

# Comprueba que el proceso pertenece a este repositorio o a uno de sus
# worktrees; así no se toca el Vite de otro proyecto que use el mismo puerto.
same_repo() {
	local cwd common theirs
	cwd="$(readlink -f "/proc/$1/cwd" 2>/dev/null)" || return 1
	common="$(git -C "$root" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" || return 1
	theirs="$(git -C "$cwd" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" || return 1
	[ "$theirs" = "$common" ]
}

parent_of() {
	ps -o ppid= -p "$1" 2>/dev/null | tr -d '[:space:]'
}

process_cmd() {
	tr '\0' ' ' <"/proc/$1/cmdline" 2>/dev/null
}

# Sube por la cadena de padres hasta el proceso que quedó colgado de systemd.
# Un proceso con un padre vivo (terminal, make, wails3) no es un huérfano y no
# debe tocarse automáticamente.
orphan_root() {
	local pid="$1" parent
	while :; do
		parent="$(parent_of "$pid")"
		[ -n "$parent" ] || return 1
		if [ "$parent" -le 1 ] || [ "$(ps -o comm= -p "$parent" 2>/dev/null)" = systemd ]; then
			printf '%s\n' "$pid"
			return 0
		fi
		pid="$parent"
	done
}

is_dev_command() {
	case "$(process_cmd "$1")" in
		*"wails3 task dev:frontend"* | *"wails3 task run"* | *"npm run dev"* | *vite* | *"./dist/better-rpt"*) return 0 ;;
		*) return 1 ;;
	esac
}

kill_tree() {
	local pid="$1" sig="${2:-TERM}" child
	for child in $(ps -o pid= --ppid "$pid" 2>/dev/null); do
		kill_tree "$child" "$sig"
	done
	kill -"$sig" "$pid" 2>/dev/null || true
}

wait_gone() {
	local pid="$1" i
	for i in {1..30}; do
		kill -0 "$pid" 2>/dev/null || return 0
		sleep 0.1
	done
	return 1
}

stop_stale_port() {
	local pid owner
	pid="$(listener_pid)"
	[ -n "$pid" ] || return 0

	if ! same_repo "$pid"; then
		printf 'error: el puerto %s lo ocupa el PID %s, que no pertenece a este repositorio.\n' \
			"$port" "$pid" >&2
		return 1
	fi

	owner="$(orphan_root "$pid")" || owner=""
	if [ "$mode" = --stop ]; then
		[ -n "$owner" ] || owner="$pid"
	elif [ -z "$owner" ] || ! is_dev_command "$owner"; then
		printf 'error: el puerto %s está ocupado por un proceso de este repositorio (PID %s)\n' \
			"$port" "$pid" >&2
		printf '       que parece una sesión activa; ciérrala o usa "make dev-stop" si está colgada.\n' >&2
		return 1
	fi

	printf 'aviso: liberando el puerto %s; se detiene un Vite huérfano de un "make dev" anterior (PID %s).\n' \
		"$port" "$pid"
	kill_tree "$owner" TERM
	if ! wait_gone "$pid"; then
		kill_tree "$owner" KILL
		wait_gone "$pid" || true
	fi
	if [ -n "$(listener_pid)" ]; then
		printf 'error: el puerto %s sigue ocupado tras detener el PID %s.\n' "$port" "$pid" >&2
		return 1
	fi
}

# Retira árboles de desarrollo huérfanos que ya no escuchan en el puerto, como
# `wails3 task run` y la app que quedan tras un SIGKILL. Si no se limpian, la
# app vieja conserva el bloqueo de instancia única y el siguiente `make dev`
# falla antes de arrancar.
stop_orphaned_dev() {
	local pid cmd owner roots=() list=() seen=" "
	while read -r pid cmd; do
		case "$cmd" in
			*"wails3 task dev:frontend"* | *"wails3 task run"* | *"npm run dev"* | *vite* | *"./dist/better-rpt"*) ;;
			*) continue ;;
		esac
		same_repo "$pid" || continue
		owner="$(orphan_root "$pid")" || continue
		is_dev_command "$owner" || continue
		list+=("$owner")
	done < <(ps -eo pid=,args=)

	for owner in "${list[@]}"; do
		case "$seen" in *" $owner "*) continue ;; esac
		seen+="$owner "
		roots+=("$owner")
	done
	[ "${#roots[@]}" -gt 0 ] || return 0

	printf 'aviso: liberando restos de un "make dev" anterior (PID %s).\n' "${roots[*]}"
	for owner in "${roots[@]}"; do
		kill_tree "$owner" TERM
	done
	for owner in "${roots[@]}"; do
		if ! wait_gone "$owner"; then
			kill_tree "$owner" KILL
			wait_gone "$owner" || true
		fi
	done
}

stop_orphaned_dev
stop_stale_port || exit 1
if [ "$mode" = --stop ]; then
	exit 0
fi

dev_pid=""

# Señala a wails3 usando su grupo solo si de verdad es el líder; así nunca se
# alcanza por error al grupo del terminal.
terminate_dev() {
	local sig="$1" pgid
	pgid="$(ps -o pgid= -p "$dev_pid" 2>/dev/null | tr -d '[:space:]')"
	if [ "$pgid" = "$dev_pid" ]; then
		kill -"$sig" -- "-$dev_pid" 2>/dev/null && return 0
	fi
	kill -"$sig" "$dev_pid" 2>/dev/null || true
}

cleanup() {
	local status=$?
	trap - EXIT HUP INT TERM
	if [ -n "$dev_pid" ]; then
		terminate_dev TERM
		sleep 1
		terminate_dev KILL
		wait "$dev_pid" 2>/dev/null || true
		dev_pid=""
	fi
	stop_orphaned_dev >/dev/null 2>&1 || true
	stop_stale_port >/dev/null 2>&1 || true
	exit "$status"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Iniciando Wails dev en el puerto %s...\n' "$port"
# Wails queda en su propio grupo: el SIGHUP del terminal no lo mata, y así el
# envoltorio puede pedirle un cierre ordenado con TERM. Wails, a su vez, mata
# sus procesos de fondo (Vite incluido) mediante refresh.
set -m
"$wails3" dev -config "$root/build/config.yml" -port "$port" &
dev_pid=$!
set +m
wait "$dev_pid"
status=$?
dev_pid=""
exit "$status"
