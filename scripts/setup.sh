#!/usr/bin/env bash
# Instala los paquetes del sistema que necesita la compilación de Better RPT.
# No gestiona Go, Node/npm ni el CLI de Wails; esos pasos están en el README.
set -euo pipefail

# Cada fila: etiqueta|comprobación|apt|dnf|pacman|zypper.
# Las comprobaciones se hacen por capacidad (binario o módulo pkg-config), no
# por nombre de paquete, para no reinstalar paquetes ya cubiertos.
prereqs=(
	"compilador C (gcc)|cmd:gcc|build-essential|gcc|gcc|gcc"
	"make|cmd:make|build-essential|make|make|make"
	"pkg-config|cmd:pkg-config|pkg-config|pkgconf-pkg-config|pkgconf|pkg-config"
	"GTK3 (desarrollo)|pc:gtk+-3.0|libgtk-3-dev|gtk3-devel|gtk3|gtk3-devel"
	"WebKit2GTK 4.1 (desarrollo)|pc:webkit2gtk-4.1|libwebkit2gtk-4.1-dev|webkit2gtk4.1-devel|webkit2gtk-4.1|webkit2gtk3-devel"
	"OpenGL/Mesa (desarrollo)|pc:gl|libgl1-mesa-dev|mesa-libGL-devel|mesa|Mesa-libGL-devel"
	"Wayland (desarrollo)|pc:wayland-client|libwayland-dev|wayland-devel|wayland|wayland-devel"
	"xkbcommon (desarrollo)|pc:xkbcommon|libxkbcommon-dev|libxkbcommon-devel|libxkbcommon|libxkbcommon-devel"
	"X11 (desarrollo)|pc:x11|libx11-dev|libX11-devel|libx11|libX11-devel"
	"Xext (desarrollo)|pc:xext|libxext-dev|libXext-devel|libxext|libXext-devel"
	"Xrender (desarrollo)|pc:xrender|libxrender-dev|libXrender-devel|libxrender|libXrender-devel"
	"Xfixes (desarrollo)|pc:xfixes|libxfixes-dev|libXfixes-devel|libxfixes|libXfixes-devel"
	"Xcursor (desarrollo)|pc:xcursor|libxcursor-dev|libXcursor-devel|libxcursor|libXcursor-devel"
	"Xi (desarrollo)|pc:xi|libxi-dev|libXi-devel|libxi|libXi-devel"
	"Xinerama (desarrollo)|pc:xinerama|libxinerama-dev|libXinerama-devel|libxinerama|libXinerama-devel"
	"Xrandr (desarrollo)|pc:xrandr|libxrandr-dev|libXrandr-devel|libxrandr|libXrandr-devel"
	"Xxf86vm (desarrollo)|pc:xxf86vm|libxxf86vm-dev|libXxf86vm-devel|libxxf86vm|libXxf86vm-devel"
)

detect_pm() {
	if command -v apt-get >/dev/null 2>&1; then
		printf 'apt\n'
	elif command -v dnf >/dev/null 2>&1; then
		printf 'dnf\n'
	elif command -v pacman >/dev/null 2>&1; then
		printf 'pacman\n'
	elif command -v zypper >/dev/null 2>&1; then
		printf 'zypper\n'
	else
		return 1
	fi
}

check_prereq() {
	local check="$1"
	case "$check" in
		cmd:*)
			command -v "${check#cmd:}" >/dev/null 2>&1
			;;
		pc:*)
			pkg-config --exists "${check#pc:}" >/dev/null 2>&1
			;;
		*)
			return 2
			;;
	esac
}

package_for() {
	local row="$1" manager="$2"
	local label check apt_pkg dnf_pkg pacman_pkg zypper_pkg
	IFS='|' read -r label check apt_pkg dnf_pkg pacman_pkg zypper_pkg <<<"$row"
	case "$manager" in
		apt) printf '%s' "$apt_pkg" ;;
		dnf) printf '%s' "$dnf_pkg" ;;
		pacman) printf '%s' "$pacman_pkg" ;;
		zypper) printf '%s' "$zypper_pkg" ;;
	esac
}

install_packages() {
	case "$pm" in
		apt)
			DEBIAN_FRONTEND=noninteractive apt-get update ||
				printf 'aviso: "apt-get update" falló; se intentará instalar con los índices disponibles.\n' >&2
			DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
			;;
		dnf)
			dnf install -y "$@"
			;;
		pacman)
			pacman -S --needed --noconfirm "$@"
			;;
		zypper)
			zypper --non-interactive install "$@"
			;;
	esac
}

print_summary() {
	printf '\nResumen:\n'
	printf '  Detectados:   %d\n' "$1"
	printf '  Ya presentes: %d\n' "$2"
	printf '  Instalados:   %d\n' "$3"
	printf '  Fallidos:     %d\n' "$4"
}

print_check_summary() {
	printf '\nResumen:\n'
	printf '  Detectados:   %d\n' "$1"
	printf '  Ya presentes: %d\n' "$2"
	printf '  Faltan:       %d\n' "$3"
}

print_unsupported() {
	printf 'error: no se detectó un gestor de paquetes soportado (apt, dnf, pacman o zypper).\n' >&2
	printf '\nPaquetes requeridos por gestor (apt | dnf | pacman | zypper):\n' >&2
	local row label check apt_pkg dnf_pkg pacman_pkg zypper_pkg
	for row in "${prereqs[@]}"; do
		IFS='|' read -r label check apt_pkg dnf_pkg pacman_pkg zypper_pkg <<<"$row"
		printf '  %s: %s | %s | %s | %s\n' \
			"$label" "$apt_pkg" "$dnf_pkg" "$pacman_pkg" "$zypper_pkg" >&2
	done
	printf '\nInstala esos paquetes con el gestor de tu distribución.\n' >&2
}

check_only=0
if [ "$#" -gt 1 ]; then
	printf 'uso: %s [--check]\n' "$0" >&2
	exit 2
fi
case "${1:-}" in
	'') ;;
	--check) check_only=1 ;;
	*)
		printf 'uso: %s [--check]\n' "$0" >&2
		exit 2
		;;
esac

if ! pm="$(detect_pm)"; then
	print_unsupported
	exit 1
fi

printf 'Gestor de paquetes: %s\n' "$pm"
printf 'Comprobando prerrequisitos del sistema...\n'

detected=${#prereqs[@]}
present_before=0
declare -a missing_indices=()
declare -a was_present=()

for i in "${!prereqs[@]}"; do
	IFS='|' read -r label check apt_pkg dnf_pkg pacman_pkg zypper_pkg <<<"${prereqs[$i]}"
	if check_prereq "$check"; then
		was_present[$i]=1
		present_before=$((present_before + 1))
		printf '  [ok]        %s\n' "$label"
	else
		was_present[$i]=0
		missing_indices+=("$i")
		package="$(package_for "${prereqs[$i]}" "$pm")"
		printf '  [falta]     %s -> %s\n' "$label" "$package"
	fi
done

missing=${#missing_indices[@]}

if [ "$check_only" -eq 1 ]; then
	if [ "$missing" -eq 0 ]; then
		printf '\nTodos los prerrequisitos están presentes.\n'
	else
		printf '\nEjecuta "sudo make setup" para instalar los paquetes que faltan.\n'
	fi
	print_check_summary "$detected" "$present_before" "$missing"
	exit 0
fi

if [ "$missing" -eq 0 ]; then
	printf '\nTodos los prerrequisitos están presentes; no hay nada que instalar.\n'
	print_summary "$detected" "$present_before" 0 0
	exit 0
fi

if [ "$(id -u)" -ne 0 ]; then
	print_summary "$detected" "$present_before" 0 "$missing"
	printf '\nerror: faltan %d prerrequisito(s) y se requieren privilegios para instalarlos.\n' "$missing" >&2
	printf 'Ejecuta "sudo make setup".\n' >&2
	exit 1
fi

declare -a install_pkgs=()
declare -A seen_pkgs=()
for i in "${missing_indices[@]}"; do
	package="$(package_for "${prereqs[$i]}" "$pm")"
	if [ -n "$package" ] && [ -z "${seen_pkgs[$package]:-}" ]; then
		seen_pkgs["$package"]=1
		install_pkgs+=("$package")
	fi
done

printf '\nInstalando %d paquete(s): %s\n' "${#install_pkgs[@]}" "${install_pkgs[*]}"
install_rc=0
if ! install_packages "${install_pkgs[@]}"; then
	install_rc=1
fi

already=0
installed=0
failed=0
for i in "${!prereqs[@]}"; do
	IFS='|' read -r label check apt_pkg dnf_pkg pacman_pkg zypper_pkg <<<"${prereqs[$i]}"
	if check_prereq "$check"; then
		if [ "${was_present[$i]}" = 1 ]; then
			already=$((already + 1))
		else
			installed=$((installed + 1))
			printf '  [instalado] %s\n' "$label"
		fi
	else
		failed=$((failed + 1))
		package="$(package_for "${prereqs[$i]}" "$pm")"
		printf '  [fallido]   %s -> %s\n' "$label" "$package"
	fi
done

if [ "$install_rc" -ne 0 ] && [ "$failed" -eq 0 ]; then
	printf '\naviso: el gestor de paquetes devolvió un error, pero todos los prerrequisitos están presentes.\n' >&2
fi

print_summary "$detected" "$already" "$installed" "$failed"
if [ "$failed" -gt 0 ]; then
	printf '\nerror: %d prerrequisito(s) no se pudieron instalar.\n' "$failed" >&2
	exit 1
fi
