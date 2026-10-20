# Compilación y empaquetado

Receta verificada para el ticket 18. El `.deb` final se construye en un
contenedor **Ubuntu 22.04** (amd64), con Fyne v2.8.2, Go 1.27.2 y nfpm v2.48.0.
El binario resultante usa como máximo `GLIBC_2.34`; el paquete declara
`libc6 >= 2.35` como mínimo soportado. Se probó una instalación limpia del
`.deb` en Ubuntu 22.04, el arranque GUI como usuario no root, el rechazo de root
y `dpkg-shlibdeps`. El AppImage se construye sobre la misma base Ubuntu 22.04; su
receta está en `docs/appimage.md`.

## Dependencias de compilación (Debian/Ubuntu)

```bash
sudo apt-get install \
  build-essential pkgconf \
  libgl-dev libwayland-dev libxkbcommon-dev \
  libx11-dev libxext-dev libxrender-dev libxfixes-dev \
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

- La compilación final usa `libgl1-mesa-dev` en Ubuntu 22.04; en Ubuntu 26.04 el
  paquete equivalente es `libgl-dev`. No se necesita WebKitGTK, GTK para
  compilar, `libdecor-dev` ni `libegl-dev`.
- GLFW enlaza X11 y Wayland; por eso se necesitan ambos conjuntos de headers.

## Desarrollo local

Para iterar no hace falta instalar el `.deb` ni el AppImage:

```bash
make dev       # go run ./cmd/better-rpt
make build     # dist/better-rpt local, versión 0.1.1
make helper    # dist/better-rpt-helper estático
make all       # compila la app y el helper
```

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

El `Makefile` fija `GOTOOLCHAIN=go1.27.2` y no usa `sudo`. `make dev` usa el
`HOME`/XDG reales; para una prueba aislada con los fixtures sintéticos, consulta
la sección «Desarrollo local» del [`README.md`](../README.md). La build de
release **no** es `make build`: se hace en el contenedor 22.04 descrito abajo.

## Go 1.27

Ubuntu 22.04 empaqueta Go 1.18; Ubuntu 26.04, Go 1.26. El proyecto declara
`go 1.27.2` y usa Go 1.27.2:

```bash
GOTOOLCHAIN=go1.27.2 go version
```

Go 1.26 puede descargar Go 1.27.2 automáticamente si `GOTOOLCHAIN` está en
`auto`. Para compilar sin conexión, instala Go 1.27.2 desde go.dev o copia esa
versión al repositorio. El usuario final no compila: solo instala el `.deb`.

## Contenedor Ubuntu 22.04

El `.deb` final no se construye en Ubuntu 26.04: se usa una imagen limpia de
Ubuntu 22.04 para que el binario enlace contra glibc 2.35. Ubuntu 22.04 trae
Go 1.18; instala el tarball oficial de Go 1.27.2.

`make deb` automatiza esta receta con `packaging/Dockerfile.build`: compila
ambos binarios dentro de Ubuntu 22.04, los copia a `dist/` y empaqueta con nfpm
en el host.

```bash
make deb                 # dist/better-rpt_0.1.1-1_amd64.deb
make deb VERSION=0.2.0   # otra versión
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
    libgl1-mesa-dev libwayland-dev libxkbcommon-dev \
    libx11-dev libxext-dev libxrender-dev libxfixes-dev \
    libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
  curl -fsSL https://go.dev/dl/go1.27.2.linux-amd64.tar.gz -o /tmp/go.tgz
  tar -C /usr/local -xzf /tmp/go.tgz
  export PATH=/usr/local/go/bin:$PATH
  export GOTOOLCHAIN=local
  go version
  CGO_ENABLED=1 go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=0.1.1" \
    -o dist/better-rpt ./cmd/better-rpt
  CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=0.1.1" \
    -o dist/better-rpt-helper ./cmd/better-rpt-helper
'
```

El binario Fyne quedó en 25,6 MB y el helper estático en 3,4 MB. Verificar la
versión de glibc con `objdump -T dist/better-rpt | grep GLIBC` (máximo
`GLIBC_2.34`) y `dpkg-shlibdeps`; no asumirla por la versión del contenedor.
La compilación en Ubuntu 22.04 se completó; no fue necesario usar Ubuntu 24.04 como
alternativa.

## Compilar

```bash
mkdir -p dist
CGO_ENABLED=1 GOTOOLCHAIN=go1.27.2 \
  go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o dist/better-rpt ./cmd/better-rpt

CGO_ENABLED=0 GOTOOLCHAIN=go1.27.2 \
  go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o dist/better-rpt-helper ./cmd/better-rpt-helper
```

El helper no tiene interfaz gráfica, es estático y no usa Fyne; la app lo copia a
`$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`. La versión final es
`0.1.1`. `-trimpath` y `-buildvcs=false` son necesarios para la
reproducibilidad; `-s -w` reduce el binario. El binario Fyne final mide
25,6 MB y el helper estático 3,4 MB; `--version` reporta `0.1.1` en ambos.

## nfpm

```bash
GOTOOLCHAIN=go1.27.2 go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
VERSION=0.1.1 SOURCE_DATE_EPOCH=1700000000 \
  "$HOME/go/bin/nfpm" package --config packaging/nfpm.yaml --packager deb \
  --target dist/
```

El destino es un directorio, así que nfpm genera
`dist/better-rpt_0.1.1-1_amd64.deb`. `packaging/nfpm.yaml` toma `version` de la
variable `VERSION`; nfpm expande
`${VERSION}` del entorno al leer la configuración. `make deb` ya define esa
variable.

Configuración validada (adaptar maintainer, homepage, versión y rutas a la
estructura del repositorio; la prueba usó rutas planas y aquí se muestran
`dist/` y `packaging/`):

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
StartupWMClass=Better RPT
```

`desktop-file-validate` pasa sin avisos. El `WM_CLASS` observado con Fyne
`app.New()` fue `"Better RPT", "Better RPT"`; si la app final usa
`app.NewWithID`, hay que volver a comprobarlo y ajustar `StartupWMClass`.

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
/usr/share/icons/hicolor/256x256/apps/better-rpt.png
/usr/share/doc/better-rpt/README.md
/usr/share/doc/better-rpt/LICENSE
/usr/share/doc/better-rpt/copyright
```

No se instala configuración de usuario ni servicios. `Installed-Size` es
29.730 KB; el `.deb` final mide 13,9 MB (14.544.880 bytes). El icono es PNG
256×256 de 8 bits.

## Dependencias de runtime

`ldd` solo muestra las bibliotecas enlazadas directamente:

- `libc6`, `libgl1`, `libwayland-client0`, `libx11-6`.

GLFW carga por `dlopen` otras bibliotecas que `dpkg-shlibdeps` no ve:

- Wayland: `libwayland-cursor0`, `libwayland-egl1`, `libxkbcommon0`.
- EGL: `libegl1` (y `libegl-mesa0` como dependencia).
- X11: `libx11-xcb1`, `libxcursor1`, `libxi6`, `libxinerama1`, `libxrandr2`,
  `libxxf86vm1`.
- Decoraciones Wayland opcionales: `libdecor-0-0` y
  `libdecor-0-plugin-1-gtk` (este último arrastra GTK3, no WebKitGTK).

`dpkg-shlibdeps -O` sobre el binario final en Ubuntu 22.04 devolvió:

```text
shlibs:Depends=libc6 (>= 2.34), libgl1, libwayland-client0 (>= 1.20.0)
```

El binario necesita como máximo `GLIBC_2.34`; el paquete declara `libc6 (>= 2.35)`
como mínimo soportado. La lista de `Depends` añade manualmente las
bibliotecas de `dlopen`.
`libdecor` queda en `Recommends`: en un compositor con decoraciones de servidor
no se necesita; sin él, GLFW puede quedar sin decoraciones o usar X11 como
alternativa. Si se quiere minimizar el paquete, el plugin puede pasar a `Suggests`.

## Wayland y X11

Probado con el binario extraído del `.deb`:

- **X11**: Xvfb, arranca y permanece vivo; `WM_CLASS` correcto.
- **Wayland puro**: con `DISPLAY` sin definir y `WAYLAND_DISPLAY` presente,
  Fyne elige Wayland y carga `libwayland-cursor`, `libwayland-egl`,
  `libxkbcommon`, EGL y `libdecor`.
- **Sesión mixta**: con `WAYLAND_DISPLAY` y `DISPLAY` definidos, Fyne 2.8.2
  consulta `zxdg_decoration_manager_v1`; si el compositor no ofrece
  decoraciones de servidor, fuerza X11 como alternativa. Se puede forzar Wayland
  con `FYNE_PLATFORM=wayland` y X11 con `FYNE_PLATFORM=x11`.
- Recomendación: dejar la autodetección; documentar `FYNE_PLATFORM` para
  la resolución de problemas. No fijarlo en el `.desktop`, porque rompería el otro backend.

## Accesibilidad en Linux

Fyne 2.8.2 implementa accesibilidad para macOS, Windows, iOS y Android. En Linux,
`internal/driver/glfw/accessibility_notdarwin.go` no hace nada; la etiqueta de
compilación `accessibility` compila pero **no añade soporte de lector de pantalla
en Linux**.

Por tanto:

- No se publica una compilación `accessibility` separada: no añadiría nada.
- Sí se mantienen navegación por teclado, nombres accesibles, chips con texto y
  contraste.
- Si el lector de pantalla pasa a ser requisito bloqueante, Fyne 2.8 no lo cubre
  en Linux; habría que evaluar Wails/GTK u otro toolkit, que añadiría WebKitGTK.
  No bloquea el MVP.

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
- Instalación real con `apt install ./better-rpt_0.1.1-1_amd64.deb` en un
  contenedor Ubuntu 22.04 limpio.
- `dpkg-shlibdeps -O /usr/bin/better-rpt` -> `libc6 (>= 2.34)`.
- Ejecución como root rechazada con código 1; GUI como usuario normal bajo
  Xvfb con HOME/XDG temporales.
- Dos compilaciones del AppImage y dos del `.deb` con
  `SOURCE_DATE_EPOCH=1700000000` dieron el mismo SHA-256 en cada formato;
  `dist/SHA256SUMS` contiene los hashes publicados.
- El paquete final está en `dist/`; no se usa un artefacto efímero de
  `/tmp` para la entrega.
