# Compilación y empaquetado

Receta verificada para la versión 0.2.0. El `.deb` final se construye en un
contenedor **Ubuntu 22.04** (amd64), con Wails v3.0.0-beta.33/GTK3/WebKit2GTK
4.1, Go 1.27.3 y nfpm v2.48.0. El paquete declara `libc6 >= 2.35` como mínimo
soportado. Se probó una instalación limpia del `.deb` en Ubuntu 22.04, el
arranque GUI como usuario no root en X11 y Wayland, el rechazo de root y
`dpkg-shlibdeps`. El AppImage se construye sobre la misma base Ubuntu 22.04; su
receta está en `docs/appimage.md`.

## Dependencias de compilación

La UI React usa Wails v3 con GTK3/WebKit2GTK 4.1. El `Makefile` instala los
paquetes del sistema con:

```bash
make setup-check   # solo informa; no necesita sudo
sudo make setup    # instala lo que falte; requiere privilegios
```

`make setup` detecta apt, dnf, pacman o zypper y cubre gcc y make, pkg-config,
GTK3, WebKit2GTK 4.1, Mesa/OpenGL, Wayland, xkbcommon y las cabeceras X11. No
gestiona Go, Node/npm ni el CLI de Wails.

La lista manual para Debian/Ubuntu es:

```bash
sudo apt-get install \
  build-essential pkgconf \
  libgtk-3-dev libwebkit2gtk-4.1-dev \
  libgl-dev libwayland-dev libxkbcommon-dev \
  libx11-dev libxext-dev libxrender-dev libxfixes-dev \
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

- La compilación final usa `libgl1-mesa-dev` en Ubuntu 22.04; en Ubuntu 26.04 el
  paquete equivalente es `libgl-dev`.
- Wails v3 se instala con el tag `gtk3`; el CLI por defecto busca GTK4:
  `go install -tags gtk3 github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.33`.
- El frontend requiere Node 20.19+ (probado con Node 24) y `npm ci`.
- `make deb` depende de `make frontend`: genera los bindings y
  `cmd/better-rpt/dist`, y después compila dentro de Ubuntu 22.04. No compila el
  binario del host; el AppImage usa los binarios del contenedor. `make helper`
  no cambia.

## Desarrollo local

Para iterar no hace falta instalar el `.deb` ni el AppImage:

```bash
# una vez, con Go y Node/npm instalados; añade $GOPATH/bin al PATH
go install -tags gtk3 github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.33
cd frontend && npm ci && cd ..

make dev       # Wails dev + Vite; ventana con recarga en caliente
make build     # frontend + dist/better-rpt con assets embebidos
make helper    # dist/better-rpt-helper estático
make all       # compila la app y el helper
```

`make dev` arranca Vite en el puerto 9245 y el binario con el proxy de Wails;
si el terminal se cierra de golpe, el siguiente `make dev` retira los procesos
huérfanos de este repositorio antes de arrancar. `make dev-stop` fuerza esa
limpieza. `make build` ejecuta `npm run build`, genera los bindings y compila con
`-tags 'gtk3 production'`. `go test ./...` funciona sin GTK porque el paquete
`cmd/better-rpt` tiene un stub sin tag; para compilar el shell real usa
`go test -tags gtk3 ./cmd/better-rpt`.

Para instalar ambos binarios y el lanzador en el sistema:

```bash
make all
sudo make install
```

`make install` no compila: copia los binarios ya generados por `make all` a
`/usr/bin`, el helper a `/usr/lib/better-rpt` y el lanzador/iconos a
`/usr/share`. No invoca `sudo`: ejecútalo con `sudo` o usa
`DESTDIR=/ruta make install` para preparar un árbol de instalación. Ejecuta
`make all` antes de `sudo make install` para que la compilación se haga como tu
usuario.

Los targets de compilación fijan `GOTOOLCHAIN=go1.27.3` y no usan `sudo`; solo
`make setup` lo requiere para instalar paquetes del sistema. `make dev` usa el
`HOME`/XDG reales; para una prueba aislada con los fixtures sintéticos, consulta
la sección «Desarrollo local» del [`README.md`](../README.md). La build de
release **no** es `make build`: se hace en el contenedor 22.04 descrito abajo.

## Go 1.27

Ubuntu 22.04 empaqueta Go 1.18; Ubuntu 26.04, Go 1.26. El proyecto declara
`go 1.27.3` y usa Go 1.27.3:

```bash
GOTOOLCHAIN=go1.27.3 go version
```

Go 1.26 puede descargar Go 1.27.3 automáticamente si `GOTOOLCHAIN` está en
`auto`. Para compilar sin conexión, instala Go 1.27.3 desde go.dev o copia esa
versión al repositorio. El usuario final no compila: solo instala el `.deb`.

## Contenedor Ubuntu 22.04

El `.deb` final no se construye en Ubuntu 26.04: se usa una imagen limpia de
Ubuntu 22.04 para que el binario enlace contra glibc 2.35. Ubuntu 22.04 trae
Go 1.18; instala el tarball oficial de Go 1.27.3.

`make deb` automatiza esta receta con `packaging/Dockerfile.build`: compila
ambos binarios dentro de Ubuntu 22.04, los copia a `dist/` y empaqueta con nfpm
en el host.

```bash
make deb                 # dist/better-rpt_0.2.4-1_amd64.deb
make deb VERSION=0.2.4   # otra versión
```

Requiere Docker y nfpm v2.48.0 en `$HOME/go/bin/nfpm`, en `PATH` o en la ruta
indicada por `NFPM`. La receta manual equivalente, ejecutada con Docker server
29.7.2, es:

```bash
docker run --rm -v "$PWD:/src" -w /src ubuntu:22.04 bash -c '
  set -e
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates curl build-essential pkgconf \
    libgtk-3-dev libwebkit2gtk-4.1-dev \
    libgl1-mesa-dev libwayland-dev libxkbcommon-dev \
    libx11-dev libxext-dev libxrender-dev libxfixes-dev \
    libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
  curl -fsSL https://go.dev/dl/go1.27.3.linux-amd64.tar.gz -o /tmp/go.tgz
  tar -C /usr/local -xzf /tmp/go.tgz
  export PATH=/usr/local/go/bin:$PATH
  export GOTOOLCHAIN=local
  go version
  CGO_ENABLED=1 go build -tags "gtk3 production" -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=0.2.4" \
    -o dist/better-rpt ./cmd/better-rpt
  CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=0.2.4" \
    -o dist/better-rpt-helper ./cmd/better-rpt-helper
'
```

El binario Wails quedó en 11,5 MB y el helper estático en 3,4 MB. El binario
embebe `cmd/better-rpt/dist`; `make frontend` genera ese directorio antes de
compilar (y `make deb` lo invoca). Verificar la versión de glibc con `objdump -T dist/better-rpt | grep GLIBC`
y `dpkg-shlibdeps`; no asumirla por la versión del contenedor. La compilación en
Ubuntu 22.04 se completó; no fue necesario usar Ubuntu 24.04 como alternativa.

## Compilar

```bash
mkdir -p dist
CGO_ENABLED=1 GOTOOLCHAIN=go1.27.3 \
  go build -tags "gtk3 production" -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o dist/better-rpt ./cmd/better-rpt

CGO_ENABLED=0 GOTOOLCHAIN=go1.27.3 \
  go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o dist/better-rpt-helper ./cmd/better-rpt-helper
```

El binario de la app necesita `cmd/better-rpt/dist` (frontend y assets); usa
`make frontend` para generarlo, o copia el resultado de `cd frontend && npm run
build` en ese directorio. El helper no tiene interfaz gráfica, es estático y no enlaza GTK ni WebKit; la
app lo copia a `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`.
La versión final es `0.2.4`. `-trimpath` y `-buildvcs=false` son necesarios para
la reproducibilidad; `-s -w` reduce el binario. El binario Wails final mide
11,5 MB y el helper estático 3,4 MB; `--version` reporta `0.2.4` en ambos.

## nfpm

```bash
GOTOOLCHAIN=go1.27.3 go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
VERSION=0.2.4 SOURCE_DATE_EPOCH=1700000000 \
  "$HOME/go/bin/nfpm" package --config packaging/nfpm.yaml --packager deb \
  --target dist/
```

El destino es un directorio, así que nfpm genera
`dist/better-rpt_0.2.4-1_amd64.deb`. `packaging/nfpm.yaml` toma `version` de la
variable `VERSION`; nfpm expande
`${VERSION}` del entorno al leer la configuración. `make deb` ya define esa
variable.

Fragmento representativo; la fuente autoritativa es `packaging/nfpm.yaml`. El
archivo real incluye los nueve iconos hicolor de 16 a 512 px y el resto de
metadatos:

```yaml
name: better-rpt
arch: amd64
platform: linux
version: ${VERSION}
release: 1
section: games
priority: optional
maintainer: srnoob2570 <302417634+srnoob2570@users.noreply.github.com>
description: |
  Launch Steam games with Remote Play Together using SpaceWar as the donor.
vendor: Better RPT
license: MIT
scripts:
  preremove: ./packaging/prerm.sh
depends:
  - libc6 (>= 2.35)
  - libgtk-3-0t64 | libgtk-3-0
  - libwebkit2gtk-4.1-0
  - libgl1
  - libegl1
  - libwayland-client0 (>= 1.20.0)
  - libwayland-cursor0
  - libwayland-egl1
  - libxkbcommon0
  - libx11-6
  - libx11-xcb1
  - libxcursor1
  - libxi6
  - libxinerama1
  - libxrandr2
  - libxxf86vm1
recommends:
  - libdecor-0-0
  - libdecor-0-plugin-1-gtk
contents:
  - src: ./dist/better-rpt
    dst: /usr/bin/better-rpt
    file_info:
      mode: 0755
  - src: ./dist/better-rpt-helper
    dst: /usr/lib/better-rpt/better-rpt-helper
    file_info:
      mode: 0755
  - src: ./packaging/better-rpt.desktop
    dst: /usr/share/applications/better-rpt.desktop
    file_info:
      mode: 0644
  - src: ./packaging/better-rpt.png
    dst: /usr/share/icons/hicolor/256x256/apps/better-rpt.png
    file_info:
      mode: 0644
  - src: ./README.md
    dst: /usr/share/doc/better-rpt/README.md
    file_info:
      mode: 0644
  - src: ./LICENSE
    dst: /usr/share/doc/better-rpt/LICENSE
    file_info:
      mode: 0644
  - src: ./packaging/copyright
    dst: /usr/share/doc/better-rpt/copyright
    file_info:
      mode: 0644
```

El `.desktop` validado:

```ini
[Desktop Entry]
Type=Application
Name=Better RPT
GenericName=Remote Play Together Enabler
Comment=Launch Steam games with Remote Play Together using SpaceWar
Exec=better-rpt
Icon=better-rpt
Terminal=false
Categories=Game;
Keywords=Steam;Remote Play;RPT;
StartupWMClass=Better-rpt
```

`desktop-file-validate` pasa sin avisos. El `WM_CLASS` observado con
Wails/GTK3 es `"better-rpt", "Better-rpt"`, que coincide con
`StartupWMClass=Better-rpt`.

## Caché de compilación

`make deb` reutiliza las capas de la imagen Docker y dos volúmenes de Go:
`better-rpt-deb-go-build` (paquetes compilados) y `better-rpt-deb-go-mod`
(módulos descargados). Cambia el prefijo con `DEB_CACHE=...` o desactiva la
caché con `DEB_NO_CACHE=1`.

Para vaciar la caché:

```bash
docker volume rm better-rpt-deb-go-build better-rpt-deb-go-mod
```

## Contenido del paquete

```text
/usr/bin/better-rpt
/usr/lib/better-rpt/better-rpt-helper
/usr/share/applications/better-rpt.desktop
/usr/share/icons/hicolor/{16x16,24x24,32x32,48x48,64x64,128x128,256x256,512x512}/apps/better-rpt.png
/usr/share/doc/better-rpt/README.md
/usr/share/doc/better-rpt/LICENSE
/usr/share/doc/better-rpt/copyright
```

No se instala configuración de usuario ni servicios. `Installed-Size` es
18.350 KB; el `.deb` final mide 8,0 MB (8.416.850 bytes). Se instalan los nueve
iconos hicolor de 16 a 512 px.

## Dependencias de runtime

El binario Wails/GTK3 enlaza GTK3 y WebKit2GTK 4.1. `packaging/nfpm.yaml`
declara, además de `libc6 (>= 2.35)`:

- `libgtk-3-0t64 | libgtk-3-0` y `libwebkit2gtk-4.1-0` (obligatorias).
- `libgl1`, `libegl1`, Wayland (`libwayland-client0`, `libwayland-cursor0`,
  `libwayland-egl1`, `libxkbcommon0`) y X11 (`libx11-6`, `libx11-xcb1`,
  `libxcursor1`, `libxi6`, `libxinerama1`, `libxrandr2`, `libxxf86vm1`).
- `libdecor-0-0` y `libdecor-0-plugin-1-gtk` quedan en `Recommends`; sin ellos
  GTK puede usar decoraciones de servidor o X11.

`dpkg-deb -I dist/better-rpt_0.2.4-1_amd64.deb` confirma esas dependencias y
recomendaciones. La instalación limpia en Ubuntu 22.04 resolvió GTK3 y
WebKit2GTK 4.1 desde los repositorios.

## Wayland y X11

Probado con el `.deb` instalado en Ubuntu 22.04:

- **X11**: Xvfb, la ventana `Better RPT` aparece y el proceso permanece vivo;
  arranque medido en 432 ms.
- **Wayland puro**: Weston headless (`--backend=headless-backend.so`) con
  `GDK_BACKEND=wayland`, `WAYLAND_DISPLAY` y sin `DISPLAY`; la app queda
  `GUI_RUNNING`, carga los assets y registra `WebKit2GTK 4.1`. Los avisos de
  asiento/portal son del compositor headless sin escritorio.
- **Recomendación**: dejar que GTK/Wails detecte el backend; no fijar
  `GDK_BACKEND` en el `.desktop`. El AppRun solo comprueba las bibliotecas.

## Accesibilidad en Linux

La UI React se renderiza en WebKitGTK y conserva navegación por teclado,
nombres accesibles, foco visible y contraste. La auditoría de paridad,
accesibilidad y snapshots se ejecuta con Playwright y axe; no se publica una
variante `accessibility` separada.

## Compatibilidad de glibc

La compilación final se hizo en Ubuntu 22.04 (glibc 2.35) y declara
`libc6 >= 2.35`; eso cubre Ubuntu 22.04+, Debian 12+, Linux Mint 21+,
Pop!_OS 22.04+ y derivados.

- `objdump -T` muestra como máximo `GLIBC_2.34`; `dpkg-shlibdeps` pidió
  `libc6 (>= 2.34)`. No se confía solo en la versión del contenedor.
- La instalación limpia del `.deb` se probó en Ubuntu 22.04; también el
  `--version`, el arranque GUI como usuario no root y el rechazo de root.
- Fedora, Arch y otras distros no usan `.deb`: se cubren con el AppImage de
  [`docs/appimage.md`](appimage.md), construido sobre la misma base Ubuntu 22.04.
  Fedora 44 y Arch solo se probaron con `--version` y bibliotecas gráficas del
  sistema; no hay pruebas de GUI de extremo a extremo ni soporte declarado
  allí. Alpine/musl, glibc antigua
  y arm64 no están soportadas.

## Pruebas

```bash
cd frontend
npm test             # Vitest: biblioteca, sesión e i18n con bindings mockeados
npm run test:e2e     # Playwright: paridad, snapshots y axe; usa Chrome del sistema
```

La suite Playwright usa `e2e.html` y un transporte mock; no descarga Chromium y
no mete fixtures en el bundle de producción. Los snapshots viven en
`frontend/e2e/visual.spec.ts-snapshots/`. Para cambios visuales intencionados:

```bash
npm run test:e2e:update
```

No hay workflow de CI todavía: el gate es local y requiere Go 1.27.3, Node,
GTK3/WebKit2GTK 4.1, Xvfb y Chrome. Un CI futuro necesitará esa matriz; no se
añadió en esta versión para no duplicar el entorno de escritorio.

El smoke real se ejecutó con el binario de producción y XDG temporal bajo Xvfb:
arranque, assets, `steam catalog refreshed libraries=1 games=5 warnings=0`,
cambio a inglés (persistido en `config.json`), cierre limpio con `WM_DELETE`
(exit 0). No se pulsó Preparar y jugar.

## Verificación

```bash
dpkg-deb -I dist/better-rpt_*.deb
dpkg-deb -c dist/better-rpt_*.deb
desktop-file-validate packaging/better-rpt.desktop
apt-get install --simulate ./dist/better-rpt_*.deb

mkdir -p /tmp/better-rpt-root
dpkg-deb -x dist/better-rpt_*.deb /tmp/better-rpt-root
/tmp/better-rpt-root/usr/bin/better-rpt --version

xvfb-run -a timeout 3s /tmp/better-rpt-root/usr/bin/better-rpt
env -u DISPLAY XDG_RUNTIME_DIR=/run/user/1000 WAYLAND_DISPLAY=wayland-0 \
  timeout 3s /tmp/better-rpt-root/usr/bin/better-rpt
```

La verificación de esta versión incluyó:

- `dpkg-deb -I/-c` y `desktop-file-validate` sin errores.
- Instalación real del `.deb` en un contenedor Ubuntu 22.04 limpio.
- `dpkg-shlibdeps -O /usr/bin/better-rpt` -> `libc6 (>= 2.34)`.
- Ejecución como root rechazada con código 1; GUI como usuario normal bajo
  Xvfb con HOME/XDG temporales.
- El pipeline de release repite la compilación y compara los SHA-256 antes de
  publicar; en esta ronda se construyó una vez cada formato. Los hashes finales
  están en `dist/SHA256SUMS` y en el historial de la release.
- El paquete final está en `dist/`; no se usa un artefacto efímero de
  `/tmp` para la entrega.

## Publicar una release

`make release` automatiza el bump, el changelog, los artefactos, el commit, el
tag, la rama y la publicación en GitHub. `make release-plan` muestra el plan sin
tocar nada. El detalle de opciones, reanudación y requisitos está en
[`release.md`](release.md).
