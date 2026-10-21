#!/usr/bin/env bash
# Publica una release de Better RPT: deduce la versión, mueve el changelog,
# actualiza referencias, compila y verifica los artefactos, y crea el commit,
# el tag, la rama y la release de GitHub.
#
# Solo pide confirmación dos veces: antes de escribir/compilar y antes de
# publicar. Usa --yes para un flujo desatendido y --dry-run para ver el plan.
# Con --repack recompila y re-sube los artefactos de una versión ya publicada.
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

# Un push no debe quedarse colgado esperando una contraseña o una conexión
# muerta. Respeta GIT_SSH_COMMAND si el entorno ya define uno.
export GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o BatchMode=yes -o ConnectTimeout=15 -o ServerAliveInterval=15 -o ServerAliveCountMax=3}"

SECONDS=0
phase_start=0
timing() {
	info "  $1: $((SECONDS - phase_start))s"
	phase_start=$SECONDS
}

version=""
dry_run=0
repack=0
force=0
assume_yes=0
no_push=0
skip_build=0
verify=1
verify_install=0

usage() {
	cat <<'EOF'
Uso: scripts/release.sh [opciones]

  --version X.Y.Z   Fija la versión; por defecto se deduce de los commits.
  --repack          Recompila y re-sube los artefactos de una versión publicada.
  --force           Con --repack, reemplaza assets aunque los SHA-256 difieran.
  --dry-run         Muestra el plan sin escribir, compilar ni publicar.
  --yes             No pide confirmación (uso desatendido).
  --no-push         Crea commit, tag y rama, pero no publica.
  --skip-build      Reutiliza los artefactos de dist/ (reanudar).
  --no-verify       No repite la compilación para comprobar reproducibilidad.
  --verify-install  Instala el .deb en un contenedor Ubuntu 22.04 limpio.
  -h, --help        Muestra esta ayuda.
EOF
}

die() {
	echo "error: $*" >&2
	exit 1
}

info() {
	printf '%s\n' "$*"
}

confirm() {
	[ "$assume_yes" = 1 ] && return 0
	local answer
	read -r -p "$1 [y/N] " answer
	case "$answer" in
	y | Y | yes | YES) return 0 ;;
	*) return 1 ;;
	esac
}

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		version="${2:-}"
		[ -n "$version" ] || die "--version necesita un valor"
		shift 2
		;;
	--repack)
		repack=1
		shift
		;;
	--force)
		force=1
		shift
		;;
	--dry-run)
		dry_run=1
		shift
		;;
	--yes)
		assume_yes=1
		shift
		;;
	--no-push)
		no_push=1
		shift
		;;
	--skip-build)
		skip_build=1
		shift
		;;
	--no-verify)
		verify=0
		shift
		;;
	--verify-install)
		verify_install=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage
		die "opción desconocida: $1"
		;;
	esac
done

[ -f CHANGELOG.md ] || die "ejecuta el script desde el repositorio"
[ -f Makefile ] || die "falta Makefile"
if [ "$force" = 1 ] && [ "$repack" = 0 ]; then
	die "--force solo se usa con --repack"
fi

# ---------------------------------------------------------------------------
# Estado del repositorio y de la release
# ---------------------------------------------------------------------------

branch="$(git rev-parse --abbrev-ref HEAD)"
[ "$branch" = "main" ] || die "la release se prepara desde main, no desde $branch"
[ -z "$(git status --porcelain)" ] || die "el árbol de trabajo no está limpio"

prev_tag="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"
[ -n "$prev_tag" ] || die "no hay ningún tag vX.Y.Z anterior"
prev_version="${prev_tag#v}"

remote_url="$(git remote get-url origin)"
repo_slug="$(python3 - "$remote_url" <<'PY'
import re, sys
match = re.search(r"github[^:]*[:/]([^/]+/[^/]+?)(?:\.git)?$", sys.argv[1])
if not match:
    sys.exit("no se pudo deducir owner/repo de " + sys.argv[1])
print(match.group(1))
PY
)" || die "no se pudo deducir el repositorio de origin"
repo_url="https://github.com/$repo_slug"
owner="${repo_slug%%/*}"

# Un repack reutiliza la última versión publicada salvo que se indique otra.
if [ "$repack" = 1 ] && [ -z "$version" ]; then
	version="$prev_version"
fi

if [ -z "$version" ]; then
	version="$(git log --format='%s%n%b%n---RELEASE-COMMIT---' "$prev_tag..HEAD" | python3 -c '
import re, sys
prev = sys.argv[1]
major, minor, patch = map(int, prev.split("."))
blocks = [b.strip() for b in sys.stdin.read().split("---RELEASE-COMMIT---") if b.strip()]
breaking = feat = user = False
for block in blocks:
    subject = block.splitlines()[0].strip()
    if re.match(r"^[a-zA-Z]+(\([^)]*\))?!:", subject) or "BREAKING CHANGE" in block:
        breaking = True
    if re.match(r"^feat(\(|:)", subject):
        feat = True
    if re.match(r"^(feat|fix|perf|revert)(\(|:)", subject) or breaking:
        user = True
if not user:
    sys.exit("no hay commits feat/fix/perf/revert desde " + prev)
if breaking:
    if major > 0:
        major, minor, patch = major + 1, 0, 0
    else:
        minor, patch = minor + 1, 0
elif feat and major > 0:
    minor, patch = minor + 1, 0
else:
    patch += 1
print(f"{major}.{minor}.{patch}")
' "$prev_version")" || die "no se pudo deducir la versión"
fi

[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "versión inválida: $version"
if [ "$version" != "$prev_version" ] &&
	[ "$(printf '%s\n%s\n' "$prev_version" "$version" | sort -V | head -n 1)" = "$version" ]; then
	die "la versión $version debe ser mayor que $prev_version"
fi

tag="v$version"
release_branch="release/$tag"
changelog_has_section=0
grep -q "^## \[$version\]" CHANGELOG.md && changelog_has_section=1

tag_exists=0
git rev-parse -q --verify "refs/tags/$tag" >/dev/null && tag_exists=1
if [ "$version" = "$prev_version" ] && [ "$tag_exists" = 0 ]; then
	die "la versión $version ya es la última; indica una mayor"
fi
if [ "$repack" = 1 ] && [ "$tag_exists" = 0 ]; then
	die "--repack necesita una versión publicada; no existe el tag $tag"
fi
branch_exists=0
git show-ref --verify --quiet "refs/heads/$release_branch" && branch_exists=1
release_commit="$(git log --format='%H' --grep="^chore(release): $tag$" -1 || true)"

# Un tag o una rama ya existentes solo valen si apuntan al commit de release.
if [ "$tag_exists" = 1 ] && [ -n "$release_commit" ] &&
	[ "$(git rev-parse "refs/tags/$tag^{commit}")" != "$release_commit" ]; then
	die "el tag $tag no apunta al commit de release $release_commit"
fi
if [ "$branch_exists" = 1 ] && [ -n "$release_commit" ] &&
	[ "$(git rev-parse "refs/heads/$release_branch")" != "$release_commit" ]; then
	die "la rama $release_branch no apunta al commit de release $release_commit"
fi

gh_ready() {
	command -v gh >/dev/null 2>&1 || die "gh no está instalado"
	local active token
	active="$(gh api user --jq .login 2>/dev/null || true)"
	if [ "$active" = "$owner" ]; then
		return 0
	fi
	token="$(gh auth token --user "$owner" 2>/dev/null || true)"
	[ -n "$token" ] || die "gh no tiene sesión para $owner; ejecuta 'gh auth login'"
	export GH_TOKEN="$token"
}

if [ "$no_push" = 0 ]; then
	gh_ready
	if [ "$repack" = 1 ]; then
		gh release view "$tag" >/dev/null 2>&1 ||
			die "la release $tag no existe; no hay artefactos que re-subir"
	elif gh release view "$tag" >/dev/null 2>&1; then
		info "La release $tag ya está publicada: $repo_url/releases/tag/$tag"
		exit 0
	fi
fi

# ---------------------------------------------------------------------------
# Cambios locales
# ---------------------------------------------------------------------------

new_section() {
	python3 - "$version" "$prev_version" "$(date +%F)" "$(git rev-parse --short HEAD)" <<'PY'
import re, sys

version, prev, date, head = sys.argv[1:5]
text = open("CHANGELOG.md", encoding="utf-8").read()
match = re.search(r"(?ms)^## \[Unreleased\]\n(.*?)(?=^## \[)", text)
if not match:
    sys.exit("falta la sección [Unreleased] en CHANGELOG.md")
body = match.group(1).strip()
if not body:
    sys.exit("la sección [Unreleased] está vacía")

print(f"## [{version}] - {date} ({head})")
print()
print(body)
print()
print(f"**Full changelog:** [v{prev}...v{version}][{version}]")
PY
}

update_changelog() {
	python3 - "$version" "$prev_version" "$repo_url" "$(date +%F)" "$(git rev-parse --short HEAD)" <<'PY'
import re, sys

version, prev, repo_url, date, head = sys.argv[1:6]
path = "CHANGELOG.md"
text = open(path, encoding="utf-8").read()
if re.search(rf"(?m)^## \[{re.escape(version)}\]", text):
    sys.exit(0)

match = re.search(r"(?ms)^## \[Unreleased\]\n(.*?)(?=^## \[)", text)
if not match:
    sys.exit("falta la sección [Unreleased] en CHANGELOG.md")
body = match.group(1).strip()
if not body:
    sys.exit("la sección [Unreleased] está vacía")

section = (
    f"## [Unreleased]\n\n"
    f"## [{version}] - {date} ({head})\n\n"
    f"{body}\n\n"
    f"**Full changelog:** [v{prev}...v{version}][{version}]\n\n"
)
text = text[: match.start()] + section + text[match.end() :]

unreleased = f"[Unreleased]: {repo_url}/compare/v{version}...HEAD"
text, done = re.subn(r"(?m)^\[Unreleased\]: .*$", unreleased, text, count=1)
if done != 1:
    sys.exit("falta el enlace [Unreleased] al final del changelog")

version_link = f"[{version}]: {repo_url}/compare/v{prev}...v{version}"
if f"[{version}]:" not in text:
    text = re.sub(
        r"(?m)^(\[Unreleased\]: .*)$", r"\1\n" + version_link, text, count=1
    )

open(path, "w", encoding="utf-8").write(text)
PY
}

bump_version_refs() {
	python3 - "$prev_version" "$version" <<'PY'
import pathlib, re, sys

prev, new = sys.argv[1:3]
pattern = re.compile(r"(?<![\d.])" + re.escape(prev) + r"(?![\d.])")
files = [
    "Makefile",
    "scripts/build-deb.sh",
    "build/config.yml",
    "README.md",
    "docs/appimage.md",
    "docs/build.md",
    "docs/distribution.md",
]
changed = []
for name in files:
    path = pathlib.Path(name)
    old = path.read_text(encoding="utf-8")
    updated = pattern.sub(lambda _: new, old)
    if updated != old:
        path.write_text(updated, encoding="utf-8")
        changed.append(name)

path = pathlib.Path(".release/conventions.md")
old = path.read_text(encoding="utf-8")
updated = re.sub(
    r"(VERSION \?= )" + re.escape(prev) + r"(?![\d.])", r"\g<1>" + new, old
)
if updated != old:
    path.write_text(updated, encoding="utf-8")
    changed.append(str(path))

if not changed:
    sys.exit(f"no se encontró {prev} en los archivos de versión")
print("Actualizados: " + ", ".join(changed))
PY
}

update_release_state() {
	python3 - "$version" "$tag" "$release_branch" "$(date +%F)" <<'PY'
import json, pathlib, sys

version, tag, branch, date = sys.argv[1:5]
path = pathlib.Path(".release/config.json")
data = json.loads(path.read_text(encoding="utf-8"))
data["lastRelease"] = {
    "version": version,
    "tag": tag,
    "branch": branch,
    "date": date,
}
path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY
}

run_checks() {
	info "Ejecutando tests y vet..."
	make test
	make vet
	if command -v govulncheck >/dev/null 2>&1; then
		info "Ejecutando govulncheck..."
		govulncheck ./...
	else
		info "aviso: govulncheck no está instalado; se omite"
	fi
}

# ---------------------------------------------------------------------------
# Artefactos
# ---------------------------------------------------------------------------

appimagetool=""
appimage_runtime=""

prepare_appimage_tools() {
	local cache="${XDG_CACHE_HOME:-$HOME/.cache}/better-rpt/release-tools"
	local fallback="/tmp/opencode/appimage-poc"
	mkdir -p "$cache"

	if [ ! -x "$cache/appimagetool" ] && [ -x "$fallback/appimagetool" ]; then
		cp -n "$fallback/appimagetool" "$cache/appimagetool"
	fi
	if [ ! -f "$cache/runtime-x86_64" ] && [ -f "$fallback/runtime-x86_64" ]; then
		cp -n "$fallback/runtime-x86_64" "$cache/runtime-x86_64"
	fi

	appimagetool="${APPIMAGETOOL:-$cache/appimagetool}"
	appimage_runtime="${APPIMAGE_RUNTIME:-$cache/runtime-x86_64}"
	[ -x "$appimagetool" ] || die "falta appimagetool; define APPIMAGETOOL o colócalo en $cache"
	[ -f "$appimage_runtime" ] || die "falta runtime-x86_64; define APPIMAGE_RUNTIME o colócalo en $cache"

	# Hashes fijados en docs/appimage.md; actualiza ambos si cambias las herramientas.
	[ "$(sha256sum "$appimagetool" | cut -d' ' -f1)" = "95cbe7cce9717fce90c484e34052ee7c7f1d7635b33c12525b4776826a7d29b6" ] ||
		die "hash inesperado de appimagetool"
	[ "$(sha256sum "$appimage_runtime" | cut -d' ' -f1)" = "617aa77314d57c8b8e4fae9c3f0b35af7b30d5529cb69e98f64f92510ed98a46" ] ||
		die "hash inesperado de runtime-x86_64"
}

build_appimage() {
	local output="$1"
	local appdir="dist/BetterRPT.AppDir"
	if [ -d "$appdir" ]; then
		rm -r "$appdir"
	fi
	mkdir -p "$appdir/usr/bin" "$appdir/usr/lib/better-rpt"
	cp dist/better-rpt "$appdir/usr/bin/better-rpt"
	cp dist/better-rpt-helper "$appdir/usr/lib/better-rpt/better-rpt-helper"
	cp packaging/AppRun "$appdir/AppRun"
	cp packaging/better-rpt.desktop "$appdir/better-rpt.desktop"
	cp packaging/better-rpt.png "$appdir/better-rpt.png"
	chmod 0755 "$appdir/AppRun" "$appdir/usr/bin/better-rpt" "$appdir/usr/lib/better-rpt/better-rpt-helper"
	ARCH=x86_64 SOURCE_DATE_EPOCH=1700000000 APPIMAGE_EXTRACT_AND_RUN=1 \
		"$appimagetool" --no-appstream --runtime-file "$appimage_runtime" \
		"$appdir" "$output"
}

build_artifacts() {
	info "Compilando el .deb en Ubuntu 22.04..."
	make deb VERSION="$version"

	prepare_appimage_tools
	info "Construyendo el AppImage..."
	build_appimage "dist/BetterRPT-${version}-x86_64.AppImage"

	if [ "$verify" = 1 ]; then
		info "Verificando reproducibilidad (segunda compilación)..."
		local deb_first app_first
		deb_first="$(sha256sum "dist/better-rpt_${version}-1_amd64.deb" | cut -d' ' -f1)"
		app_first="$(sha256sum "dist/BetterRPT-${version}-x86_64.AppImage" | cut -d' ' -f1)"
		# El frontend ya está generado y no cambia entre pasadas. Recompilar solo
		# el .deb evita repetir bindings y Vite en la comprobación de reproducibilidad.
		VERSION="$version" ./scripts/build-deb.sh >/dev/null
		build_appimage "/tmp/better-rpt-${version}-verify.AppImage" >/dev/null
		[ "$deb_first" = "$(sha256sum "dist/better-rpt_${version}-1_amd64.deb" | cut -d' ' -f1)" ] ||
			die "el .deb no es reproducible"
		[ "$app_first" = "$(sha256sum "/tmp/better-rpt-${version}-verify.AppImage" | cut -d' ' -f1)" ] ||
			die "el AppImage no es reproducible"
		rm -f "/tmp/better-rpt-${version}-verify.AppImage"
	fi

	(
		cd dist
		sha256sum "better-rpt_${version}-1_amd64.deb" "BetterRPT-${version}-x86_64.AppImage" >SHA256SUMS
	)
	verify_artifacts

	if [ "$verify_install" = 1 ]; then
		info "Instalando el .deb en un contenedor Ubuntu 22.04 limpio..."
		docker run --rm -v "$PWD/dist:/dist:ro" ubuntu:22.04 bash -c "
			set -e
			apt-get update -qq
			DEBIAN_FRONTEND=noninteractive apt-get install -y -qq /dist/better-rpt_${version}-1_amd64.deb >/dev/null
			better-rpt --version
		" | grep -qx "better-rpt $version" || die "la instalación limpia falló"
	fi
}

verify_artifacts() {
	local deb="dist/better-rpt_${version}-1_amd64.deb"
	local app="dist/BetterRPT-${version}-x86_64.AppImage"
	[ -f "$deb" ] || die "falta $deb; ejecuta sin --skip-build para generarlo"
	[ -f "$app" ] || die "falta $app; ejecuta sin --skip-build para generarlo"
	[ -f dist/SHA256SUMS ] || die "falta dist/SHA256SUMS; ejecuta sin --skip-build para generarlo"
	[ "$(dpkg-deb -f "$deb" Version)" = "$version-1" ] || die "versión incorrecta en el .deb"
	[ "$(APPIMAGE_EXTRACT_AND_RUN=1 "$app" --version)" = "better-rpt $version" ] ||
		die "versión incorrecta en el AppImage"
	(
		cd dist
		sha256sum -c SHA256SUMS >/dev/null
	) || die "SHA256SUMS no coincide con los artefactos"
}

backup_repack_assets() {
	local dir="dist/repack-backup-$(date +%Y%m%d-%H%M%S)"
	local name copied=0
	mkdir -p "$dir"
	for name in "better-rpt_${version}-1_amd64.deb" "BetterRPT-${version}-x86_64.AppImage" "SHA256SUMS"; do
		if [ -f "dist/$name" ]; then
			cp "dist/$name" "$dir/"
			copied=1
		fi
	done
	if [ "$copied" = 1 ]; then
		info "Copia local de los artefactos actuales: $dir"
	else
		rmdir "$dir"
	fi
}

verify_repack_assets() {
	python3 - "$repo_slug" "$tag" <<'PY'
import hashlib, json, pathlib, subprocess, sys

repo, tag = sys.argv[1:3]
try:
    data = json.loads(
        subprocess.check_output(
            ["gh", "api", f"repos/{repo}/releases/tags/{tag}"],
            stderr=subprocess.DEVNULL,
        )
    )
except subprocess.CalledProcessError:
    sys.exit(f"no se pudo leer la release {tag}")

published = {
    asset["name"]: (asset.get("digest") or "").removeprefix("sha256:")
    for asset in data.get("assets", [])
}
sums_path = pathlib.Path("dist/SHA256SUMS")
local = {}
for line in sums_path.read_text(encoding="utf-8").splitlines():
    line = line.strip()
    if not line:
        continue
    digest, name = line.split(maxsplit=1)
    local[name.strip().lstrip("*")] = digest
local["SHA256SUMS"] = hashlib.sha256(sums_path.read_bytes()).hexdigest()

different = {
    name: (published.get(name), digest)
    for name, digest in local.items()
    if published.get(name) != digest
}
if different:
    for name, (old, new) in different.items():
        print(f"  {name}: publicado {old or 'sin digest'} / nuevo {new}", file=sys.stderr)
    sys.exit("los artefactos nuevos no coinciden con los publicados; no se reemplaza la release")
print(f"Los {len(local)} artefactos coinciden con los publicados en {tag}.")
PY
}

update_artifact_docs() {
	local deb_size app_size deb_sha app_sha installed
	deb_size="$(stat -c %s "dist/better-rpt_${version}-1_amd64.deb")"
	app_size="$(stat -c %s "dist/BetterRPT-${version}-x86_64.AppImage")"
	deb_sha="$(sha256sum "dist/better-rpt_${version}-1_amd64.deb" | cut -d' ' -f1)"
	app_sha="$(sha256sum "dist/BetterRPT-${version}-x86_64.AppImage" | cut -d' ' -f1)"
	installed="$(dpkg-deb -f "dist/better-rpt_${version}-1_amd64.deb" Installed-Size)"

	python3 - "$deb_size" "$app_size" "$deb_sha" "$app_sha" "$installed" <<'PY'
import pathlib, re, sys

deb_size, app_size = int(sys.argv[1]), int(sys.argv[2])
deb_sha, app_sha = sys.argv[3], sys.argv[4]
installed = int(sys.argv[5])
thousands = lambda n: f"{n:,}".replace(",", ".")
megabytes = lambda n: f"{n / 1048576:.1f}".replace(".", ",")

path = pathlib.Path("docs/appimage.md")
text = path.read_text(encoding="utf-8")
text, count = re.subn(
    r"(produjo `)[0-9a-f]{64}(` \()[0-9,]+( MB\))",
    rf"\g<1>{app_sha}\g<2>{megabytes(app_size)}\g<3>",
    text,
    count=1,
)
if count != 1:
    sys.exit("no se encontró la línea del hash del AppImage en docs/appimage.md")
path.write_text(text, encoding="utf-8")

path = pathlib.Path("docs/build.md")
text = path.read_text(encoding="utf-8")
text, count = re.subn(
    r"[0-9.]+ KB; el `\.deb` final mide [0-9,]+ MB \([0-9.]+ bytes\)\.",
    f"{thousands(installed)} KB; el `.deb` final mide {megabytes(deb_size)} MB ({thousands(deb_size)} bytes).",
    text,
    count=1,
)
if count != 1:
    sys.exit("no se encontró la línea de tamaño del .deb en docs/build.md")
path.write_text(text, encoding="utf-8")
PY
}

# ---------------------------------------------------------------------------
# Git y GitHub
# ---------------------------------------------------------------------------

create_tag() {
	local message
	message="$(mktemp)"
	# El cuerpo se genera aparte para no duplicar el encabezado con el subject.
	python3 - "$version" "$repo_url" "$prev_tag" <<'PY' >"$message"
import re, sys

version, repo_url, prev_tag = sys.argv[1:4]
text = open("CHANGELOG.md", encoding="utf-8").read()
match = re.search(
    rf"(?ms)^## \[{re.escape(version)}\].*?\n(.*?)(?=^## \[)", text
)
if not match:
    sys.exit("no se encontró la sección de " + version + " en el changelog")
body = match.group(1).strip()
bullet = re.search(r"^- (.+)$", body, re.M)
if not bullet:
    sys.exit("la sección de " + version + " no tiene entradas")
subject = bullet.group(1).strip().rstrip(".")
if len(subject) > 72:
    subject = subject[:69].rsplit(" ", 1)[0] + "..."
header = re.search(rf"(?m)^## \[{re.escape(version)}\].*?$", text).group(0)
print(subject)
print()
print(header)
print()
print(body)
print()
print(f"[{version}]: {repo_url}/compare/{prev_tag}...v{version}")
PY
	git tag -a "$tag" -F "$message" --cleanup=verbatim
	rm -f "$message"
}

push_refs() {
	local attempt delay
	for attempt in 1 2 3; do
		if git push --atomic origin main "$tag" "$release_branch"; then
			return 0
		fi
		[ "$attempt" -lt 3 ] || break
		delay=$((attempt * 2))
		info "push falló (intento $attempt/3); reintentando en ${delay}s..."
		sleep "$delay"
	done
	return 1
}

publish() {
	info "Publicando $tag..."
	push_refs || die "no se pudo publicar; el commit, tag y rama quedan locales"
	timing "push"
	gh release create "$tag" --title "$tag" --notes-from-tag --verify-tag \
		"dist/better-rpt_${version}-1_amd64.deb" \
		"dist/BetterRPT-${version}-x86_64.AppImage" \
		"dist/SHA256SUMS"
	timing "subida de artefactos"
	info "Publicada: $repo_url/releases/tag/$tag"
}

# ---------------------------------------------------------------------------
# Plan y ejecución
# ---------------------------------------------------------------------------

if [ "$changelog_has_section" = 1 ] && [ -z "$release_commit" ]; then
	die "ya existe la sección $version en el changelog pero no el commit de release; revisa el árbol"
fi

info "Release:      $tag (anterior $prev_tag)"
info "Repositorio:  $repo_slug"
if [ "$repack" = 1 ]; then
	info "Modo:         reempaquetar y re-subir artefactos"
else
	if [ -n "$release_commit" ]; then
		info "Estado:       reanudar; commit de release $release_commit"
	else
		info "Commits desde $prev_tag:"
		git log --format='  %h %s' "$prev_tag..HEAD"
		info ""
		info "Nueva sección del changelog:"
		info ""
		new_section
	fi
fi
info ""
if [ "$dry_run" = 1 ]; then
	info "Plan (sin cambios):"
	if [ "$repack" = 1 ]; then
		info "  1. recompilar .deb y AppImage de $version"
		info "  2. verificar reproducibilidad y generar dist/SHA256SUMS"
		info "  3. comprobar que los SHA-256 coinciden con la release publicada"
		info "  4. gh release upload $tag --clobber <artefactos>"
	else
		info "  1. mover [Unreleased] a [$version] y actualizar enlaces"
		info "  2. ejecutar tests, vet y govulncheck (si está instalado)"
		info "  3. reemplazar $prev_version por $version en Makefile, script y docs"
		info "  4. compilar .deb y AppImage, generar dist/SHA256SUMS"
		info "  5. actualizar hash/tamaños y .release/config.json"
		info "  6. commit chore(release): $tag, tag anotado y rama $release_branch"
		info "  7. push de main, $tag y $release_branch; gh release create con artefactos"
	fi
	exit 0
fi

phase_start=$SECONDS
if [ "$repack" = 1 ]; then
	confirm "¿Reempaquetar y re-subir $tag?" || die "cancelado"
	timing "preparación y aprobación"
	if [ "$skip_build" = 1 ]; then
		verify_artifacts
	else
		backup_repack_assets
		build_artifacts
	fi
	timing "artefactos"
	if [ "$no_push" = 1 ]; then
		info "Subida omitida (--no-push)."
		info "Tiempo total: ${SECONDS}s"
		exit 0
	fi
	if [ "$force" = 1 ]; then
		info "Aviso: --force reemplaza los assets aunque los SHA-256 difieran."
	else
		verify_repack_assets
	fi
	gh release upload "$tag" --clobber \
		"dist/better-rpt_${version}-1_amd64.deb" \
		"dist/BetterRPT-${version}-x86_64.AppImage" \
		"dist/SHA256SUMS"
	timing "subida de artefactos"
	verify_repack_assets
	info "Re-subido: $repo_url/releases/tag/$tag"
	info "Tiempo total: ${SECONDS}s"
	exit 0
fi

if [ -z "$release_commit" ]; then
	confirm "¿Crear la release $tag?" || die "cancelado"
	timing "preparación y aprobación"
	run_checks
	timing "tests y vet"
	update_changelog
	bump_version_refs
	update_release_state
	timing "changelog y versión"

	if [ "$skip_build" = 1 ]; then
		verify_artifacts
	else
		build_artifacts
	fi
	update_artifact_docs
	timing "artefactos y docs"

	git add -- Makefile scripts/build-deb.sh build/config.yml README.md docs/appimage.md docs/build.md \
		docs/distribution.md CHANGELOG.md .release/config.json .release/conventions.md
	git commit --message "chore(release): $tag"
	release_commit="$(git rev-parse HEAD)"
	timing "commit"
fi

if [ "$tag_exists" = 0 ]; then
	create_tag
fi
if [ "$branch_exists" = 0 ]; then
	git branch "$release_branch" "$release_commit"
fi
timing "tag y rama"

verify_artifacts
timing "verificación final"

info ""
info "Gate de publicación:"
info "  versión:  $version"
info "  commit:   $release_commit"
info "  tag:      $tag"
info "  rama:     $release_branch"
info "  artefactos:"
info "    dist/better-rpt_${version}-1_amd64.deb"
info "    dist/BetterRPT-${version}-x86_64.AppImage"
info "    dist/SHA256SUMS"
info ""
info "Se ejecutará: git push --atomic origin main $tag $release_branch"
info "              gh release create $tag --notes-from-tag --verify-tag <artefactos>"

if [ "$no_push" = 1 ]; then
	info "Publicación omitida (--no-push)."
	info "Tiempo total: ${SECONDS}s"
	exit 0
fi
confirm "¿Publicar $tag?" || die "publicación cancelada; el commit, tag y rama quedan locales"
timing "confirmación de publicación"
publish
info "Tiempo total: ${SECONDS}s"
