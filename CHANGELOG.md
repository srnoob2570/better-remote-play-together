# Changelog

Todos los cambios relevantes de este proyecto se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y
el proyecto usa [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

### Changed

- El arranque y el refresco del catálogo evitan trabajo duplicado: la
  biblioteca principal se recorre una sola vez, la detección del tipo de
  instalación se calcula una vez por proceso, las portadas se resuelven sin
  globs innecesarios y la edición/restauración de VDF y la verificación de la
  release reutilizan el análisis ya hecho.
- El catálogo se carga una sola vez aunque la interfaz y la recuperación de
  sesión lo pidan a la vez, y se entrega una sola vez a React. El reloj de la
  sesión activa ya no re-renderiza toda la aplicación cada segundo y se pausa
  con la ventana oculta.
- `appinfo.vdf` se indexa al abrirlo y cada registro se decodifica solo cuando
  el catálogo lo consulta, de modo que los juegos no instalados ya no se
  parsean. En una caché real de 1,5 MB, la apertura baja de ~13,8 ms a ~0,7 ms
  y de ~18,2 MB a ~1,8 MB por refresco.

## [0.2.2] - 2026-10-21 (efe5aea)

### Added

- La aplicación comprueba GitHub Releases al arrancar sin bloquear la interfaz
  y añade un botón **Buscar actualizaciones** junto a la versión. La caché
  reutiliza el último resultado correcto durante 24 h y el botón consulta
  siempre la red. Avisa de las versiones nuevas con un indicador discreto y
  muestra versión actual, versión nueva, notas del release y progreso en ES/EN.
- Actualización interna para AppImage y `.deb`: descarga HTTPS con límite de
  200 MB, verificación SHA-256 obligatoria contra `SHA256SUMS`, reemplazo
  atómico del AppImage, instalación Debian mediante `pkexec dpkg -i` y reinicio
  asistido. Si el AppImage no es escribible, guarda el archivo verificado en
  `~/Descargas`. Las compilaciones de desarrollo no ofrecen autoinstalación.
- Los fallos de red, rate limit, checksum, asset ausente o autorización polkit
  se muestran con reintento y enlace al release sin modificar la versión actual.

**Full changelog:** [v0.2.1...v0.2.2][0.2.2]

## [0.2.1] - 2026-10-20 (5c7d8eb)

### Fixed

- Steam Input vuelve a aplicar la plantilla Gamepad genérica al control táctil:
  incluye `controller_mobile_touch`, sustituye la selección automática
  `autosave` del App ID 480 y repara el configset antes de cada lanzamiento. El
  archivo autosave y los layouts personalizados no se modifican.
- El diagnóstico ya no marca como fallo el montaje FUSE normal de un AppImage
  (`nosuid`/`nodev`); solo `noexec` indica un problema real. La reparación deja
  de ofrecer «Copiar comando» cuando la comprobación no tiene un comando seguro.
- Al lanzar, el botón muestra **Lanzando…** y el panel recuerda aceptar el aviso
  de Steam. La app pide a Steam que traiga su ventana al frente después de
  solicitar el lanzamiento, sin fallar si el compositor deniega el foco.

**Full changelog:** [v0.2.0...v0.2.1][0.2.1]

## [0.2.0] - 2026-10-20 (370a955)

### Added

- La aplicación se inicia ahora sobre Wails v3 y React, conservando la ventana
  de 1280×800, el mínimo de 1024×600, el bloqueo de instancia única y el
  registro de errores en stderr.
- La biblioteca muestra búsqueda, filtros, contador, avisos, portadas e iconos
  con carga diferida, además de los estados de carga, vacío, sin resultados y
  error en la nueva interfaz.
- El panel de sesión permite preparar, cancelar, seguir y restaurar sesiones,
  muestra los detalles y la guía de lanzamiento, y cubre la última sesión,
  avisos y diálogos en ES/EN.
- El `.deb` y el AppImage se construyen sobre Wails v3 con GTK3/WebKit2GTK 4.1,
  incluyen el helper estable y comprueban las bibliotecas del sistema antes de
  arrancar.
- El diagnóstico de solo lectura comparte motor entre `better-rpt --diagnose`
  (Markdown, `--json` y `--out`) y la pantalla **Diagnóstico**. Comprueba
  entorno, AppImage/FUSE, bibliotecas ELF, Steam, helper y sesión, sanea el
  informe y no modifica Steam ni adquiere el lock de instancia. El log
  persistente rota en `$XDG_DATA_HOME/better-rpt/logs/`.
- La pantalla **Diagnóstico** prioriza los avisos y fallos, deja el resto en un
  desplegable, abre la documentación en el navegador del sistema y confirma
  copiar o guardar con notificaciones toast. La guía usa capturas reales de
  Steam para las opciones de lanzamiento y el comando pegado.
- `make uninstall` elimina los archivos que instala `sudo make install` y el
  README separa esa vía de la desinstalación del `.deb`.
- La aplicación instala o actualiza el helper estable al arrancar la GUI. Un
  wizard de configuración inicial guía por Steam, SpaceWar, helper, comando del
  dispatcher, dependencias del AppImage y verificación, sin crear un flag de
  primera vez y sin borrar setup, backups ni última sesión.
- **Diagnóstico y reparación** abre un flujo de solo lectura con el ID de cada
  problema y una acción segura para helper, dispatcher, Steam/SpaceWar,
  dependencias, backups o sesiones. Reutiliza la restauración y la guía
  existentes, no cierra Steam ni usa root.
- El comando del dispatcher no se puede copiar hasta guardar el valor original
  cuando falta el setup o está restaurado, tanto en la guía como en el wizard.

### Changed

- El estado de la integración de SpaceWar se muestra como texto compacto en la
  cabecera, en lugar de ocupar la tira de pestañas del panel lateral.

### Fixed

- La sección **Restauración manual** de la guía usa un desplegable accesible
  con los componentes e iconos del resto de la interfaz.
- El resumen de la última sesión muestra “Resultado: Terminada” en lugar de
  repetir “Terminada Terminada”, y el panel de preparación cabe completo en
  ventanas de 1024×600.
- Las capturas de la guía respetan su proporción 2.5:1 y se muestran completas a
  1024, 1280 y 1920 px; el paso 1 se recorta al menú de SpaceWar para que
  **Propiedades** se lea.

### Removed

- Se retiró la interfaz anterior y sus dependencias; la aplicación usa
  exclusivamente Wails v3, React y WebKitGTK.

**Full changelog:** [v0.1.2...v0.2.0][0.2.0]

## [0.1.2] - 2026-10-20 (57d6fc6)

### Added

- Aviso en el panel de sesión y en el README para marcar el juego objetivo como
  privado en Steam y evitar que su actividad sustituya a la de SpaceWar.

### Fixed

- La edición de `localconfig.vdf` ya no rechaza archivos con atributos
  extendidos o ACLs, como `security.selinux` en Fedora/Bazzite. La escritura
  atómica conserva esos metadatos y solo falla antes de reemplazar el original
  si no puede hacerlo.

**Full changelog:** [v0.1.1...v0.1.2][0.1.2]

## [0.1.1] - 2026-10-19 (e842cbc)

### Security

- Actualiza `golang.org/x/image` a `v0.47.0` y `golang.org/x/net` a `v0.61.0`,
  junto con `x/sys` y `x/text`, para corregir las vulnerabilidades
  CVE-2026-46599, CVE-2026-33809, CVE-2026-25680, CVE-2025-22872 y
  CVE-2025-22870 en dependencias indirectas de la interfaz anterior.

**Full changelog:** [v0.1.0...v0.1.1][0.1.1]

## [0.1.0] - 2026-10-19 (36984f1)

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
- Interfaz de escritorio con biblioteca, búsqueda por nombre o App ID, filtros por estado,
  panel de sesión, detalles técnicos y textos en español e inglés.
- Kit de marca Relay Link con logotipos, iconos y recursos para el escritorio y
  el empaquetado.
- Paquetes `.deb` y AppImage reproducibles para Linux amd64, con sumas SHA-256.
- Documentación de arquitectura, compilación, compatibilidad, distribución y
  diseño de la interfaz.
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
- La biblioteca pasa a tabla con iconos y portadas locales de Steam, oculta
  SpaceWar del listado y fija las columnas; el panel separa Sesión y Detalles y
  coloca la acción tras la información del juego.
- La última sesión se compacta a una cabecera y una fila de acciones, y la guía
  agrupa **Preparar guía** y **Verificar** en dos columnas.
- README, arquitectura, UX, compatibilidad, distribución, glosario, gates y
  changelog reconciliados con el flujo nuevo.

### Fixed

- El juego objetivo usa su propio `SteamAppId` para que SteamAPI/DRM lo
  reconozca, mientras `SteamGameId` y el overlay siguen asociados al donante;
  la sesión de Remote Play Together permanece en SpaceWar y los invitados ya no
  ven al host ocupado.
- Lanzar el juego A y después el B ya no muestra una restauración pendiente de A
  como si fuera la última sesión.
- La app adopta un dispatcher de SpaceWar ya instalado aunque falte `setup.json`,
  lo muestra como instalado y permite jugar; **Restaurar** queda deshabilitado
  con su motivo cuando no hay un valor original del que recuperar.
- La pestaña Sesión prioriza el juego seleccionado y la guía coloca cada
  ilustración junto al paso que explica. **Copiar comando** confirma con
  «Copiado/Copied» y se restablece a los dos segundos; los errores de preparar o
  instalar se muestran dentro de la guía.

### Removed

- Cierre y reapertura automática de Steam durante la preparación.
- Diálogo «Cerrar Steam para preparar» y textos asociados.

[Unreleased]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/srnoob2570/better-remote-play-together/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/srnoob2570/better-remote-play-together/releases/tag/v0.1.0
