# Interfaz de Better RPT

Documento de apoyo del ticket 04. Define jerarquía, estados, acciones y textos del MVP. El ticket 07 cierra la taxonomía final de errores.

## Ventana

- Tamaño inicial 1280×800; mínimo 1024×600.
- Cabecera: **Better RPT** a la izquierda; selector **ES | EN** a la derecha.
- Cuerpo en dos columnas: biblioteca (≈65 %) y panel lateral (≈35 %, mínimo 340 px).
- Sin barra de menús ni pantallas adicionales. La ayuda y los errores se muestran en diálogos o en el panel lateral.

## Biblioteca

- Campo de búsqueda siempre visible: nombre o App ID.
- Filtro por estado: Todos, Listos, Desconocidos, No compatibles.
- Botón de refrescar.
- Fila: nombre del juego, badge **Nativo** o **Proton**, chip de estado.
- El App ID, la biblioteca, la ruta y el Proton resuelto viven en **Detalles** del panel lateral.
- Orden alfabético por nombre; sin ordenación configurable en el MVP.
- Las herramientas/runtimes no aparecen.
- Si no hay resultados: «No hay juegos que coincidan» + «Limpiar búsqueda».
- Si la biblioteca está vacía: «No hay juegos instalados» + «Actualizar» + «Abrir Steam».

## Estados y acciones

Propuesta sujeta a la tabla definitiva del ticket 07.

| Estado | Chip | Acción principal | Notas |
|---|---|---|---|
| Listo | verde | Preparar y jugar | Sin aviso adicional |
| Desconocido | ámbar | Preparar y jugar | Aviso una vez por juego y ejecución de la app |
| No compatible | gris/rojo | Deshabilitada | Mostrar motivo y «Ver detalles» |
| No instalado | gris | Deshabilitada | «Actualizar biblioteca» |
| Error | rojo | Reintentar | Detalle técnico plegable |
| Sesión activa | azul | Deshabilitada | «Ver sesión» en el panel |

Estados del donante/sesión: Steam cerrado, SpaceWar ausente, listo, preparando, sesión activa, terminada, error, restaurando, estado obsoleto.

## Acción principal

Un único botón contextual en el panel lateral:

- **Preparar y jugar**: prepara SpaceWar y lanza el donante. El helper inicia el target automáticamente; no hay un segundo botón «Jugar».
- Durante la preparación: botón «Cancelar» antes de escribir; después, «Cancelar y restaurar».
- Durante la sesión: deshabilitado con el texto «Sesión activa».
- Al terminar: **Restaurar SpaceWar**.
- Sin selección: deshabilitado con «Selecciona un juego».
- Juego no compatible: deshabilitado con «No compatible».

## Preparación

El panel lateral muestra una lista de pasos con estado pendiente/activo/hecho/error:

1. Verificar Steam y SpaceWar.
2. Instalar o actualizar el helper estable.
3. Cerrar Steam si está abierto.
4. Guardar backup de la configuración.
5. Instalar la opción de lanzamiento.
6. Iniciar Steam y SpaceWar.
7. Esperar al helper e iniciar el juego.

Si la app cerró Steam y el usuario cancela antes de escribir, la app vuelve a iniciar Steam. Si cancela después de escribir, ofrece restaurar. No se puede cancelar una vez que el donante está arrancando: se muestra «La sesión está iniciando» y se espera.

## Sesión activa

Panel lateral:

- Chip **Sesión activa**.
- Nombre del target y tiempo transcurrido.
- Estado del helper: iniciando, activo, terminando.
- Guía:
  1. Abre el overlay de Steam (Shift+Tab por defecto).
  2. Invita a tus amigos a Remote Play Together.
  3. Cuando termines, cierra el juego.
- Sin botón para «abrir overlay»: no es fiable. Si el usuario cambió el atajo, el texto dice «abre el overlay de Steam».

La biblioteca sigue visible y navegable, pero los botones de lanzamiento están deshabilitados.

## Fin de sesión

- Título «Sesión terminada»; target y duración.
- Botón principal **Restaurar SpaceWar**.
- Botón secundario **Volver a la biblioteca**.
- No se cierra Steam automáticamente. Si hace falta para restaurar, se pide confirmación.
- Si la sesión terminó por error: mostrar el error y la misma acción de restaurar.
- Si la restauración no puede aplicarse porque el usuario cambió la opción de lanzamiento, no se sobrescribe: se muestra el conflicto y la ruta del backup.

## Diálogos

**Cerrar Steam para preparar**

> Better RPT necesita cerrar Steam para configurar SpaceWar. Se cerrará y se volverá a abrir.
>
> [Cerrar Steam y continuar] [Cancelar]

**Compatibilidad desconocida**

> La compatibilidad de «{juego}» con Remote Play Together no está garantizada. Algunos juegos con anti-cheat, launcher propio o overlay no funcionan.
>
> [Continuar] [Cancelar]

Se muestra una vez por juego en cada ejecución de la app.

**Estado obsoleto**

> SpaceWar quedó preparado para una sesión anterior. ¿Quieres restaurar su configuración?
>
> [Restaurar] [Ignorar por ahora]

**Error**

Patrón: qué pasó, acción primaria, detalle técnico plegable. Ejemplos:

- «No encontramos Steam» → [Reintentar] [Ver guía]
- «SpaceWar no está instalado» → [Añadir SpaceWar] [Reintentar]
- «No se puede escribir la configuración de Steam» → [Reintentar] [Ver detalles]
- «El juego se cerró al iniciar» → [Ver detalles] [Restaurar SpaceWar]

«Añadir SpaceWar» abre `steam://install/480`; no instala nada por su cuenta.

## Idioma

- Inicial: `LANGUAGE`, `LC_ALL` o `LANG`; si no hay ninguna, español.
- Selector en cabecera, cambio instantáneo, persistido en la configuración de la app.
- Todos los textos viven en `internal/i18n`; no hay cadenas incrustadas en la UI.

## Accesibilidad

- Navegación completa por teclado: Tab por cabecera → búsqueda → filtro → biblioteca → panel; Enter selecciona/activa; Ctrl+F enfoca la búsqueda; F5 refresca; Esc cierra diálogos.
- Nombres accesibles en botones e iconos; los chips incluyen texto, no solo color.
- Contraste y foco visibles.
- Lector de pantalla: Fyne 2.8 no lo implementa en Linux. El ticket 08 evalúa la compilación de accesibilidad. No bloquea el MVP.

## Textos ES/EN

| Clave | ES | EN |
|---|---|---|
| app.title | Better RPT | Better RPT |
| search.placeholder | Buscar juego o App ID | Search game or App ID |
| filter.all | Todos | All |
| filter.ready | Listos | Ready |
| filter.unknown | Desconocidos | Unknown |
| filter.incompatible | No compatibles | Incompatible |
| action.refresh | Actualizar | Refresh |
| action.prepare | Preparar y jugar | Prepare and play |
| action.cancel | Cancelar | Cancel |
| action.cancelRestore | Cancelar y restaurar | Cancel and restore |
| action.restore | Restaurar SpaceWar | Restore SpaceWar |
| action.back | Volver a la biblioteca | Back to library |
| action.retry | Reintentar | Retry |
| action.details | Ver detalles | View details |
| action.addSpacewar | Añadir SpaceWar | Add SpaceWar |
| action.openSteam | Abrir Steam | Open Steam |
| status.ready | Listo | Ready |
| status.unknown | Desconocido | Unknown |
| status.incompatible | No compatible | Incompatible |
| status.missing | No instalado | Not installed |
| status.error | Error | Error |
| status.active | Sesión activa | Session active |
| donor.steamClosed | Steam está cerrado | Steam is closed |
| donor.spacewarMissing | SpaceWar no está instalado | SpaceWar is not installed |
| donor.ready | Listo para preparar | Ready to prepare |
| prep.title | Preparando la sesión | Preparing session |
| prep.closeSteam | Cerrar Steam | Close Steam |
| prep.backup | Guardar backup | Save backup |
| prep.launchOption | Configurar SpaceWar | Configure SpaceWar |
| prep.startDonor | Iniciar Steam y SpaceWar | Start Steam and SpaceWar |
| prep.waitHelper | Iniciar el juego | Start the game |
| session.title | Sesión activa | Session active |
| session.elapsed | Tiempo | Elapsed |
| session.guide1 | Abre el overlay de Steam (Shift+Tab por defecto). | Open the Steam overlay (Shift+Tab by default). |
| session.guide2 | Invita a tus amigos a Remote Play Together. | Invite your friends to Remote Play Together. |
| session.guide3 | Cuando termines, cierra el juego. | When you are done, close the game. |
| ended.title | Sesión terminada | Session ended |
| ended.duration | Duración | Duration |
| stale.title | Configuración pendiente | Pending configuration |
| stale.body | SpaceWar quedó preparado para una sesión anterior. | SpaceWar was left prepared for a previous session. |
| common.continue | Continuar | Continue |
| common.cancel | Cancelar | Cancel |
| common.close | Cerrar | Close |
| common.ignore | Ignorar por ahora | Ignore for now |
| language.es | Español | Spanish |
| language.en | English | English |
