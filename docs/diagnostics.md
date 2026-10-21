# Diagnóstico

Better RPT incluye un diagnóstico de **solo lectura** para explicar por qué la
aplicación no arranca o por qué una sesión no puede prepararse. El mismo motor
alimenta dos entradas:

- `better-rpt --diagnose`, que no abre la GUI ni necesita `DISPLAY` ni Steam.
- La pantalla **Diagnóstico** de la aplicación.

No modifica la configuración de Steam, no instala el dispatcher, no lanza juegos,
no usa la red y no envía telemetría. La única escritura es el informe cuando se
pide `--out` (o se guarda desde la GUI).

## CLI

```bash
better-rpt --diagnose
better-rpt --diagnose --json
better-rpt --diagnose --json --out /tmp/better-rpt-diagnostic.json
```

El informe Markdown va a stdout por defecto. `--json` cambia la salida a JSON e
incluye `schema_version: 1`; `--out` guarda una copia con permisos `0600` en la
ruta indicada. El comando termina con:

| Código | Significado |
|---|---|
| `0` | No hay ninguna comprobación en `FAIL`. |
| `1` | Al menos una comprobación en `FAIL`. |
| `2` | Uso incorrecto, argumentos inesperados o error al guardar el informe. |

`--diagnose` se ejecuta **antes** de comprobar el usuario root, adquirir el
bloqueo de instancia o crear la ventana. Por eso:

- funciona con la aplicación abierta y no compite por el lock;
- funciona como root, pero marca `system.root` como `FAIL` porque la aplicación
  se niega a arrancar como root;
- puede ejecutarse sin sesión gráfica.

### AppImage que no arranca

Si FUSE no está disponible, el propio AppImage no puede montarse. En ese caso:

```bash
APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.2.0-x86_64.AppImage --diagnose
# o
./BetterRPT-0.2.0-x86_64.AppImage --appimage-extract-and-run --diagnose
```

El informe distingue entre un fallo de FUSE, una biblioteca ELF ausente, una
sesión gráfica no detectable y un problema de Steam. Para las comprobaciones de
AppImage se usan `APPIMAGE`/`APPDIR`, el montaje real y `fusermount3`.

Si el cargador dinámico no puede ejecutar el binario (por ejemplo, falta GTK3 o
WebKit2GTK), `AppRun` imprime las bibliotecas ausentes antes de intentar
arrancar; `--diagnose` necesita que el binario pueda cargarse. En ese caso,
instala las bibliotecas que indique `AppRun` y vuelve a ejecutar el diagnóstico.

## Pantalla Diagnóstico

El botón **Diagnóstico** de la cabecera ejecuta el mismo motor. Muestra el
veredicto y los recuentos OK/WARN/FAIL/SKIP. Las comprobaciones con `WARN` o
`FAIL` aparecen primero, con el estado, el mensaje, la evidencia, el siguiente
paso y los enlaces a la documentación. El resto queda en un desplegable
compacto para no enterrar lo que requiere acción; el informe exportado sigue
incluyendo todas las comprobaciones. **Copiar informe** pone el Markdown en el
portapapeles; **Guardar informe** crea un archivo en:

```text
$XDG_DATA_HOME/better-rpt/diagnostics/better-rpt-diagnostic-<UTC>.md
```

## Comprobaciones

Los IDs son estables y no se traducen. Un `SKIP` significa que la comprobación no
aplica o que no se puede verificar sin root; no equivale a un fallo.

### Entorno y empaquetado

| ID | Qué comprueba |
|---|---|
| `system.version` | Versión de Better RPT. |
| `system.format` | `.deb`, AppImage o desarrollo. |
| `system.os` | Distribución y versión desde `/etc/os-release`. |
| `system.kernel` | Kernel. |
| `system.arch` | Arquitectura. |
| `system.glibc` | Versión de glibc y mínimo documentado (2.35). |
| `system.locale` | Locale efectivo. |
| `system.session` | Sesión X11/Wayland y `DISPLAY`/`WAYLAND_DISPLAY`. |
| `system.desktop` | Escritorio detectado. |
| `system.root` | `FAIL` si se ejecuta como root. |

### AppImage y bibliotecas

| ID | Qué comprueba |
|---|---|
| `appimage.fuse` | `fusermount3` y `/dev/fuse`; si no aplica, propone extracción. |
| `appimage.mount` | Montaje FUSE del AppImage: solo `noexec` es un fallo; `nosuid`/`nodev` son normales. |
| `appimage.mac` | SELinux/AppArmor y denegaciones legibles; `SKIP` si no se pueden leer sin root. |
| `deps.elf` | `DT_NEEDED` real del binario resuelto por el cargador; enumera las bibliotecas ausentes si el binario puede cargarse. |

### Steam

| ID | Qué comprueba |
|---|---|
| `steam.install` | Instalación detectada. |
| `steam.variant` | Nativa (soportada), Flatpak o Snap (no soportadas). |
| `steam.running` | Proceso de Steam en ejecución. |
| `steam.spacewar` | SpaceWar (480) instalado. |
| `steam.dispatcher` | Dispatcher/opción de lanzamiento instalada. |
| `steam.backups` | Backups disponibles. |
| `steam.permissions` | Lectura/escritura de `localconfig.vdf` y `config.vdf`; aviso si Steam abierto. |
| `steam.overlay` | Flag de overlay: `SKIP` si no existe, `WARN` si está desactivado. |
| `steam.controllers` | Entrada 480 en configsets de mandos físicos. |
| `steam.warnings` | Advertencias actuales del catálogo (`steam.Warning`). |

### Aplicación y sesión

| ID | Qué comprueba |
|---|---|
| `app.config` | `config.json` válido. |
| `app.xdg` | Directorios XDG creables/escribibles. |
| `app.lock` | Lock libre u ocupado; solo lee el PID, nunca lo adquiere. |
| `app.helper` | Ruta, propietario, permisos, hash y versión del helper. |
| `app.logs` | Último error del log persistente. |
| `session.state` | Sesión activa o última sesión, helper, target, salida y error saneado. |
| `limits.rpt` | Recuerda que NAT/P2P, anti-cheat y compatibilidad de un target concreto no se diagnostican. |

## Informe y privacidad

El Markdown está pensado para pegarse en un issue. El JSON sigue
[`internal/diagnostics/schema.json`](../internal/diagnostics/schema.json) y el
contrato `schema_version: 1`. Cada comprobación incluye:

```json
{
  "id": "appimage.fuse",
  "status": "warn",
  "title": "FUSE",
  "message": "No se encontró FUSE...",
  "evidence": ["fusermount3: not found", "/dev/fuse: missing"],
  "nextStep": "Instala fuse3...",
  "docs": ["docs/appimage.md"]
}
```

Antes de renderizar, el motor sustituye el home por `$HOME`, el usuario por
`$USER`, el SteamID64 por `[steamid]`, los tokens de sesión de 32 hex por
`[token]` y las IPs por `[ip]`. No incluye stderr crudo ni secretos. Los enlaces
apuntan a [`docs/appimage.md`](appimage.md),
[`docs/compatibility.md`](compatibility.md) y
[`docs/known-issues.md`](known-issues.md).

El log persistente vive en `$XDG_DATA_HOME/better-rpt/logs/better-rpt.log`, rota
a 1 MiB con tres copias y pasa por el mismo saneado. `app.logs` solo muestra el
último error relevante, no el log completo.

## Límites

El diagnóstico no comprueba NAT/P2P, anti-cheat, latencia ni si un juego concreto
es compatible con Remote Play Together. Tampoco repara nada: las acciones
propuestas son manuales. Para esos casos consulta
[`docs/compatibility.md`](compatibility.md) y
[`docs/known-issues.md`](known-issues.md).
