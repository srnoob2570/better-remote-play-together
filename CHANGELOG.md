# Changelog

Todos los cambios relevantes de este proyecto se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y
el proyecto usa [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Guía paso a paso en español e inglés con el comando exacto, el popup de Steam,
  imágenes (capturas reales o placeholders documentados) y copia al portapapeles.
- Dispatcher persistente de SpaceWar (`'<helper>' --donor -- %command%`) que
  permite el lanzamiento automático sin cerrar ni reiniciar Steam; sin intención
  ejecuta `%command%` para que SpaceWar siga funcionando.
- Instalación automática del dispatcher solo con Steam cerrado, con backup,
  verificación, rollback y restauración.
- Registro durable de la última app lanzada (`last-launch.json`) con App ID,
  token, parámetros, fechas y resultado; acciones **Jugar de nuevo**,
  **Restaurar** y **Ver guía**.
- Máquina de estados explícita y testeable: inactivo, preparado, activo,
  terminado, restauración pendiente, restaurado, configuración huérfana y
  conflicto.
- Detección separada de configuraciones huérfanas y conflictos, sin presentarlas
  como la última sesión.
- Migración/reconciliación de estados legacy sin falsos positivos.
- ADR `docs/adr/0001-guide-first-launch.md` y `docs/guide-images.md`.
- `docs/known-issues.md` con la regresión del overlay de Steam y la limitación
  del mando del invitado por el layout de Steam Input del donante.
- Aplicación idempotente de la plantilla Gamepad genérica de SpaceWar al
  refrescar el catálogo: solo añade la entrada `480` cuando falta, no sobrescribe
  layouts existentes y no modifica el control táctil.

### Changed

- El flujo por defecto ya no cierra Steam: la guía es el camino principal y el
  lanzamiento directo es opcional.
- «Pendiente» ya no se infiere de un `LaunchOptions` no predeterminado; exige
  marcador legacy, backup y estado/token coherentes.
- La biblioteca permanece visible y navegable; se elimina la acción **Volver a
  la biblioteca** y el diálogo de cierre de Steam.
- La restauración con Steam abierto muestra la guía y nunca cierra Steam.
- README, arquitectura, UX, compatibilidad, distribución, glosario, gates y
  changelog reconciliados con el flujo nuevo.

### Fixed

- El juego objetivo usa su propio `SteamAppId` para que SteamAPI/DRM lo
  reconozca, mientras `SteamGameId` y el overlay siguen asociados al donante;
  la sesión de Remote Play Together permanece en SpaceWar y los invitados ya no
  ven al host ocupado.
- Lanzar el juego A y después el B ya no muestra una restauración pendiente de A
  como si fuera la última sesión.

### Removed

- Cierre y reapertura automática de Steam durante la preparación.
- Diálogo «Cerrar Steam para preparar» y textos asociados.

## [0.1.0] - 2026-10-19

### Added

- Aplicación de escritorio para Linux amd64 que abre juegos de Steam sin Remote
  Play Together dentro del contexto de SpaceWar (App ID 480) como donante fijo.
- Detección de Steam nativo, sus bibliotecas y `appinfo.vdf`, con elección del
  lanzamiento nativo o del mapping de Proton.
- Preparación y restauración de la opción de lanzamiento de SpaceWar en
  `localconfig.vdf`, con copia de seguridad, verificación, bloqueo y recuperación
  tras un cierre inesperado o un plan expirado.
- Helper estático instalado en `$XDG_DATA_HOME/better-rpt/bin` que inicia el
  juego objetivo como hijo del contexto del donante.
- Interfaz Fyne con biblioteca, búsqueda por nombre o App ID, filtros por estado,
  panel de sesión, detalles técnicos y textos en español e inglés.
- Kit de marca Relay Link con logotipos, iconos y recursos para el escritorio y
  el empaquetado.
- Paquetes `.deb` y AppImage reproducibles para Linux amd64, con sumas SHA-256.
- Documentación de arquitectura, compilación, compatibilidad, distribución y
  diseño de la interfaz.

[Unreleased]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/srnoob2570/better-remote-play-together/releases/tag/v0.1.0
