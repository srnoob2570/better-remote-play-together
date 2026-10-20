# Arquitectura

Estado: implementación del rediseño «guía primero». El flujo anterior, que
cerraba y relanzaba Steam para cada sesión, ya no es el comportamiento por
defecto.

## Principios

- El núcleo no depende de la UI. La UI consume un servicio de aplicación.
- La lectura de Steam y la escritura de configuración están separadas: solo
  `internal/config` modifica `localconfig.vdf`.
- Better RPT nunca cierra ni reinicia Steam como efecto de preparar, lanzar,
  cancelar o restaurar.
- Con Steam abierto, la app no escribe `localconfig.vdf`: la vía fiable es que
  el usuario pegue el comando en la UI de Steam. La escritura automática solo se
  ofrece con Steam cerrado.
- El lanzamiento automático usa un **dispatcher persistente** instalado una vez
  en SpaceWar. No se reescribe la opción de lanzamiento en cada sesión.
- El estado «pendiente» se deriva de evidencia durable (sesión, token y backup
  coherentes), nunca solo de que `LaunchOptions` no sea el valor de fábrica.
- Las escrituras de configuración son transaccionales: backup, temporal en el
  mismo filesystem, `fsync`, `rename` y verificación semántica.
- El helper se ejecuta dentro del contexto donante y lanza el target como hijo;
  fija `SteamAppId` al App ID del target para que SteamAPI/DRM lo reconozca, pero
  conserva `SteamGameId`/`SteamOverlayGameId` del donante para que el overlay y
  la sesión RPT sigan asociados a SpaceWar.
- Consecuencia: Steam Input asocia el mando del invitado al App ID 480 y puede
  cargar el action set de SpaceWar; el soporte de mando del target hereda esa
  limitación. Ver `docs/known-issues.md`.
- Al refrescar el catálogo, el servicio añade de forma idempotente una referencia
  a la plantilla Gamepad genérica en los configsets de mandos físicos de
  SpaceWar cuando falta la entrada `480`. No sobrescribe layouts existentes ni
  modifica el control táctil.
- No se usan APIs internas de Steam para lanzar el target ni para invitar.

## Paquetes

```text
cmd/better-rpt/          # entrada Wails v3 (GTK3); --version, sin --donor
cmd/better-rpt-helper/   # helper sin interfaz gráfica, sin CGO, --donor
build/                   # configuración de build/dev de Wails
design/                  # tokens CSS compartidos
frontend/                # React + Vite, bindings Wails y assets del shell
internal/domain/         # Game, Status, Session, LaunchSpec, errores centinela
internal/app/            # servicio de aplicación y orquestación
internal/art/            # resolución segura de portadas/iconos de la caché de Steam
internal/brand/          # icono de la ventana
internal/steam/          # detección de raíces, bibliotecas, appinfo y ciclo de vida
internal/config/         # setup del dispatcher, backups y transacción de localconfig
internal/launch/         # planificador puro + ejecutor nativo/Proton
internal/session/        # máquina de estados, intención, última sesión y helper
internal/bridge/         # fachada Wails: DTOs, eventos, errores y contexto de sesión
internal/diagnostics/    # comprobaciones de solo lectura y render del informe
internal/logging/        # log persistente rotado y saneado
internal/redact/         # saneado de home, usuario, tokens e IPs
internal/i18n/           # textos ES/EN y locale del sistema
internal/fsutil/         # escritura atómica y permisos
packaging/               # nfpm, AppImage, .desktop, icono
```

Dependencias permitidas: `cmd/better-rpt → {bridge, diagnostics}`;
`bridge → {app, art, diagnostics}`; `app → {steam, config, launch, session}`;
`diagnostics → {config, steam, session, helper, i18n}`; `logging → {config,
fsutil, redact}`; `art → domain`; `bridge` no importa Wails; `domain` no importa
nada del proyecto. `diagnostics` solo usa APIs de lectura de esos paquetes.

## Flujo

### 1. Detección y catálogo

Se localizan Steam nativo, bibliotecas, manifiestos y cuenta activa. El catálogo
se construye con App ID, nombre, ruta, tipo (nativo/Proton) y estado. SpaceWar
(480) no aparece como target.

### 2. Instalación única del dispatcher

La opción de lanzamiento de SpaceWar se sustituye una sola vez por:

```text
'<helper absoluto>' --donor -- %command%
```

Steam ejecuta el helper dentro del contexto donante. El helper lee
`launch-intent.json`; si hay una intención armada y válida, lanza el target. Si
no la hay, ejecuta `%command%`, de modo que SpaceWar sigue funcionando como
juego normal.

Hay dos vías:

- **Guía (principal):** el usuario pega el comando en Steam → SpaceWar →
  Propiedades → Opciones de lanzamiento. Steam lo aplica en memoria y es la vía
  fiable con Steam abierto. Better RPT no escribe el archivo. Steam puede
  escribir `localconfig.vdf` al cerrarse; hasta entonces Better RPT no puede
  verificar el setup. **Verificar de nuevo** vuelve a leerlo y, si sigue en
  guía, el usuario puede cerrar Steam una vez (la app nunca lo hace).
- **Instalación automática (opcional):** solo con Steam cerrado. La app crea un
  backup `dispatcher`, escribe, verifica y registra `setup.json`. Si algo falla,
  restaura el original.

El setup queda instalado hasta que el usuario pulse **Restaurar**. Es una
decisión explícita, reversible y visible, no un efecto secundario de cada
partida.

### 3. Lanzamiento

1. La app valida el juego, el setup instalado y la ausencia de otra sesión.
2. Escribe plan, estado `preparing`, intención armada y `last-launch.json`
   (`outcome=pending`).
3. Pide a Steam que arranque SpaceWar con
   `steam://run/480//--better-rpt-session=<token>`. Ese URI muestra el popup de
   confirmación de Steam («Allow game launch?» / «¿Permitir iniciar el juego?»),
   verificado con Steam nativo en Linux. Si esa forma no estuviera disponible en
   la versión de Steam del usuario, Better RPT no simula el lanzamiento: muestra
   la guía y el error, y la opción automática queda deshabilitada.
4. El dispatcher lee la intención, la marca `consumed` y lanza el target como
   hijo del contexto donante.
5. La app sigue el estado por PID + hora de inicio. Al terminar, actualiza
   `last-launch.json` con `ok`/`error` y la duración.

No se escribe `LaunchOptions` en este paso y no se ejecuta `steam -shutdown` ni
se reinicia Steam.

### 4. Última sesión

`last-launch.json` guarda App ID, nombre, token, parámetros, tipo, origen
(`better-rpt`/`steam`), fechas y resultado. Es independiente del estado de
restauración:

- **Jugar de nuevo** repite el mismo target con los mismos parámetros y un token
  nuevo.
- **Restaurar** deshace el setup o la transacción legacy si hay algo que
  restaurar; si no, aparece deshabilitado con motivo visible.
- **Ver guía** abre la guía sin perder selección ni filtros.

Para lanzamientos hechos desde Steam, la app reconcilia con `LastPlayed` de
`localconfig.vdf` e ignora SpaceWar. Si el juego B es más reciente que una
configuración legacy de A, la UI muestra B como última sesión y A aparte como
configuración anterior/huérfana.

### 5. Restauración

- **Setup instalado y Steam cerrado:** `RestoreSetup` carga el backup
  `dispatcher`, comprueba que el valor actual sigue siendo el que escribió
  Better RPT, restaura el original y verifica el archivo completo.
- **Steam abierto:** la app abre la guía para que el usuario quite el comando en
  la UI de Steam. No cierra Steam y no escribe el archivo.
- **Sesión legacy:** se conserva la restauración por token y backup, con
  detección de conflictos.
- Si el valor actual cambió fuera de Better RPT, no se sobrescribe; se muestra
  el conflicto y la ruta del backup.

### 6. Wizards de setup y reparación

La UI deriva ambos wizards de las mismas fuentes: `setup.json` mediante
`SetupSnapshot`, `Situation` para el ciclo de sesión y el informe de
`internal/diagnostics`. No hay un flag de primera vez ni un archivo de estado
del wizard.

- **Setup inicial:** se abre al arrancar solo si la integración no está
  verificada. Los pasos se calculan de nuevo cada vez; cerrar no borra setup,
  backups ni última sesión. Reutiliza `RecordGuideSetup`, `InstallSetup`,
  `EnsureHelper` y `DispatcherCommand`.
- **Reparación:** siempre a demanda desde Diagnóstico o el error de sesión.
  Primero es de solo lectura y solo ejecuta una acción tras una elección
  explícita. Reutiliza `EnsureHelper`, `RestoreSetup`, `RestoreSession`,
  `RecoverSession` y `CancelSession`; no crea transacciones paralelas.
- Los checks conservan sus IDs estables; la UI combina `session.state` con
  `Situation` para las acciones de recuperación y restauración, sin volver a
  inferir el ciclo de vida desde cadenas de `localconfig.vdf`.
- La instalación del dispatcher y la restauración siguen exigiendo Steam
  cerrado. Ningún wizard cierra Steam, escribe `localconfig.vdf` con Steam
  abierto ni usa root.

## Máquina de estados

`session.Classify(Evidence)` es una función pura y testeable. Recibe evidencia ya
validada y devuelve un `Situation`:

```text
idle              sin setup, intención ni marcador
prepared          intención armada + plan/estado + setup
active            helper vivo + estado active
finished          estado terminal
restore_pending   marcador legacy + backup + estado/token coherentes
restored          setup restaurado y sin marcador
orphan            marcador de Better RPT sin sesión/backup coherente
conflict          setup o backup no coincide con el valor actual
```

Reglas:

- Un `LaunchOptions` no predeterminado y ajeno a Better RPT es `idle`, no
  `restore_pending`.
- Un marcador legacy sin estado es `orphan`; sin backup, además, no es
  restaurable.
- Un marcador del dispatcher sin `setup.json` es `orphan`.
- Un setup instalado con un marcador distinto es `conflict`.
- `RestoreState` explica por qué está deshabilitada la acción: Steam abierto,
  sesión activa, conflicto, sin backup o nada que restaurar.

La UI consume `Situation`, no vuelve a inferir el estado a partir de cadenas de
`localconfig.vdf`.

## Persistencia

```text
$XDG_DATA_HOME/better-rpt/
├── bin/better-rpt-helper
├── backups/localconfig-*.vdf|json      # Kind=session|dispatcher
├── diagnostics/better-rpt-diagnostic-*.md
├── logs/better-rpt.log                 # 1 MiB, 3 backups
├── sessions/
│   ├── plan-<token>.json
│   ├── state-<token>.json
│   ├── launch-intent.json
│   └── last-launch.json
└── setup.json
```

- Todos los archivos son privados (`0600`), las escrituras atómicas y los
  directorios `0700`.
- `plan`/`state` conservan la recuperación de sesiones y el bloqueo.
- `launch-intent.json` es la autorización de un solo uso que consume el
  dispatcher; caduca con el mismo TTL que el plan.
- `setup.json` registra el backup y el valor original para restaurar el
  dispatcher.
- Los backups legacy no se borran al migrar; se clasifican por `Kind`.

## Migración y recuperación

Al arrancar:

0. `Service.EnsureHelper` copia el helper empaquetado a la ruta XDG estable. Es
   idempotente, no usa root, no cierra Steam y conserva el helper de una sesión
   activa. Un fallo se registra y queda visible en el diagnóstico.
1. Se leen `setup.json`, `last-launch.json`, la intención y los estados.
2. Se inspecciona el `LaunchOptions` actual y se clasifica el marcador.
3. Un marcador legacy con backup y estado coherente se ofrece como restauración
   pendiente; sin estado pasa a huérfano.
4. Un estado activo con helper muerto se terminaliza (`helper_dead`) y se
   actualiza la última sesión.
5. Los planes caducados y las intenciones caducadas se limpian sin tocar la
   configuración de Steam.
6. Los estados antiguos no se convierten en «pendientes» solo por tener un
   `LaunchOptions` no estándar.

## Concurrencia

- `context.Context` en detección, lanzamiento, espera y restauración.
- Trabajo de disco/procesos en goroutines; actualizaciones de UI en el hilo de React.
- Una sola sesión activa; el helper toma un bloqueo en el store.
- Cancelar antes de armar la intención no escribe nada; cancelar después limpia
  la intención y deja el estado/última sesión en un estado recuperable.

## i18n y errores

- `internal/i18n/catalogs/es.json` y `en.json` son la única fuente de textos.
  Go los embebe con `go:embed`; React importa los mismos archivos. El
  `i18n.Translator` conserva la detección de locale, el fallback a español y
  `Language()`. Las claves interpoladas usan `%s`/`%d`; `T(key, args...)`
  formatea solo cuando recibe argumentos.
- Los errores del bridge llevan un `code` estable (una clave i18n, por ejemplo
  `error.refresh`), un `message` técnico y un `detail`. React resuelve `code`
  contra el catálogo compartido y cae a `error.generic` si no lo conoce. Go no
  pre-traduce los errores del bridge, para que cambiar ES/EN en caliente no deje
  mensajes antiguos. Los mensajes fatales del CLI sí los resuelve Go con el
  mismo catálogo.
- `design/tokens.css` contiene la paleta, tipografías y tamaños que consume
  React; es la fuente única de los tokens visuales.

## Shell Wails

- `cmd/better-rpt/main.go` ejecuta `diagnostics.Main` antes de parsear los flags
  normales y antes del rechazo de root, para que `--diagnose` funcione sin GUI,
  sin DISPLAY, con la app abierta y como root. Después mantiene el orden del
  MVP: flags, rechazo de root, resolución de `config.Paths`, `logging.Open`,
  `app.NewWithPaths` (lock de instancia) y creación de la ventana. Un segundo
  proceso falla con `error.instance` y no abre ventana.
- Wails v3 `v3.0.0-beta.33` se compila con `-tags gtk3`; la ventana abre en
  1280×800, mínimo 1024×600 y `InitialPosition: WindowCentered`. El icono viene
  de `internal/brand`.
- `application.Options` registra el bridge como servicio, usa
  `bridge.MarshalError`, los eventos tipados y `OnShutdown` para cancelar la
  preparación en curso.
- Producción (`-tags production`) embebe `cmd/better-rpt/dist`; desarrollo
  (`wails3 dev`) proxya Vite y lee los assets de disco. `make build` orquesta
  frontend, bindings y binario.
- `frontend/src/App.tsx` mantiene la vista principal (biblioteca + panel de
  sesión) y la vista **Diagnóstico** a pantalla completa. `Toaster` muestra las
  notificaciones de copia, guardado y error; sus textos salen del catálogo
  compartido.
- Los atajos Ctrl+F, F5 y Esc viven en React: son de la página, no atajos
  globales del escritorio.

## Bridge React

`internal/bridge` es la única superficie que se bindea desde Wails:

- `Bridge` envuelve la interfaz `App` (satisfecha por `*app.Service`) y traduce
  `domain.Game`, `session.State`, `session.Situation`, `config.Setup` y
  `config.SetupSnapshot` a DTOs camelCase. No importa Wails.
- `Snapshot` solo devuelve la caché; `RefreshCatalog` emite
  `catalog.updated`. `StartSession` y `ReplaySession` usan un contexto propio
  del bridge y emiten `session.updated` en cada fase observada y al terminar.
  `CancelSession` solo cancela en `preparing`; el shell llama a `Shutdown`
  para cancelar y esperar.
- Los errores usan `Problem{code,message,detail}`; `MarshalError` es el hook de
  Wails. `ErrorCode` mapea los centinelas de dominio/config/session/helper/
  launch/steam a códigos estables y cae a `internal`.
- `Recovery` incluye la situación, el setup y la última sesión para que la UI
  actual pueda reconstruir la guía y el panel sin llamadas adicionales.
- `RunDiagnostics` ejecuta el mismo motor que `--diagnose` con las rutas y el
  idioma del servicio; no adquiere el lock ni escribe. `SaveDiagnostics` es la
  única escritura de la UI: valida el Markdown y lo guarda como `0600` en
  `$XDG_DATA_HOME/better-rpt/diagnostics/`.
- El contrato TS manual vive en `frontend/src/bridge/types.ts` y
  `frontend/src/bridge/client.ts`; `TestTypeScriptContractMatchesDTOs` compara
  los nombres de campo con los tags JSON de Go. `frontend/src/bridge/wails.ts`
  adapta los bindings generados por Wails a esa fachada y normaliza los slices
  que Wails modela como nullable.

## Diagnóstico

- `internal/diagnostics` define 31 IDs estables, estados OK/WARN/FAIL/SKIP y un
  `Report` con `schema_version: 1`. El CLI usa el mismo `Run` que el bridge y
  renderiza Markdown o JSON; `--json` se valida antes de imprimirse.
- Las dependencias reales del binario se leen con `debug/elf` (`DT_NEEDED`) y se
  resuelven con `ldd`; no se asume una lista fija de bibliotecas.
- AppImage: se comprueban FUSE, opciones del montaje, SELinux/AppArmor y la
  alternativa `--appimage-extract-and-run`. Las denegaciones que requieren root
  se marcan `SKIP`, nunca `FAIL` inventado.
- Steam y sesión se inspeccionan con APIs de lectura (`steam.Detect`,
  `config.DetectSetup`, `config.ListBackups`, `config.InspectOverlay`,
  `config.InspectControllerLayouts`, `helper.Inspect`, store de sesiones). No se
  llama a `app.New`, `RefreshCatalog` ni `Ensure`: no hay lock, escritura de
  configsets ni lanzamiento.
- La pantalla Diagnóstico agrupa primero `WARN`/`FAIL` y deja `OK`/`SKIP` en un
  desplegable. Los enlaces de documentación se abren con `Browser.OpenURL` en el
  navegador del sistema; copiar y guardar confirman con un toast.
- Todo el texto pasa por `internal/redact` antes de entrar en el informe: home,
  usuario, SteamID64, tokens de 32 hex e IPs. El log persistente de
  `internal/logging` usa el mismo saneado, rota a 1 MiB con tres copias y
  `app.logs` solo muestra el último error relevante.

## Arte local

- `internal/art` resuelve portadas e iconos sin dependencias de UI. Solo considera
  `<root>/appcache/librarycache/<appId>` para las raíces de Steam conocidas; el
  directorio de la app no puede ser un symlink. Cada candidato se canonicaliza,
  debe quedar dentro de esa base, ser un archivo regular y no superar 8 MiB.
- El orden de cover es `library_hero`, `library_header`, `header`,
  `library_capsule`, `library_header_latam`, `library_600x900`, con un único
  subdirectorio permitido. El icono prefiere el `.jpg` de 40 hex y cae a
  `icon.jpg`/`icon.png`.
- `bridge.ArtMiddleware` monta `/api/art/{appId}/{cover|icon}` como middleware
  del asset server de Wails, antes de Vite en desarrollo. La única entrada es
  App ID + enum: no hay rutas del filesystem en la URL ni en la respuesta.
- El handler valida método (`GET`/`HEAD`), App ID y kind; responde 404 para arte
  ausente o rechazado, 400 para entradas inválidas y 405 con `Allow`. Sirve solo
  JPEG/PNG/WebP detectados por contenido, con `Cache-Control: private,
  max-age=3600`, `ETag` débil y `X-Content-Type-Options: nosniff`; `If-None-Match`
  produce 304. No copia imágenes ni las incrusta como base64.
- `frontend/src/art.ts` expone `artUrl(appId, kind)`; si la imagen falla, la UI
  conserva el fallback tipográfico.

## Pruebas

- Unitarias: VDF, appinfo, bibliotecas, construcción de comandos, setup,
  backup/restore, máquina de estados, intención, última sesión e i18n.
- Diagnóstico: tablas de comprobaciones con dependencias inyectadas, saneado,
  rotación del log, códigos de salida del CLI y golden JSON/Markdown de un
  Fedora/AppImage simulado sin DISPLAY, Steam ni FUSE.
- Integración: árboles de Steam simulados, dispatcher con intención/fallback,
  no-shutdown, conflicto de configuración y regresión A→B.
- Humo manual: instalar el `.deb`/AppImage, abrir la app, pegar/instalar el
  dispatcher, verificar el popup, lanzar, invitar y restaurar.
- La prueba real con Steam y el popup se registra en `GATES.md`; si no se puede
  verificar, la opción automática queda deshabilitada con motivo y la guía sigue
  siendo el flujo principal.

## Fuera de la arquitectura del MVP

Windows/macOS, Steam Deck/Game Mode, Flatpak/Snap, invitaciones propias y
selección de otro donante. El mecanismo no es oficial y puede romperse con
actualizaciones de Steam.
