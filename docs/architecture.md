# Arquitectura propuesta

Estado: propuesta inicial. Los tickets 01 a 03 y 08 cierran los detalles pendientes.

## Principios

- El núcleo no depende de Fyne ni de la UI. La UI consume un servicio de aplicación.
- La lectura de Steam y la escritura de configuración están separadas: solo `internal/config` modifica `localconfig.vdf`.
- Toda operación externa usa `context.Context`; la UI nunca bloquea el hilo principal.
- Los errores son valores tipados/sentinelas, se envuelven con `%w` y se traducen a mensajes accionables en la capa UI.
- Las escrituras de configuración son transaccionales: backup, temporal en el mismo filesystem, `fsync`, `rename` y verificación.
- El helper se ejecuta dentro del contexto donante y lanza el target como hijo; no usa APIs internas de Steam.

## Paquetes

```text
cmd/better-rpt/          # entrada GUI; --version, sin --donor en producción
cmd/better-rpt-helper/   # helper sin interfaz gráfica, sin CGO, --donor
internal/domain/         # Game, Status, Session, LaunchSpec, errores centinela
internal/app/            # servicio de aplicación y orquestación
internal/steam/          # detección de raíces y bibliotecas
  ├── library/           # libraryfolders.vdf + appmanifest_*.acf
  ├── appinfo/           # parser binario v41 (ticket 02)
  └── compat/            # Proton/runtime, config.vdf (ticket 03)
internal/config/         # transacción de localconfig.vdf + backups (ticket 01)
internal/launch/         # planificador puro + ejecutor nativo/Proton
internal/helper/         # instalación estable + modo donante
internal/session/        # máquina de estados y detección de procesos (ticket 03)
internal/ui/             # interfaz Fyne
internal/i18n/           # textos ES/EN y locale del sistema
internal/fsutil/         # escritura atómica y permisos
packaging/               # nfpm, AppImage, .desktop, icono
```

Dependencias permitidas: `ui → app → {steam, config, launch, session}`; `domain` no importa nada del proyecto; `launch` no importa `ui`.

## Flujo

1. **Detección**: localizar Steam nativo, bibliotecas, manifiestos y cuenta activa.
2. **Catálogo**: construir `Game` con App ID, nombre, ruta, tipo (nativo/Proton) y estado.
3. **Preparación**: instalar/actualizar el helper estable, verificar SpaceWar, pedir confirmación si Steam está abierto, cerrar Steam, backup + escribir launch option, verificar, relanzar Steam y lanzar SpaceWar.
4. **Sesión**: el helper estable recibe target y token, construye el comando, lanza el target como hijo y espera; la app refleja el estado.
5. **Cierre**: al terminar el target, la app ofrece restaurar SpaceWar. La restauración vuelve a cerrar Steam si hace falta y deja el valor previo solo si el actual sigue siendo el nuestro.

## Fyne

Fyne v2.8 sigue siendo la opción por defecto: se compila con CGO y sin WebKitGTK, soporta Wayland y X11 y mantiene la interfaz en una sola base de código. La accesibilidad de Fyne en Linux es limitada. Si un requisito bloqueante de accesibilidad o UI no se puede cumplir, se documentará la evidencia y se propondrá Wails v2 con aprobación previa, asumiendo WebKitGTK.

## Concurrencia

- `context.Context` en detección, preparación, lanzamiento y espera.
- Trabajo de disco/procesos en goroutines; actualizaciones de UI con `fyne.Do`.
- La sesión tiene una única transición válida a la vez; no se permite un segundo lanzamiento mientras haya una sesión activa.
- Cancelar una preparación debe dejar el backup y el archivo original intactos.

## Errores y estados

Estados de dominio definidos en `docs/compatibility.md`: `ready`, `unknown`, `incompatible`, `missing`, `error`, `active`. Los errores de entorno (Steam cerrado, permisos, SpaceWar ausente, Proton ausente, escritura fallida) se modelan por separado del estado de compatibilidad; los textos ES/EN viven en `docs/ux.md` y `docs/compatibility.md`.

## Pruebas

- Unitarias: VDF de texto, `appinfo.vdf` v41, detección de bibliotecas, construcción de comandos, backup/restore, i18n.
- Integración: árboles de Steam simulados (ticket 06) con biblioteca principal/secundaria, nativo/Proton, SpaceWar presente/ausente y configuraciones ajenas.
- Empaquetado: instalador del helper estable, AppImage con runtime fijado y launch option con rutas que contienen espacios.
- Humo manual: instalar el `.deb`/AppImage, abrir la app, preparar una sesión y verificar el overlay/invitación.

## Fuera de la arquitectura del MVP

Windows/macOS, Steam Deck/Game Mode, Flatpak/Snap, invitaciones propias y selección de otro donante. El mecanismo no es oficial y puede romperse con actualizaciones de Steam.
