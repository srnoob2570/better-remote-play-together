# Better RPT

Remote Play Together para juegos de Steam que no lo incluyen.

Better RPT usa SpaceWar (App ID 480) como donante fijo para abrir un juego
seleccionado dentro de un contexto que Steam reconoce como Remote Play Together.
La opción de lanzamiento de SpaceWar se modifica solo durante la sesión, con
copia de seguridad, verificación y restauración. Better RPT no modifica los
archivos del juego ni las opciones de otras aplicaciones.

> Estado: MVP Linux amd64. No promete compatibilidad universal. Léase la sección
> de limitaciones antes de usarlo.

## Requisitos

- Linux amd64 con glibc 2.35 o superior. El binario usa como máximo
  `GLIBC_2.34` y el paquete declara 2.35 como mínimo soportado.
- Steam nativo (no Flatpak ni Snap).
- SpaceWar (480) instalado en la misma cuenta de Steam. Es gratuito.
- Sesión gráfica X11 o Wayland y las bibliotecas OpenGL del sistema.
- El AppImage no empaqueta Wayland, GL ni X11: el sistema debe tener
  `libwayland-client`, `libwayland-cursor`, `libwayland-egl`, `libxkbcommon`,
  Mesa/GL y las bibliotecas X11 que use la sesión. Los contenedores mínimos sin
  estas bibliotecas no pueden ejecutarlo.
- Para ejecutar el AppImage con FUSE: `fuse3`/`fusermount3`. Si no está
  disponible, se puede usar `--appimage-extract-and-run`.
- No se necesita root para usar la aplicación. Better RPT se niega a arrancar
  como root.

## Instalación

### Paquete `.deb` (recomendado)

```bash
sha256sum -c SHA256SUMS
sudo apt install ./better-rpt_0.1.0-1_amd64.deb
```

Si `apt` no acepta el archivo local:

```bash
sudo dpkg -i ./better-rpt_0.1.0-1_amd64.deb
sudo apt-get install -f
```

El paquete instala:

```text
/usr/bin/better-rpt
/usr/lib/better-rpt/better-rpt-helper
/usr/share/applications/better-rpt.desktop
/usr/share/icons/hicolor/256x256/apps/better-rpt.png
/usr/share/doc/better-rpt/
```

No instala servicios, no crea configuración de usuario y no necesita `sudo`
para funcionar.

### AppImage (otras distribuciones)

```bash
chmod +x BetterRPT-0.1.0-x86_64.AppImage
./BetterRPT-0.1.0-x86_64.AppImage
```

Si FUSE no está disponible:

```bash
./BetterRPT-0.1.0-x86_64.AppImage --appimage-extract-and-run
# o
APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.1.0-x86_64.AppImage
```

El AppImage y el `.deb` comparten el helper estable y la configuración. No se
recomienda usar ambos a la vez. El AppImage se probó con FUSE directo y con
`--appimage-extract-and-run`; también se comprobó `--version` en Fedora 44 y
Arch con las bibliotecas gráficas del sistema, pero no se declara soporte de GUI
fuera de Ubuntu 22.04 en esta versión.

## Uso

1. Abre Steam e inicia sesión. Comprueba que SpaceWar aparece en tu
   biblioteca; si no, instálalo desde la ficha de la tienda (App ID 480).
2. Abre Better RPT.
3. Busca el juego y selecciónalo en la biblioteca. El panel lateral muestra su
   estado, si es nativo o Proton y los detalles técnicos.
4. Pulsa **Preparar y jugar**.
   - Si el juego usa Proton, aparece un aviso de compatibilidad desconocida la
     primera vez en cada ejecución.
   - Si Steam está abierto, Better RPT pide confirmación para cerrarlo y lo
     relanzará al terminar la preparación.
5. Cuando la sesión esté activa, abre el overlay de Steam con **Shift+Tab** y
   usa **Remote Play Together** para invitar a tus amigos. Better RPT no envía
   invitaciones por APIs internas.
6. Cuando termines, cierra el juego. En el panel de la sesión pulsa
   **Restaurar SpaceWar** para devolver su opción de lanzamiento al valor
   original. Si cierras Better RPT antes, la próxima apertura detectará la
   restauración pendiente.

El botón de restaurar solo actúa si el valor actual sigue siendo el que escribió
Better RPT. Si Steam u otro programa cambió la opción de lanzamiento, no se
sobrescribe y se muestra el conflicto con el detalle técnico.

## Qué hace exactamente

- Detecta una instalación nativa de Steam y sus bibliotecas.
- Lee `appinfo.vdf` para elegir el lanzamiento nativo o el mapping de Proton.
- Prepara SpaceWar como donante dentro de `localconfig.vdf`.
- Instala un helper estático en
  `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`.
- Lanza Steam/SpaceWar y el target como hijo del contexto donante.
- Mantiene un plan y un estado de sesión privados, con bloqueo y seguimiento
  por PID + hora de inicio.
- Restaura la opción de lanzamiento de SpaceWar y ofrece recuperación tras un
  cierre inesperado o un plan expirado.

Better RPT **no** usa APIs internas de Steam para invitar, no mata procesos y no
toca opciones de lanzamiento de otras aplicaciones.

## Limitaciones

- No hay compatibilidad universal. Que un juego arranque no garantiza que
  Remote Play Together funcione con todos sus modos, launchers o anti-cheat.
- **Anti-cheat:** los juegos con anti-cheat pueden bloquear el contexto del
  donante o rechazar el lanzamiento. No se intenta evadirlos.
- **Launchers externos:** los juegos que abren otro launcher pueden perder el
  contexto o requerir pasos adicionales.
- **Proton:** la compatibilidad de Remote Play Together con Proton es
  desconocida por diseño. Se permite intentarlo con aviso, pero no se
  garantiza.
- **Overlay:** las invitaciones dependen del overlay de Steam (`Shift+Tab`).
  Si el overlay no aparece, revisa la configuración de Steam y el juego.
- **Actualizaciones de Steam:** el formato de `localconfig.vdf` o `appinfo.vdf`
  puede cambiar. Better RPT muestra un error y no escribe datos dudosos.
- **Flatpak/Snap:** se detectan, pero no se lanzan ni se modifican. Solo se
  soporta Steam nativo.
- **Plataformas:** solo Linux amd64. No Windows, macOS, Steam Deck ni Game Mode.
- **AppImage:** no cubre Alpine/musl, glibc anterior a 2.35, 32-bit ni arm64.
  En Fedora 44 y Arch solo se verificó `--version` con las bibliotecas gráficas
  instaladas; no hay pruebas de GUI de extremo a extremo ni soporte declarado
  en esas distribuciones.
- **Accesibilidad:** hay navegación por teclado, atajos y chips con texto.
  Fyne 2.8 no implementa lector de pantalla en Linux, y no es un requisito
  bloqueante del MVP.
- **Una sesión a la vez:** no se pueden preparar dos donantes ni lanzar dos
  targets concurrentes.

## Datos y privacidad

Better RPT no descarga código ni envía datos por red. Todo queda en el equipo:

- Configuración: `$XDG_CONFIG_HOME/better-rpt/config.json`
  (por defecto `~/.config/better-rpt/config.json`).
- Helper estable: `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`
  (por defecto `~/.local/share/better-rpt/bin/better-rpt-helper`).
- Copias de `localconfig.vdf`: `$XDG_DATA_HOME/better-rpt/backups/`
  (directorio `0700`, archivos `0600`, retención limitada).
- Plan/estado de la sesión: `$XDG_RUNTIME_DIR/better-rpt/`; no es dato
  persistente.

## Problemas frecuentes

### «No se encontró una instalación nativa de Steam»

Comprueba que Steam esté instalado en `~/.local/share/Steam` o
`~/.steam/steam`, y que no estés usando solo Flatpak/Snap. Pulsa **Actualizar**
en Better RPT. Si usas una ruta no estándar, el MVP no la detecta.

### «SpaceWar no está instalado»

Instala SpaceWar (gratuito, App ID 480) desde Steam. Better RPT ofrece el botón
**Añadir SpaceWar**; si Steam no responde, instálalo manualmente desde la tienda
y vuelve a abrir Better RPT.

### Steam no se cierra o no vuelve a abrir

Cierra Steam manualmente, espera unos segundos y vuelve a intentarlo. Better RPT
solo escribe cuando Steam está cerrado. Si el cierre queda a medias, no se
modifica `localconfig.vdf` y se muestra el error.

### «Restauración pendiente» o conflicto

Abre Better RPT y pulsa **Restaurar SpaceWar** antes de desinstalar. Si la opción
de lanzamiento cambió fuera de Better RPT, la app no la sobrescribe: restaura
manualmente el valor original o reinstala la misma versión para recuperar la
copia si existe.

### El AppImage no arranca

Instala `fuse3` o ejecuta con `--appimage-extract-and-run`. En Wayland, si hay
problemas de decoración o GL, prueba `FYNE_PLATFORM=x11` (o `wayland` si tu
sesión lo soporta). La autodetección es el comportamiento recomendado.

### El juego arranca pero no aparece Remote Play Together

Abre el overlay con `Shift+Tab` y comprueba que Steam esté en línea. Si el juego
usa un launcher externo, anti-cheat o un modo que abandona el contexto del
donante, no hay una solución universal en este MVP.

## Desinstalación

```bash
sudo apt remove better-rpt
# o, para eliminar también los archivos de documentación del paquete:
sudo apt purge better-rpt
```

El paquete no borra configuración, backups ni el helper estable. Para eliminar
todos los datos del usuario:

```bash
rm -rf ~/.config/better-rpt
rm -rf ~/.local/share/better-rpt
rm -rf "${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/better-rpt"
```

Para el AppImage, borra el archivo. Los datos anteriores siguen las mismas rutas.
**Antes de desinstalar, restaura SpaceWar** si preparaste una sesión.

## Desarrollo local (sin instalar)

No hace falta instalar el `.deb` ni el AppImage para desarrollar o probar la
app. `make dev` compila y ejecuta el binario Fyne desde el repositorio:

```bash
make dev
# equivalente sin Make:
GOTOOLCHAIN=go1.27.2 go run ./cmd/better-rpt
```

El `Makefile` usa `GOTOOLCHAIN=go1.27.2` y no necesita `sudo`. La versión en
desarrollo se muestra como `dev`; la versión publicada (`0.1.0`) se inyecta
con `-ldflags` al compilar el paquete. Fyne no recarga los cambios
automáticamente: detén con `Ctrl+C`, edita y vuelve a ejecutar `make dev`.

Si faltan dependencias de Fyne en Ubuntu/Debian:

```bash
sudo apt-get install \
  build-essential pkgconf \
  libgl1-mesa-dev libwayland-dev libxkbcommon-dev \
  libx11-dev libxext-dev libxrender-dev libxfixes-dev \
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

Comandos de desarrollo:

```bash
make dev                         # compila y abre la GUI
make dev ARGS=--version          # ejecuta con argumentos
make build                       # dist/better-rpt, binario local
make helper                      # dist/better-rpt-helper, estático
make test                        # go test ./...
make vet                         # go vet ./...
make test-race                   # go test -race ./internal/...
```

`make build` produce un binario para desarrollo en el equipo; **no** es la
compilación final. El `.deb` y el AppImage de publicación se construyen en
Ubuntu 22.04 con la receta de [`docs/build.md`](docs/build.md) para mantener la
compatibilidad con glibc 2.35.

### Probar con Steam simulado

`make dev` usa tu `HOME` y tus directorios XDG reales por defecto, así que puede
leer tu Steam y crear configuración y un helper de usuario. Para una prueba
aislada, genera los datos de prueba y arranca con un `HOME` y XDG temporales:

```bash
base=$(mktemp -d)
mkdir -p "$base/home" "$base/config" "$base/data" "$base/run"
bash testdata/steam/materialize.sh "$base/fixtures" >/dev/null
cp -a "$base/fixtures/primary" "$base/data/Steam"
chmod 700 "$base/run"

# Conserva las cachés de Go aunque el HOME sea temporal.
go_cache="${GOCACHE:-$HOME/.cache/go-build}"
go_modcache="${GOMODCACHE:-$HOME/go/pkg/mod}"
go_path="${GOPATH:-$HOME/go}"

HOME="$base/home" \
GOCACHE="$go_cache" \
GOMODCACHE="$go_modcache" \
GOPATH="$go_path" \
XDG_CONFIG_HOME="$base/config" \
XDG_DATA_HOME="$base/data" \
XDG_RUNTIME_DIR="$base/run" \
make dev
```

La primera ejecución puede descargar Go 1.27.2 y los módulos de Fyne; después
reutiliza esas cachés.

Esta prueba no toca `~/.local/share/Steam` ni lanza Steam o juegos reales. El
conjunto incluye SpaceWar y targets nativo/Proton simulados.

### Probar el paquete instalado

Solo para validar la integración real del paquete:

```bash
sudo apt install ./dist/better-rpt_0.1.0-1_amd64.deb
better-rpt
# o
./dist/BetterRPT-0.1.0-x86_64.AppImage
```

## Compilación reproducible

La receta completa está en [`docs/build.md`](docs/build.md) y
[`docs/appimage.md`](docs/appimage.md). Los artefactos de esta versión se
construyeron en un contenedor Ubuntu 22.04 con Go 1.27.2, Fyne 2.8.2, CGO,
`-trimpath`, `-buildvcs=false`, `-ldflags '-s -w'`, nfpm 2.48.0 y
`SOURCE_DATE_EPOCH=1700000000`. El AppImage usa el runtime type2 fijado y
`appimagetool` con hash publicado. La instalación limpia del `.deb` y el arranque
GUI se probaron en Ubuntu 22.04; el AppImage se probó con FUSE y con extracción.

Pruebas automatizadas:

```bash
GOTOOLCHAIN=go1.27.2 go test ./...
GOTOOLCHAIN=go1.27.2 go vet ./...
GOTOOLCHAIN=go1.27.2 go test -race ./internal/...
```

Las pruebas usan datos de Steam simulados y dobles de prueba; nunca lanzan
Steam ni juegos reales.

## Estado de verificación de esta versión

- 223 pruebas en 16 paquetes; `go vet` y `go test -race` pasan sin errores.
- Binario y helper construidos en Ubuntu 22.04; símbolo máximo `GLIBC_2.34`.
  `dpkg-shlibdeps` pidió `libc6 (>= 2.34)`; el paquete declara `2.35` como
  mínimo soportado.
- `.deb` instalado desde cero en Ubuntu 22.04, con `dpkg-shlibdeps`, arranque
  GUI como usuario no root y rechazo de root comprobados.
- AppImage arrancado directamente con FUSE y con `--appimage-extract-and-run`;
  GUI bajo Xvfb y helper estable probados. Fedora 44 y Arch solo se probaron con
  `--version` y bibliotecas gráficas del sistema.
- Dos compilaciones consecutivas del AppImage y del `.deb` dieron el mismo
  SHA-256 en cada formato. Consulta `dist/SHA256SUMS`.
- **No se ha completado una sesión de extremo a extremo con Steam real y
  SpaceWar + un target nativo/Proton en este entorno.** No se afirma que esa
  combinación funcione; es la verificación pendiente antes de una publicación
  pública.

## Licencia

MIT. Consulta [`LICENSE`](LICENSE).
