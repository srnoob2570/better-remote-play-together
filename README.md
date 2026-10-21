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

> Estado: MVP Linux amd64. No promete compatibilidad universal. Lee la sección
> de limitaciones antes de usarlo.

## Requisitos

- Linux amd64 con glibc 2.35 o superior. El binario usa como máximo
  `GLIBC_2.34` y el paquete declara 2.35 como mínimo soportado.
- Steam nativo (no Flatpak ni Snap).
- SpaceWar (480) instalado en la misma cuenta de Steam. Es gratuito.
- Sesión gráfica X11 o Wayland, GTK 3 y WebKit2GTK 4.1
  (`libgtk-3-0`/`libgtk-3-0t64` y `libwebkit2gtk-4.1-0` en Ubuntu 22.04), además
  de las bibliotecas OpenGL del sistema.
- El AppImage no empaqueta GTK3, WebKit2GTK 4.1, Wayland, GL ni X11: el sistema
  debe tener `libgtk-3-0`, `libwebkit2gtk-4.1-0`, `libwayland-client`,
  `libwayland-cursor`, `libwayland-egl`, `libxkbcommon`, Mesa/GL y las
  bibliotecas X11 que use la sesión. Los contenedores mínimos sin estas
  bibliotecas no pueden ejecutarlo; `AppRun` lo comprueba y muestra qué falta.
- Para ejecutar el AppImage con FUSE: `fuse3`/`fusermount3`. Si no está
  disponible, se puede usar `--appimage-extract-and-run`.
- No necesitas root para usar la aplicación. Better RPT se niega a arrancar
  como root.

## Instalación

### Paquete `.deb` (recomendado)

```bash
sha256sum -c SHA256SUMS
sudo apt install ./better-rpt_0.2.2-1_amd64.deb
```

Si `apt` no acepta el archivo local:

```bash
sudo dpkg -i ./better-rpt_0.2.2-1_amd64.deb
sudo apt-get install -f
```

El paquete instala:

```text
/usr/bin/better-rpt
/usr/lib/better-rpt/better-rpt-helper
/usr/share/applications/better-rpt.desktop
/usr/share/icons/hicolor/{16x16,24x24,32x32,48x48,64x64,128x128,256x256,512x512}/apps/better-rpt.png
/usr/share/doc/better-rpt/
```

No instala servicios, no crea configuración de usuario y no necesita `sudo`
para funcionar.

### AppImage (otras distribuciones)

```bash
chmod +x BetterRPT-0.2.2-x86_64.AppImage
./BetterRPT-0.2.2-x86_64.AppImage
```

Si FUSE no está disponible:

```bash
./BetterRPT-0.2.2-x86_64.AppImage --appimage-extract-and-run
# o
APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.2.2-x86_64.AppImage
```

El AppImage y el `.deb` comparten el helper estable y la configuración. No uses
ambos a la vez. El AppImage se probó con FUSE directo y con
`--appimage-extract-and-run`; también se comprobó `--version` en Fedora 44 y
Arch con las bibliotecas gráficas del sistema, pero no se declara soporte de GUI
fuera de Ubuntu 22.04 en esta versión.

## Actualizaciones

Better RPT comprueba GitHub Releases al arrancar sin bloquear la interfaz y
muestra la versión (`v0.2.0`) junto a un botón de actualización solo con icono
en la esquina inferior izquierda. El tooltip y los iconos explican el estado; el
diálogo muestra versiones, progreso y un botón para abrir las notas del release.
Se cierra con la X superior o con **Aceptar**. La comprobación de arranque
reutiliza el último resultado correcto durante 24 horas; el botón consulta
siempre la red. No requiere cuenta, token ni configuración.

La actualización solo se ofrece para instalaciones Linux amd64:

- **AppImage:** usa la variable `APPIMAGE` para localizar el archivo en uso,
  descarga el asset `BetterRPT-<versión>-x86_64.AppImage`, verifica su SHA-256
  contra `SHA256SUMS`, le aplica `0755` y lo reemplaza con un `rename` atómico.
  Si el directorio del AppImage no es escribible, guarda el archivo verificado
  en `~/Descargas` con un nombre versionado y explica la ruta; no intenta elevar
  privilegios. Si la sustitución fue posible, **Reiniciar ahora** lanza la nueva
  versión y cierra la actual.
- **Paquete `.deb`:** comprueba que `/usr/bin/better-rpt` pertenece al paquete
  `better-rpt` con `dpkg -S`, descarga el `.deb` en un directorio temporal
  privado, verifica el SHA-256 e instala con `pkexec dpkg -i` usando argumentos
  separados. Si falta `pkexec`, no hay agente polkit o cancelas la autorización,
  la versión actual no se toca y se ofrece el enlace al release para instalarla
  a mano. Si `dpkg` falla, se muestra el error y el enlace para resolverlo
  manualmente; Better RPT no ejecuta `apt-get -f install` automáticamente. Tras
  instalar, **Reiniciar ahora** abre la versión nueva.
- **Compilación de desarrollo:** si la versión es `dev` o el binario no se
  detecta como AppImage ni como paquete Debian, no se ofrece autoinstalación;
  el diálogo indica que es una compilación de desarrollo y ofrece **Ver notas
  del release**.

La descarga tiene un límite de 200 MB y un tiempo máximo de cinco minutos; la
consulta de metadatos tiene un timeout de 10 segundos. La verificación del
checksum es obligatoria y fail-closed: si el asset o `SHA256SUMS` falta, o el
hash no coincide, se aborta sin tocar el binario actual y se eliminan los
temporales. Un fallo de red, un rate limit de GitHub o una respuesta inválida
no interrumpen la aplicación; la comprobación de arranque falla en silencio y
el botón muestra un error con reintento y el botón **Abrir release**.

El updater no se ofrece en Flatpak/Snap, otras arquitecturas ni instalaciones
manuales en rutas distintas de `/usr/bin/better-rpt`.

## Uso

1. Abre Steam e inicia sesión. Comprueba que SpaceWar aparece en tu
   biblioteca; si no, instálalo desde la ficha de la tienda (App ID 480).
2. Abre Better RPT.
3. Busca el juego y selecciónalo en la biblioteca. El panel lateral muestra su
   estado, si es nativo o Proton y los detalles técnicos.
4. Antes de jugar, marca el juego objetivo como privado en Steam: abre la
   biblioteca, haz clic derecho en el juego y elige **Administrar** y luego
   **Marcar como privado**. Así su actividad no sustituye a la de SpaceWar en
   tu perfil.
5. Pulsa **Jugar**.
   - Si es la primera vez, se abre **Configuración inicial**. Instala SpaceWar,
     comprueba el helper, guarda el valor original y pega el comando en
     Steam → SpaceWar → Propiedades → Opciones de lanzamiento. Steam lo aplica
     sin que Better RPT cierre ni reinicie nada. La guía completa sigue
     disponible en la pestaña **Guía**.
   - Si prefieres la instalación automática, ciérrala tú mismo y pulsa
     **Instalar automáticamente**. Better RPT nunca cierra Steam por ti.
   - Si el juego usa Proton, aparece un aviso de compatibilidad desconocida la
     primera vez en cada ejecución.
6. Cuando el setup aparezca como **Instalado**, pulsa **Jugar**. El botón pasa a
   **Lanzando…** y Better RPT arma la intención, pide a Steam que arranque
   SpaceWar con los parámetros de la sesión y trae Steam al frente. Acepta el
   popup de Steam («Allow game launch?» / «¿Permitir iniciar el juego?»). Si
   Steam no aparece, cambia manualmente a su ventana; el aviso sigue visible.
7. Cuando la sesión esté activa, abre el overlay de Steam con **Shift+Tab** y
   usa **Remote Play Together** para invitar a tus amigos. Better RPT no envía
   invitaciones por APIs internas.
8. Cuando termines, cierra el juego. En **Última sesión** puedes pulsar
   **Jugar de nuevo** o **Restaurar**. La biblioteca sigue visible y navegable.
   Si cierras Better RPT antes, la próxima apertura reconocerá la última sesión.
   Si el arranque falla, no se escribe ninguna opción nueva: se muestra el error
   y se puede reintentar.

El botón de restaurar solo actúa si el valor actual sigue siendo el que escribió
Better RPT. Si Steam u otro programa cambió la opción de lanzamiento, no se
sobrescribe y se muestra el conflicto con el detalle técnico. Con Steam abierto,
**Restaurar** muestra la guía para quitar el comando; no cierra Steam.

**Diagnóstico y reparación** abre un flujo aparte, primero de solo lectura, con
la acción segura para cada problema: helper, comando, Steam/SpaceWar,
dependencias del AppImage, backups o sesiones. Nunca borra setup, backups ni
última sesión sin confirmación.

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
  Si el overlay no aparece, revisa la configuración de Steam y el juego. Hay una
  regresión conocida de Steam documentada en `docs/known-issues.md`.
- **Steam Input:** al conservar la identidad del donante, Steam puede cargar el
  layout de SpaceWar para el mando del invitado. Better RPT cambia la selección
  automática (`autosave`) de los configsets por una plantilla Gamepad genérica,
  también en el control táctil, sin tocar layouts personalizados. Si Steam
  mantiene el layout del juego, desactiva Steam Input para SpaceWar. Ver
  `docs/known-issues.md`.
- **Actualizaciones de Steam:** el formato de `localconfig.vdf` o `appinfo.vdf`
  puede cambiar. Better RPT muestra un error y no escribe datos dudosos.
- **Flatpak/Snap:** se detectan, pero no se lanzan ni se modifican. Solo se
  soporta Steam nativo.
- **Plataformas:** solo Linux amd64. No Windows, macOS, Steam Deck ni Game Mode.
- **AppImage:** no cubre Alpine/musl, glibc anterior a 2.35, 32-bit ni arm64.
  En Fedora 44 y Arch solo se verificó `--version` con las bibliotecas gráficas
  instaladas; no hay pruebas de GUI de extremo a extremo ni soporte declarado
  en esas distribuciones.
- **Accesibilidad:** navegación por teclado, atajos, foco visible y contraste.
  La UI React se renderiza en WebKitGTK; la auditoría de accesibilidad está
  cubierta por Playwright y axe.
- **Una sesión a la vez:** no se pueden preparar dos donantes ni lanzar dos
  targets concurrentes.

## Datos y privacidad

Better RPT consulta GitHub Releases sin autenticación y solo descarga una
actualización cuando el usuario lo pide. No envía telemetría ni guarda tokens.
El resto de los datos queda en el equipo:

- Configuración: `$XDG_CONFIG_HOME/better-rpt/config.json`
  (por defecto `~/.config/better-rpt/config.json`).
- Última comprobación de versión: `$XDG_CONFIG_HOME/better-rpt/update.json`
  (fecha y tag del último resultado correcto, `0600`; se reutiliza 24 h).
- Helper estable: `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`
  (por defecto `~/.local/share/better-rpt/bin/better-rpt-helper`).
- Copias de `localconfig.vdf`: `$XDG_DATA_HOME/better-rpt/backups/`
  (directorio `0700`, archivos `0600`, retención limitada; `Kind=session` o
  `dispatcher`).
- Setup del dispatcher: `$XDG_DATA_HOME/better-rpt/setup.json`.
- Sesiones: `$XDG_DATA_HOME/better-rpt/sessions/` con plan/estado,
  `launch-intent.json` y `last-launch.json`.
- Log persistente: `$XDG_DATA_HOME/better-rpt/logs/better-rpt.log`, rotado a
  1 MiB con tres copias y saneado; el diagnóstico solo muestra el último error.
- Informes guardados: `$XDG_DATA_HOME/better-rpt/diagnostics/`.
- Runtime: `$XDG_RUNTIME_DIR/better-rpt/` para el bloqueo; no es dato
  persistente.

Better RPT no cambia la privacidad de tus juegos. Si el juego objetivo no está
marcado como privado en Steam, su actividad puede sustituir a la de SpaceWar en
tu perfil. El panel de sesión de la app incluye este recordatorio.

## Problemas frecuentes

### Diagnóstico automático

Si no sabes qué falla, ejecuta el diagnóstico de solo lectura antes de abrir un
issue:

```bash
better-rpt --diagnose
better-rpt --diagnose --json --out /tmp/better-rpt-diagnostic.json
```

No necesita GUI ni `DISPLAY`; funciona con la app abierta y también como root
(lo marca como `FAIL`, porque la app se niega a arrancar así). No modifica Steam,
no usa la red y no envía nada. El comando devuelve `0` si no hay `FAIL`, `1` si
hay alguno y `2` si los argumentos son incorrectos. En la app, el botón
**Diagnóstico** muestra el mismo informe y permite copiarlo o guardarlo. Si el
binario no puede cargarse por una biblioteca ausente, `AppRun` del AppImage
imprime cuáles faltan antes de arrancar. Detalles, IDs de comprobación y esquema
JSON en [`docs/diagnostics.md`](docs/diagnostics.md).

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

Ejecuta primero el diagnóstico; si FUSE no está disponible, usa la extracción:

```bash
APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.2.2-x86_64.AppImage --diagnose
```

Instala `fuse3` o ejecuta con `--appimage-extract-and-run`. En Wayland, si hay
problemas de decoración o GL, deja que GTK/Wails detecte el backend; no hace
falta fijar `GDK_BACKEND`.

### El juego arranca pero no aparece Remote Play Together

Abre el overlay con `Shift+Tab` y comprueba que Steam esté en línea. Si el juego
usa un launcher externo, anti-cheat o un modo que abandona el contexto del
donante, no hay una solución universal en este MVP.

### El overlay de Steam va lento o se congela

Cierra la ventana principal de Steam antes de jugar. No salgas de Steam. El
cliente deja de renderizar la biblioteca y la tienda, y el overlay suele
responder mejor. Es una mitigación de una regresión conocida de Steam; si
persiste, prueba la beta o abre el overlay solo para invitar y ciérralo. Esto no
corrige la pérdida de paquetes de la red local. Detalles y enlaces en
[`docs/known-issues.md`](docs/known-issues.md).

### El mando del invitado aparece conectado pero no responde

Steam Input está cargando el layout de SpaceWar (App ID 480) para el mando del
invitado. Better RPT cambia al iniciar y antes de cada lanzamiento la selección
automática (`autosave`) de los configsets por una plantilla **Gamepad**
genérica, incluido `controller_mobile_touch`; no modifica el archivo autosave ni
sobrescribe un layout personalizado. Si el editor sigue mostrando «Spacewar
Menu Controls», comprueba que la entrada `480` apunta a
`controller_<tipo>_gamepad_joystick.vdf` y, como alternativa, desactiva Steam
Input para SpaceWar en Steam > Administrar > Configuración del controlador.
Cierra el overlay antes de comprobar el mando; si está en crash-loop, reinicia
Steam. Ver [`docs/known-issues.md`](docs/known-issues.md).

## Desinstalación

Si instalaste el `.deb`:

```bash
sudo apt remove better-rpt
# o, para eliminar también los archivos de documentación del paquete:
sudo apt purge better-rpt
```

Si instalaste con `sudo make install`, `apt` y `dpkg` no conocen la app; elimina
los archivos desde este repositorio con los mismos `BINDIR`, `HELPERDIR` e
`ICONDIR` que usaste al instalar:

```bash
sudo make uninstall
```

Ninguna de las dos vías borra configuración, backups ni el helper estable del
usuario. Para eliminar todos los datos del usuario:

```bash
rm -rf ~/.config/better-rpt
rm -rf ~/.local/share/better-rpt
rm -rf "${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/better-rpt"
```

Para el AppImage, borra el archivo. Los datos anteriores siguen las mismas rutas.
**Antes de desinstalar, restaura SpaceWar** si preparaste una sesión.

## Desarrollo local (sin instalar)

No hace falta instalar el `.deb` ni el AppImage para desarrollar o probar la
app. La UI React usa Wails v3; `make dev` arranca Vite y el binario Wails con
recarga en caliente.

Primero, comprueba e instala los paquetes del sistema:

```bash
make setup-check   # informa qué falta; no necesita sudo
sudo make setup    # instala solo lo que falte; requiere privilegios
```

`make setup` detecta apt, dnf, pacman o zypper e instala únicamente los
paquetes ausentes. Cubre gcc y make, pkg-config, GTK3, WebKit2GTK 4.1,
Mesa/OpenGL, Wayland, xkbcommon y las cabeceras X11 que usa Wails. Es
idempotente: si ya está todo, no invoca al gestor de paquetes. No instala Go,
Node/npm ni el CLI de Wails; esas dependencias de lenguaje se instalan aparte:

```bash
# una vez, con Go y Node/npm instalados; añade $GOPATH/bin al PATH
go install -tags gtk3 github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.33
cd frontend && npm ci && cd ..

make dev
```

Los targets de compilación usan `GOTOOLCHAIN=go1.27.3` y no necesitan `sudo`;
solo `sudo make setup` eleva privilegios para instalar paquetes del sistema.
La versión en desarrollo se muestra como `dev`; la versión publicada (`0.2.2`)
se inyecta con `-ldflags` al compilar. `make dev` recarga Go y el frontend;
detén con `Ctrl+C`. Si el terminal se cierra de golpe, el siguiente `make dev`
retira los procesos huérfanos de este repositorio antes de arrancar;
`make dev-stop` fuerza esa limpieza.

Si prefieres instalar los paquetes a mano en Ubuntu/Debian:

```bash
sudo apt-get install \
  build-essential pkgconf \
  libgtk-3-dev libwebkit2gtk-4.1-dev \
  libgl1-mesa-dev libwayland-dev libxkbcommon-dev \
  libx11-dev libxext-dev libxrender-dev libxfixes-dev \
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

Comandos de desarrollo:

```bash
make setup-check                 # estado de los prerrequisitos del sistema
sudo make setup                  # instala los paquetes del sistema que falten
make dev                         # Wails dev + Vite, recarga en caliente
make dev-stop                    # libera restos huérfanos de `make dev`
make build                       # frontend + dist/better-rpt con assets embebidos
make frontend                    # solo bindings + frontend, sin binario host
make helper                      # dist/better-rpt-helper, estático
make deb                         # .deb de release (Ubuntu 22.04 + nfpm)
make test                        # go test ./...
make vet                         # go vet ./...
make test-race                   # go test -race ./internal/...
```

`make build` produce un binario para desarrollo en el equipo; **no** es la
compilación final. `make deb` genera solo bindings y frontend en el host, y
construye el `.deb` de release dentro de una imagen Ubuntu 22.04 para mantener la
compatibilidad con glibc 2.35; el AppImage reutiliza esos binarios del
contenedor. Requiere Docker y nfpm v2.48.0; la receta manual está en
[`docs/build.md`](docs/build.md). El AppImage sigue la receta de
[`docs/appimage.md`](docs/appimage.md).

### Pre-commit

El repo incluye `.pre-commit-config.yaml` con la base de normalización, escaneo
de secretos, formato (`gofmt` + `goimports`), `golangci-lint` y `go mod tidy`.
pre-commit instala las herramientas en un entorno aislado. Se activa una vez
por clon:

```bash
pipx install pre-commit
pre-commit install
pre-commit run --all-files
```

Los hooks de auto-fix modifican archivos y hacen fallar el commit: revisa el
diff, vuelve a `git add` y repite. `no-commit-to-branch` bloquea los commits
directos a `main` y `master`; usa una rama de trabajo.

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

export PATH="$(go env GOPATH)/bin:$PATH"

HOME="$base/home" \
GOCACHE="$go_cache" \
GOMODCACHE="$go_modcache" \
GOPATH="$go_path" \
XDG_CONFIG_HOME="$base/config" \
XDG_DATA_HOME="$base/data" \
XDG_RUNTIME_DIR="$base/run" \
make dev
```

La primera ejecución puede descargar Go 1.27.3, los módulos de Wails y las
dependencias npm; después reutiliza esas cachés.

Esta prueba no toca `~/.local/share/Steam` ni lanza Steam o juegos reales. El
conjunto incluye SpaceWar y targets nativo/Proton simulados.

### Probar el paquete instalado

Solo para validar la integración real del paquete:

```bash
sudo apt install ./dist/better-rpt_0.2.2-1_amd64.deb
better-rpt
# o
./dist/BetterRPT-0.2.2-x86_64.AppImage
```

## Compilación reproducible

La receta completa está en [`docs/build.md`](docs/build.md) y
[`docs/appimage.md`](docs/appimage.md). Los artefactos se construyen en un
contenedor Ubuntu 22.04 con Go 1.27.3, Wails v3.0.0-beta.33/GTK3/WebKit2GTK 4.1,
CGO, `-trimpath`, `-buildvcs=false`, `-ldflags '-s -w'`, nfpm 2.48.0 y
`SOURCE_DATE_EPOCH=1700000000`. El AppImage usa el runtime type2 fijado y
`appimagetool` con hash publicado. La instalación limpia del `.deb`, el arranque
GUI en X11 y Wayland y el AppImage con FUSE, extracción y cierre limpio se
probaron en Ubuntu 22.04. Tamaños de referencia: binario 11,5 MB, `.deb` 6,3 MB
y AppImage 6,6 MB.

Pruebas automatizadas:

```bash
GOTOOLCHAIN=go1.27.3 go test ./...
GOTOOLCHAIN=go1.27.3 go vet ./...
GOTOOLCHAIN=go1.27.3 go test -race ./internal/...
```

Las pruebas usan datos de Steam simulados y dobles de prueba; nunca lanzan
Steam ni juegos reales.

## Estado de verificación de esta versión

- 400 pruebas en 21 paquetes; `go vet` pasa sin errores. En el frontend,
  15 pruebas Vitest y 44 Playwright (paridad, 27 snapshots y accesibilidad).
- Binario y helper construidos en Ubuntu 22.04 con `-tags "gtk3 production"`;
  el `.deb` declara `libc6 (>= 2.35)`, GTK3 y WebKit2GTK 4.1.
- `.deb` instalado desde cero en Ubuntu 22.04; `better-rpt --version` y arranque
  GUI como usuario no root comprobados en X11 (432 ms hasta la ventana). Wayland
  se comprobó con Weston headless en el mismo contenedor.
- AppImage arrancado con FUSE y con `--appimage-extract-and-run`; GUI X11 y
  cierre limpio con `WM_DELETE_WINDOW` comprobados (883 ms hasta la ventana).
- `better-rpt --diagnose` probado sin `DISPLAY`/Wayland, como root (`FAIL`
  esperado), con un lock de instancia ocupado y con `--json`/`--out` (`0600`);
  el informe tiene 31 comprobaciones, no adquiere el lock ni escribe Steam.
- Medición orientativa en reposo, X11: 157.480 KB del proceso principal y
  276.340 KB sumando los procesos de WebKit. No hay límite duro de RAM.
- Una compilación de cada formato; los SHA-256 están en `dist/SHA256SUMS` cuando
  se ejecuta el release completo. La verificación de reproducibilidad del
  pipeline queda en `scripts/release.sh`.
- **No se ha completado una sesión de extremo a extremo con Steam real y
  SpaceWar + un target nativo/Proton en este entorno.** No se afirma que esa
  combinación funcione; es la verificación pendiente antes de una publicación
  pública.

## Licencia

MIT. Consulta [`LICENSE`](LICENSE).
