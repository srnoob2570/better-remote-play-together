# Compatibilidad y errores

Documento de apoyo del ticket 07. Define la política de estados, bloqueos, advertencias y textos ES/EN. Es la base para `README.md` y la resolución de problemas.

## Principios

- No se promete compatibilidad universal: el mecanismo no es oficial y depende de Steam.
- El estado de cada juego es independiente de los bloqueos globales (Steam cerrado, SpaceWar ausente, variante no soportada).
- Se bloquea solo lo demostrablemente no lanzable; lo desconocido se puede intentar con aviso.
- No hay lista curada de juegos incompatibles: solo hechos verificables y categorías documentadas.
- El detalle técnico (rutas, código, stderr saneado) va plegado y bajo demanda; nunca stderr crudo por defecto.

## Estados por juego

| Estado | Condición exacta | Acción | UI |
|---|---|---|---|
| **Listo** | Instalado, appinfo `InfoState=2`, `common.type=Game`, launch para Linux, ejecutable presente y ruta segura; sin Proton forzado. | Jugar o Ver guía | Chip verde |
| **Desconocido** | Target Windows/Proton con launch resuelto, ejecutable presente y Proton resuelto. La compatibilidad RPT no se puede verificar. | Jugar o Ver guía + aviso una vez por juego/ejecución | Chip ámbar |
| **No compatible** | Herramienta/runtime, app sin appinfo, `InfoState≠2`, sin configuración de lanzamiento, sin lanzamiento para el SO, plantilla `%command%`, ruta no segura, ejecutable ausente/ambiguo o Proton no resuelto. | Deshabilitada; Ver detalles | Chip gris/rojo |
| **No instalado** | El juego estaba en el catálogo y su manifiesto/directorio desapareció tras refrescar. | Actualizar biblioteca | Chip gris |
| **Error** | Fallo de lectura/detección específico de esa app (appinfo corrupta, permisos, error inesperado). | Reintentar; Ver detalles | Chip rojo |
| **Sesión activa** | Es el target de la sesión actual. | Deshabilitada; Ver sesión | Chip azul |

`No instalado` no se usa para SpaceWar: su ausencia es un bloqueo global del donante.

## Bloqueos globales

| Condición | Estado del panel | Acción |
|---|---|---|
| Steam no detectado | Error global | Reintentar; Ver guía |
| Steam Flatpak/Snap | Variante no soportada | Ver guía; no se intenta lanzar |
| SpaceWar ausente | Donante no disponible | Añadir SpaceWar; Reintentar |
| Steam cerrado | Instalación automática disponible | Instalar el dispatcher o pegar el comando; no es error |
| Steam abierto | No se escribe `localconfig.vdf` | Guía para pegar el comando, o lanzar si el dispatcher ya está instalado; Better RPT no cierra Steam |
| Setup del dispatcher ausente | Guía como flujo principal | Ver guía; el botón Jugar abre la guía |
| Setup instalado | Lanzamiento disponible | Jugar, Jugar de nuevo o Restaurar |
| Popup de Steam rechazado o no aceptado | El helper no arranca; timeout de inicio | Reintentar o Ver guía |
| `appinfo.vdf` desconocido/corrupto | Error global de catálogo | Actualizar Better RPT / Reintentar; no se lanza |
| Permisos de lectura/escritura | Error de entorno | Reintentar; Ver detalles |
| Overlay explícitamente desactivado | Advertencia | Ver guía; no bloquea |
| Helper no arranca | Error de sesión | Reintentar; Restaurar |
| Target sale en <2 s | Error de lanzamiento | Ver detalles; Jugar de nuevo; Restaurar |
| Target sale en ≥2 s | Fin normal | Restaurar si hay algo que deshacer |

## Bloquear vs intentar

**Bloquear:** herramienta/runtime, sin configuración de lanzamiento, sin lanzamiento para el SO, plantilla `%command%`, ruta no segura, ejecutable ausente/ambiguo, app ausente o corrupta, Proton no resuelto, Steam no soportado, SpaceWar ausente.

**Advertir e intentar:** todo target Proton, anti-cheat, launcher propio, overlay desactivado y cualquier juego cuyo soporte RPT no se pueda verificar. Anti-cheat y launchers no se detectan desde appinfo: se documentan como causas probables de fallo en ejecución.

## Overlay

Hecho local: `OverlayAppEnable` aparece por app en `UserLocalConfigStore > apps > <appid>`; para 1599600 vale `1`. Para 480 no está presente. No hay un flag global fiable de overlay en `config.vdf`.

Política:

- Si `OverlayAppEnable=0` está explícito para 480 o para el target, mostrar advertencia con instrucciones; no bloquear.
- Si el flag no existe, no se puede concluir que el overlay esté desactivado: mostrar checklist.
- Nunca bloquear el lanzamiento solo por overlay.
- La guía de sesión incluye: «Ajustes de Steam > En el juego > Activar el overlay» y el atajo configurado (`InGameOverlayShortcutKey`; por defecto Shift+Tab).
- Mitigación local: cerrar la ventana principal de Steam (sin salir de Steam) antes de jugar reduce la contención del cliente CEF. No sustituye la corrección de Valve ni arregla la pérdida de paquetes de la red local.

## Mando y Steam Input

Hecho local: durante una sesión de RE2 vía SpaceWar, Steam cargó el action set de
SpaceWar para el mando físico del invitado y para el control táctil, y mantuvo
el foco en App ID 480. No cargó la configuración de 883710 durante la sesión.

Política:

- No bloquear el lanzamiento por el mando: es una limitación conocida, no una
  condición verificable antes de jugar.
- Si el invitado ve el mando conectado pero el target no responde, probar una
  plantilla **Gamepad** genérica en SpaceWar y cerrar el overlay antes de
  repetir. Son las dos causas mejor respaldadas.
- No modificar automáticamente el layout del usuario ni cambiar
  `SteamGameId`/`SteamOverlayGameId` sin una prueba real de RPT; conservar la
  identidad del donante es lo que mantiene la sesión y el overlay asociados a
  SpaceWar.
- Documentar la evidencia y las pruebas pendientes en `docs/known-issues.md`.

## Target que sale inmediatamente

- Si el target termina en menos de 2 s desde que el helper lo arranca: estado de error «El juego se cerró al iniciar», posible incompatibilidad y oferta de restaurar.
- Si dura 2 s o más: fin de sesión normal.
- Limitación conocida: un launcher que se separa del proceso puede hacer que el helper vea el fin antes de que el juego real termine. Se documenta; no se promete cubrirlo.

## Textos de estados

| Clave | ES | EN |
|---|---|---|
| status.ready | Listo | Ready |
| status.unknown | Desconocido | Unknown |
| status.incompatible | No compatible | Incompatible |
| status.missing | No instalado | Not installed |
| status.error | Error | Error |
| status.active | Sesión activa | Session active |
| status.ready.detail | El juego tiene una configuración de lanzamiento válida. | The game has a valid launch configuration. |
| status.unknown.detail | Compatibilidad con Remote Play Together no verificada. Puede fallar por anti-cheat, launcher o overlay. | Remote Play Together compatibility is not verified. It may fail due to anti-cheat, a launcher, or the overlay. |
| status.incompatible.detail | Hay un problema verificable que impide lanzar el juego. | A verifiable problem prevents launching this game. |
| status.missing.detail | El juego ya no está instalado. | The game is no longer installed. |
| status.error.detail | No se pudo comprobar este juego. | This game could not be checked. |
| status.active.detail | Este juego es el target de la sesión actual. | This game is the target of the current session. |

## Textos de errores

| Código | ES | EN | Acción |
|---|---|---|---|
| steam_not_found | No encontramos Steam. | Steam was not found. | Reintentar / Ver guía |
| steam_unsupported | Esta instalación de Steam no es compatible. | This Steam installation is not supported. | Ver guía |
| spacewar_missing | SpaceWar no está instalado. | SpaceWar is not installed. | Añadir SpaceWar / Reintentar |
| setup_missing | Falta instalar el comando del dispatcher. | The dispatcher command is not installed. | Ver guía |
| setup_conflict | La configuración de SpaceWar cambió fuera de Better RPT. | SpaceWar's configuration changed outside Better RPT. | Ver guía / Ver detalles |
| dispatcher_launch_failed | Steam no inició el contexto donante. | Steam did not start the donor context. | Reintentar / Ver guía |
| restore_steam_running | Steam está abierto; la restauración automática no está disponible. | Steam is open; automatic restore is unavailable. | Cerrar Steam manualmente / Ver guía |
| config_read | No se pudo leer la configuración de Steam. | Steam configuration could not be read. | Reintentar |
| config_write | No se pudo escribir la configuración de Steam. | Steam configuration could not be written. | Reintentar / Ver detalles |
| backup_failed | No se pudo guardar el backup. | The backup could not be saved. | Reintentar |
| verify_failed | La configuración no se pudo verificar y se restauró el original. | Configuration verification failed; the original was restored. | Reintentar / Ver detalles |
| account_ambiguous | No se pudo determinar la cuenta de Steam. | The Steam account could not be determined. | Reintentar |
| appinfo_unsupported | Esta versión de Steam no es compatible con Better RPT. | This Steam version is not supported by Better RPT. | Actualizar Better RPT / Ver detalles |
| appinfo_corrupt | No se pudo leer la lista de juegos de Steam. | The Steam game list could not be read. | Reintentar / Ver detalles |
| game_no_launch | Este juego no tiene una opción de lanzamiento compatible. | This game has no compatible launch option. | Ver detalles |
| game_tool | Esto es una herramienta de Steam, no un juego. | This is a Steam tool, not a game. | Ninguna |
| game_executable_missing | No se encontró el ejecutable del juego. | The game executable was not found. | Ver detalles / Actualizar biblioteca |
| game_template | La opción de lanzamiento del juego no es compatible. | The game launch option is not supported. | Ver detalles |
| game_path_escape | La ruta del juego no es segura. | The game path is not safe. | Ver detalles |
| proton_not_found | No se encontró el Proton configurado. | The configured Proton was not found. | Ver guía / Reintentar |
| session_active | Ya hay una sesión activa. | A session is already active. | Ver sesión |
| helper_timeout | Steam no inició el contexto donante. | Steam did not start the donor context. | Reintentar / Restaurar |
| target_early_exit | El juego se cerró al iniciar. | The game exited on startup. | Ver detalles / Restaurar |
| overlay_disabled | El overlay de Steam está desactivado para este juego. | The Steam overlay is disabled for this game. | Ver guía |
| permission | No hay permisos suficientes. | Insufficient permissions. | Ver detalles |
| unknown | Ocurrió un error inesperado. | An unexpected error occurred. | Reintentar / Ver detalles |

## Detalle técnico

- El mensaje principal y la acción son siempre visibles.
- «Ver detalles» muestra código estable, ruta afectada, fase, timestamp y stderr saneado.
- «Copiar detalles» copia el bloque para soporte.
- Nunca se muestra stderr crudo en el mensaje principal ni se envía a ningún servicio.

## Limitaciones documentadas

- Mecanismo no oficial; una actualización de Steam puede romperlo.
- El popup de confirmación de Steam depende de la versión y de la configuración
  del cliente. Si `steam://run/480//...` no lo muestra o no propaga los
  argumentos, Better RPT no simula el lanzamiento: deja la opción automática
  deshabilitada con motivo y mantiene la guía como flujo principal.
- Editar `localconfig.vdf` con Steam abierto no es fiable. Better RPT no lo hace:
  la vía con Steam abierto es la UI de Steam o un dispatcher ya instalado.
- La restauración automática requiere Steam cerrado. Con Steam abierto se ofrece
  la guía; Better RPT nunca lo cierra.
- Juegos con anti-cheat de kernel, launchers que se separan del proceso o
  protección que bloquea el overlay pueden fallar. Ejemplos de categoría:
  shooters competitivos con anti-cheat, juegos con launcher propio de otra
  tienda y títulos que desactivan el overlay; no se afirma que un título
  concreto falle.
- Proton no garantiza RPT aunque el prefijo se cree correctamente.
- El target usa su propio `SteamAppId` para SteamAPI/DRM, pero conserva `SteamGameId`/`SteamOverlayGameId` del donante para que Steam lo asocie a SpaceWar; la presencia y los logros del target no están garantizados.
- Juegos nativos fuera de Steam o con lanzadores propios pueden no registrarse igual.
- Steam Flatpak/Snap no están soportados para ejecución.
- No hay lista de juegos garantizados; los ejemplos son categorías, no diagnósticos.
