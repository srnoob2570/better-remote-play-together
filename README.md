# Better RPT

> [!WARNING]
> **Origen y uso de IA.** Better RPT se basa en la idea de Remote Play Detached
> (RPD), de smaTc
> ([github.com/smaTc/RemotePlayDetached](https://github.com/smaTc/RemotePlayDetached),
> licencia MIT), y no reutiliza su código. El mantenedor usó herramientas de IA
> para desarrollar la aplicación, las pruebas y la documentación. El código y los
> textos pueden contener errores pese a las pruebas. Si encuentras un fallo,
> informa de él con los pasos para reproducirlo.

Remote Play Together para juegos de Steam que no lo incluyen.

Better RPT usa SpaceWar (App ID 480) como donante fijo para abrir un juego
seleccionado dentro de un contexto que Steam reconoce como Remote Play Together.
La integración se instala una sola vez como un dispatcher persistente en la
opción de lanzamiento de SpaceWar, con copia de seguridad, verificación y
restauración. Better RPT no modifica los archivos del juego ni las opciones de
otras aplicaciones. El flujo principal es una guía paso a paso; el lanzamiento
automático es opcional y no cierra Steam.

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
4. Pulsa **Jugar**.
   - Si es la primera vez, se abre la pestaña **Guía**. Pega el comando en
     Steam → SpaceWar → Propiedades → Opciones de lanzamiento. Steam lo aplica
     sin que Better RPT cierre ni reinicie nada.
   - Si prefieres la instalación automática, ciérrala tú mismo y pulsa
     **Instalar automáticamente**. Better RPT nunca cierra Steam por ti.
   - Si el juego usa Proton, aparece un aviso de compatibilidad desconocida la
     primera vez en cada ejecución.
5. Cuando el setup aparezca como **Instalado**, pulsa **Jugar**. Better RPT
   arma la intención y pide a Steam que arranque SpaceWar con los parámetros de
   la sesión. Acepta el popup de Steam («Allow game launch?» / «¿Permitir
   iniciar el juego?»).
6. Cuando la sesión esté activa, abre el overlay de Steam con **Shift+Tab** y
   usa **Remote Play Together** para invitar a tus amigos. Better RPT no envía
   invitaciones por APIs internas.
7. Cuando termines, cierra el juego. En **Última sesión** puedes pulsar
   **Jugar de nuevo** o **Restaurar**. La biblioteca sigue visible y navegable.
   Si cierras Better RPT antes, la próxima apertura reconocerá la última sesión.
   Si el arranque falla, no se escribe ninguna opción nueva: se muestra el error
   y se puede reintentar.

El botón de restaurar solo actúa si el valor actual sigue siendo el que escribió
Better RPT. Si Steam u otro programa cambió la opción de lanzamiento, no se
sobrescribe y se muestra el conflicto con el detalle técnico. Con Steam abierto,
**Restaurar** muestra la guía para quitar el comando; no cierra Steam.

> Las imágenes de la guía deben ser capturas reales de Steam. Si alguna no está
> disponible en el entorno de compilación, se usa una ilustración claramente
> etiquetada y se registra la sustitución pendiente en
> [`docs/guide-images.md`](docs/guide-images.md).

## Qué hace exactamente

- Detecta una instalación nativa de Steam y sus bibliotecas.
- Lee `appinfo.vdf` para elegir el lanzamiento nativo o el mapping de Proton.
- Instala un dispatcher persistente en la opción de lanzamiento de SpaceWar,
  con backup, verificación y restauración. La instalación es única y reversible.
- Instala un helper estático en
  `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`.
- Pide a Steam que arranque SpaceWar con la intención de sesión; el helper lanza
  el target como hijo del contexto donante.
- Mantiene un plan, un estado, una intención y la última sesión en archivos
  privados, con bloqueo y seguimiento por PID + hora de inicio.
- Reconoce también el último juego lanzado desde Steam (`LastPlayed` de
  `localconfig.vdf`, ignorando SpaceWar) y lo muestra como última sesión, sin
  confundirlo con una restauración pendiente.
- Restaura la opción de lanzamiento de SpaceWar y ofrece recuperación tras un
  cierre inesperado o un plan expirado.
- No cierra ni reinicia Steam como parte de preparar, lanzar, cancelar o
  restaurar.

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
  (directorio `0700`, archivos `0600`, retención limitada; `Kind=session` o
  `dispatcher`).
- Setup del dispatcher: `$XDG_DATA_HOME/better-rpt/setup.json`.
- Sesiones: `$XDG_DATA_HOME/better-rpt/sessions/` con plan/estado,
  `launch-intent.json` y `last-launch.json`.
- Runtime: `$XDG_RUNTIME_DIR/better-rpt/` para el bloqueo; no es dato
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

### Steam está abierto y quiero instalar el comando

No hace falta cerrarlo: abre la pestaña **Guía**, copia el comando y pégalo en
Steam → SpaceWar → Propiedades → Opciones de lanzamiento. Steam lo aplica en
memoria. Better RPT no escribe `localconfig.vdf` mientras Steam está abierto.
Steam puede tardar en guardar el cambio hasta que cierre; si la guía sigue en
estado «Guía», pulsa **Verificar de nuevo** o cierra Steam una vez para que
Better RPT pueda leerlo. La app no lo cierra por ti.

Si prefieres la instalación automática, cierra Steam manualmente y pulsa
**Instalar automáticamente**. Better RPT no cierra Steam por ti.

Si SpaceWar ya tenía el comando del dispatcher pero no hay `setup.json`, Better
RPT lo reconoce como instalado y permite jugar; **Restaurar** queda deshabilitado
con el motivo visible porque no existe una copia del valor original.

### «Restauración pendiente», configuración huérfana o conflicto

Una restauración pendiente solo aparece si hay un marcador legacy, un backup y
un estado/token coherentes. Un `LaunchOptions` no estándar de SpaceWar no basta
para crearla.

- **Pendiente:** pulsa **Restaurar** con Steam cerrado. Con Steam abierto, la
  app muestra la guía; no cierra Steam.
- **Huérfana:** el marcador no tiene sesión/backup coherente. Se muestra aparte
  y no se presenta como la última sesión; si no hay backup, no se sobrescribe.
- **Conflicto:** el valor actual cambió fuera de Better RPT. No se sobrescribe;
  restaura manualmente el valor original o revisa el backup.

La biblioteca sigue disponible. No hay que pulsar «Volver a la biblioteca»: ya
está en pantalla. **Jugar de nuevo** repite el último lanzamiento con los mismos
parámetros.

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
make deb                         # .deb de release (Ubuntu 22.04 + nfpm)
make test                        # go test ./...
make vet                         # go vet ./...
make test-race                   # go test -race ./internal/...
```

`make build` produce un binario para desarrollo en el equipo; **no** es la
compilación final. `make deb` construye el `.deb` de release dentro de una
imagen Ubuntu 22.04 para mantener la compatibilidad con glibc 2.35 y lo empaqueta
con nfpm. Requiere Docker y nfpm v2.48.0; la receta manual está en
[`docs/build.md`](docs/build.md). El AppImage sigue la receta de
[`docs/appimage.md`](docs/appimage.md).

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

- 289 pruebas en 16 paquetes; `go vet` y `go test -race` pasan sin errores.
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
