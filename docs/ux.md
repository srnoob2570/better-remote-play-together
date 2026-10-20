# Interfaz de Better RPT

Documento de apoyo del rediseño «guía primero». La implementación actual es
una aplicación Wails v3 con React y WebKitGTK. Define jerarquía, estados,
acciones y textos del MVP. El flujo de cerrar Steam ya no existe.

## Ventana

- Tamaño inicial 1280×800; mínimo 1024×600.
- Cabecera: **Better RPT** a la izquierda; a la derecha, botón **Diagnóstico**,
  versión y selector **ES | EN**.
- Cuerpo en dos columnas: biblioteca (≈60 %) y panel lateral (≈40 %, mínimo
  400 px).
- La biblioteca y la sesión son dos campos de color distintos: tinta para la
  biblioteca y verde profundo para la sesión.
- No hay barra de menús. **Diagnóstico** es la única pantalla secundaria y se
  abre desde la cabecera; la guía vive en una pestaña del panel lateral. Los
  errores se muestran en el panel, en diálogos o en notificaciones toast.

## Diagnóstico

- El botón **Diagnóstico** de la cabecera abre una pantalla de solo lectura con
  el mismo motor que `better-rpt --diagnose`; el botón de volver regresa a la
  biblioteca sin perder selección ni filtros.
- La cabecera muestra el veredicto y los recuentos OK/WARN/FAIL/SKIP. Los
  problemas (`FAIL` y `WARN`) aparecen primero, y las comprobaciones correctas u
  omitidas quedan en un desplegable compacto.
- Cada problema muestra estado, mensaje, evidencia, siguiente paso y enlaces a
  `docs/appimage.md`, `docs/compatibility.md` o `docs/known-issues.md`. Los
  enlaces abren el navegador del sistema, no una página dentro del WebView.
- **Actualizar** vuelve a ejecutar las comprobaciones. **Copiar informe** copia
  el Markdown y **Guardar informe** escribe un archivo privado en
  `$XDG_DATA_HOME/better-rpt/diagnostics/`.
- No adquiere el lock de instancia, no modifica Steam y no lanza juegos. El
  detalle de las 31 comprobaciones y del JSON está en
  [`docs/diagnostics.md`](diagnostics.md).

## Biblioteca

- Campo de búsqueda siempre visible: nombre o App ID.
- Filtro por estado: Todos, Listos, Desconocidos, No compatibles.
- Botón de refrescar.
- Tabla con cabecera **Juego / Tipo / Estado**. Las columnas mantienen su
  posición; el nombre se trunca si no cabe.
- Fila: icono de Steam, nombre, tipo (**Nativo** o **Proton**) y estado en
  texto. Sin puntos, insignias, chips ni pastillas.
- La portada se obtiene de la caché de arte de Steam; si no existe, se muestra
  una portada tipográfica con el nombre, no una imagen inventada.
- El App ID, la biblioteca, la ruta y el Proton resuelto viven en **Detalles**.
- Orden alfabético por nombre; sin ordenación configurable.
- Las herramientas/runtimes y SpaceWar no se listan.
- La biblioteca sigue visible y navegable durante la sesión, la guía y la
  restauración. Diagnóstico es la única pantalla secundaria; el resto del flujo
  no obliga a «volver a la biblioteca».

## Panel lateral: tres pestañas

1. **Sesión**: última sesión, sesión activa, preparación, errores o selección.
2. **Detalles**: App ID, biblioteca, tipo, ruta y Proton resuelto.
3. **Guía**: comando exacto, instalación, pasos e imágenes.

La pestaña activa se conserva al cambiar de juego. Abrir la guía no pierde la
selección ni el estado de la biblioteca.

## Estados y acciones

| Estado | Acción principal | Notas |
|---|---|---|
| Listo | Jugar (si el dispatcher está instalado) o Ver guía | Sin aviso adicional |
| Desconocido | Jugar o Ver guía | Aviso de compatibilidad no confirmada, una vez por juego/ejecución |
| No compatible | Deshabilitada | Motivo en **Detalles** |
| No instalado | Deshabilitada | «Actualizar biblioteca» |
| Error | Reintentar | Detalle técnico plegable |
| Sesión activa | Deshabilitada | «Sesión activa» en **Sesión** |

Estados del setup: `missing`, `guide`, `installed`, `restored`, `conflict`.
Estados de situación: `idle`, `prepared`, `active`, `finished`,
`restore_pending`, `restored`, `orphan`, `conflict`.

La tira de pestañas muestra el estado del setup como indicador compacto cuando
existe; la guía lo repite con una instrucción y la acción **Verificar de
nuevo**.

Una configuración huérfana o un conflicto se muestran como un aviso separado,
con una sola explicación: no se añade además un motivo de restauración que
repita la misma falta de backup. No se presentan como «última sesión» ni
seleccionan automáticamente un juego.

## Última sesión

Cuando existe `last-launch.json`, la pestaña **Sesión** muestra una tarjeta
completa del último juego si no hay otra selección o si el juego seleccionado es
ese mismo último juego:

- portada, nombre y estado del último juego;
- origen (Better RPT o Steam);
- estado: pendiente, correcto, error, timeout, cancelado o desconocido;
- fecha de inicio y duración;
- acciones:
  - **Jugar de nuevo**: repite el mismo target y los mismos parámetros con un
    token nuevo.
  - **Restaurar**: deshace el setup o la transacción legacy. Si no hay nada que
    restaurar, está deshabilitado y muestra el motivo.
  - **Ver guía**: abre la guía sin cambiar la selección.

Si el usuario selecciona otro juego, la tarjeta principal sigue siendo la del
juego seleccionado. **Última sesión** aparece debajo en una fila compacta, con
portada pequeña, nombre, estado y **Abrir última sesión**; nunca reemplaza la
selección actual. **Abrir última sesión** selecciona ese juego en la biblioteca.
No se repite **Jugar de nuevo** en la fila compacta: la acción principal sigue
siendo la del juego seleccionado.

Si el último juego ya no está instalado o cambió su lanzamiento, **Jugar de
nuevo** se deshabilita con el motivo visible. La biblioteca sigue navegable.

## Acción principal al seleccionar un juego

- **Jugar**: solo si el setup está instalado y el juego es lanzable.
- **Ver guía**: si falta el setup. La guía explica cómo instalarlo.
- **No compatible / No instalado / Sesión activa**: deshabilitado con motivo.
- Si hay una restauración pendiente de otro juego, se muestra un aviso aparte,
  sin bloquear la selección actual.

## Guía

La guía es el flujo principal y debe poder consultarse con Steam abierto:

- El estado del setup se muestra como indicador compacto en la tira de
  pestañas (etiqueta **Integración de SpaceWar** + estado), visible también
  fuera de la guía. Sustituye al aviso de compatibilidad y no usa una tarjeta
  de aviso para el estado normal «no instalada».
- Dentro de la guía, una fila de estado en la parte superior muestra el mensaje
  correspondiente y **Verificar de nuevo**, siempre disponible. Al terminar
  muestra un toast **Estado actualizado**; no hay confirmación inline.
- Si SpaceWar ya contiene el dispatcher pero no existe `setup.json`, la guía lo
  reconoce como **instalado** y explica que el lanzamiento funciona pero
  **Restaurar** no está disponible porque no hay copia del valor original.
- Comando exacto:

  ```text
  '<helper>' --donor -- %command%
  ```

- Botón **Copiar comando**: al copiar muestra un toast **Copiado** (o un toast
  de error si falla), sin cambiar el texto del botón ni mover el scroll.
- Aviso de Steam y acciones de instalación solo si el setup no está instalado
  ni en conflicto: con Steam cerrado se ofrece **Instalar automáticamente**;
  con Steam abierto está deshabilitado y se muestra un único aviso para pegar
  el comando manualmente.
- **Guardar valor original** solo se muestra si falta el setup o está
  restaurado; registra el backup del valor actual sin escribir Steam.
- Pasos:
  1. Steam → Biblioteca → SpaceWar → Propiedades.
  2. Opciones de lanzamiento.
  3. Pegar el comando y guardar.
  4. Lanzar SpaceWar desde Steam.
  5. Aceptar el popup de Steam («Allow game launch?» / «¿Permitir iniciar el
     juego?»).
  6. Abrir el overlay (Shift+Tab) e invitar a Remote Play Together.
- Imágenes inline, justo después del paso que explican: opciones de
  lanzamiento tras el paso 2, comando pegado tras el 3, popup tras el 5 e
  invitación tras el 6. Se muestran a 160 px de alto para no alargar la guía.
  Las capturas de opciones y comando son reales; popup e invitación siguen
  pendientes y se documentan en `docs/guide-images.md`.
- **Restauración manual** es un bloque plegable al final, después de los pasos,
  y solo aparece si hay backup. Contiene el valor original de `LaunchOptions` y
  cómo restaurarlo; no interrumpe el flujo de la guía.

## Sesión activa

- Estado **Sesión activa**, nombre del target y tiempo transcurrido.
- Guía:
  1. Abre el overlay de Steam (Shift+Tab por defecto).
  2. Invita a tus amigos a Remote Play Together.
  3. Cuando termines, cierra el juego.
- Sin botón para «abrir overlay»: no es fiable.
- La biblioteca sigue navegable; los lanzamientos están deshabilitados.

## Fin de sesión

- Se actualiza **Última sesión** con el resultado y la duración.
- **Restaurar** queda disponible si hay setup/backup; si no, deshabilitado con
  motivo.
- **Jugar de nuevo** y **Ver guía** siguen disponibles.
- No se cierra Steam automáticamente.
- Si la sesión terminó por error, se muestra el error y se mantiene la última
  sesión recuperable.
- Si la restauración no puede aplicarse porque el usuario cambió la opción, no
  se sobrescribe: se muestra el conflicto y la ruta del backup.

## Restauración con Steam abierto

- La app **no** cierra Steam.
- Se abre la guía con las instrucciones para quitar el comando en la UI de
  Steam.
- La acción **Restaurar** aparece deshabilitada con el motivo «Steam está
  abierto…»; si el usuario pega el valor original y cierra Steam, la app
  verificará y marcará `restored`.
- Si el usuario prefiere la restauración automática, debe cerrar Steam
  manualmente.

## Diálogos

**Compatibilidad desconocida**

> La compatibilidad de «{juego}» con Remote Play Together no está garantizada.
> Algunos juegos con anti-cheat, launcher propio u overlay no funcionan.
>
> [Continuar] [Cancelar]

Se muestra una vez por juego en cada ejecución.

**Restaurar**

> Se restaurará la opción de lanzamiento original de SpaceWar. ¿Continuar?

Solo se muestra si la restauración es posible con Steam cerrado.

**Conflicto**

> La configuración actual no coincide con la copia. No se sobrescribirá.

Con **Ver guía** y **Ver detalles**.

**Error**

Patrón: qué pasó, acción primaria, detalle técnico plegable. Ejemplos:

- «No encontramos Steam» → [Reintentar] [Ver guía]
- «SpaceWar no está instalado» → [Añadir SpaceWar] [Reintentar]
- «Falta instalar el comando» → [Ver guía]
- «El juego se cerró al iniciar» → [Ver detalles] [Jugar de nuevo] [Restaurar]

## Idioma

- Inicial: `LANGUAGE`, `LC_ALL` o `LANG`; si no hay ninguna, español.
- Selector en cabecera, cambio instantáneo, persistido en la configuración.
- Todos los textos viven en `internal/i18n/catalogs/es.json` y
  `internal/i18n/catalogs/en.json`; no hay cadenas incrustadas en la UI.
- Las claves ES/EN deben tener el mismo conjunto.

## Notificaciones

La UI usa notificaciones toast (toast notifications) de shadcn/Base UI para
confirmar acciones y mostrar errores sin bloquear la biblioteca ni el panel:

- avisos de sesión (éxito o advertencia), incluidos el comando copiado, la guía
  preparada, el estado actualizado y la instalación completada;
- error de sesión, con acción **Reintentar** cuando corresponde;
- error al cargar o actualizar el catálogo, al cambiar de idioma o al abrir
  Steam;
- éxito o error al copiar o guardar el informe de diagnóstico, al cargarlo o
  actualizarlo, o al abrir su documentación;
- error al copiar el comando de la guía.

Los toast se apilan en una región accesible y se pueden cerrar con su botón. No
sustituyen la evidencia persistente del panel: los avisos de restauración
pendiente, configuración huérfana o conflicto, el error de una sesión terminada
y el estado de error de la biblioteca siguen visibles en su contexto, con sus
acciones de reintento o restauración.

## Accesibilidad

- Navegación completa por teclado: Tab por cabecera → Diagnóstico → búsqueda →
  filtro → biblioteca → pestañas → panel; Enter selecciona/activa; Ctrl+F
  enfoca la búsqueda; F5 refresca; Esc cierra diálogos y devuelve el foco.
- En Diagnóstico, el orden de tabulación recorre volver, actualizar, copiar,
  guardar y las acciones de cada comprobación; los enlaces de documentación son
  alcanzables por teclado.
- Nombres accesibles en botones e iconos; los estados incluyen texto, no solo
  color.
- Contraste y foco visibles; los toast tienen botón de cierre accesible.
- Lector de pantalla: la UI React/WebKitGTK expone nombres y roles accesibles;
  la auditoría automatizada cubre WCAG A/AA y el recorrido por teclado. Una
  auditoría manual con lector de pantalla queda fuera del MVP.

## Textos ES/EN (extracto)

| Clave | ES | EN |
|---|---|---|
| action.play | Jugar | Play |
| action.playAgain | Jugar de nuevo | Play again |
| action.openLast | Abrir última sesión | Open last session |
| action.viewGuide | Ver guía | View guide |
| action.copyCommand | Copiar comando | Copy command |
| action.copied | Copiado | Copied |
| action.installAuto | Instalar automáticamente | Install automatically |
| action.prepareGuide | Guardar valor original | Save original value |
| action.refreshSetup | Verificar de nuevo | Check again |
| action.restore | Restaurar | Restore |
| last.title | Última sesión | Last session |
| last.started | Inicio | Started |
| last.duration | Duración | Duration |
| guide.title | Guía de lanzamiento | Launch guide |
| guide.command | Comando para SpaceWar | SpaceWar command |
| guide.prepareHint | Guarda el valor actual de Opciones de lanzamiento de SpaceWar para poder restaurarlo después. No modifica Steam. | Saves SpaceWar's current Launch Options value so you can restore it later. It does not modify Steam. |
| guide.copyError | No se pudo copiar el comando. | The command could not be copied. |
| guide.manualRestore | Restauración manual | Manual restore |
| guide.originalValue | Valor original | Original value |
| guide.restoreHint | Para restaurar sin cerrar Steam, pega el valor original en Opciones de lanzamiento. También puedes usar Restaurar cuando Steam esté cerrado. | To restore without closing Steam, paste the original value into Launch Options. You can also use Restore while Steam is closed. |
| guide.setupAdopted | SpaceWar ya usa Better RPT. No hay copia del valor original, así que Restaurar no está disponible. | SpaceWar already uses Better RPT. There is no copy of the original value, so Restore is unavailable. |
| guide.checkDone | Estado actualizado. | Status updated. |
| guide.step1 | Abre Steam → Biblioteca → SpaceWar → Propiedades. | Open Steam → Library → SpaceWar → Properties. |
| guide.step2 | Entra en Opciones de lanzamiento. | Open Launch Options. |
| guide.step3 | Pega el comando y guarda. | Paste the command and save. |
| guide.step4 | Lanza SpaceWar desde Steam. | Launch SpaceWar from Steam. |
| guide.step5 | Acepta el popup de confirmación de Steam. | Accept Steam's confirmation popup. |
| guide.step6 | Abre el overlay con Shift+Tab e invita a tus amigos. | Open the overlay with Shift+Tab and invite your friends. |
| situation.active | Sesión activa | Session active |
| situation.finished | Sesión terminada | Session finished |
| situation.restorePending | Restauración pendiente | Restore pending |
| situation.orphan | Configuración huérfana | Orphaned configuration |
| situation.conflict | Conflicto de configuración | Configuration conflict |
| restore.reason.steamRunning | Steam está abierto. Ciérralo manualmente o restaura desde la guía; Better RPT no lo cerrará. | Steam is open. Close it manually or restore from the guide; Better RPT will not close it. |
| restore.reason.notNeeded | No hay nada que restaurar. | There is nothing to restore. |
| pending.title | Configuración anterior | Previous configuration |
| pending.orphanTitle | Configuración huérfana | Orphaned configuration |

La lista completa está en `internal/i18n/catalogs/es.json` y
`internal/i18n/catalogs/en.json`; este extracto no sustituye la fuente.
