# Interfaz de Better RPT

Documento de apoyo del rediseño «guía primero». Define jerarquía, estados,
acciones y textos del MVP. El flujo de cerrar Steam ya no existe.

## Ventana

- Tamaño inicial 1280×800; mínimo 1024×600.
- Cabecera: **Better RPT** a la izquierda; selector **ES | EN** a la derecha.
- Cuerpo en dos columnas: biblioteca (≈60 %) y panel lateral (≈40 %, mínimo
  400 px).
- La biblioteca y la sesión son dos campos de color distintos: tinta para la
  biblioteca y verde profundo para la sesión.
- Sin barra de menús ni pantallas adicionales. La guía vive en una pestaña del
  panel lateral; los errores se muestran en el panel o en diálogos.

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
  restauración. No hay una pantalla que obligue a «volver a la biblioteca».

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

Una configuración huérfana o un conflicto se muestran como un aviso separado.
No se presentan como «última sesión» ni seleccionan automáticamente un juego.

## Última sesión

Cuando existe `last-launch.json`, la pestaña **Sesión** muestra una tarjeta
completa del último juego si no hay otra selección o si el juego seleccionado es
ese mismo último juego:

- portada, nombre y App ID del último juego;
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
juego seleccionado. **Última sesión** aparece debajo en formato compacto, con
portada pequeña, nombre, estado, **Jugar de nuevo** y **Abrir última sesión**;
nunca reemplaza la selección actual. **Abrir última sesión** selecciona ese
juego en la biblioteca.

Si el último juego ya no está instalado o cambió su lanzamiento, **Jugar de
nuevo** se deshabilita con el motivo visible. La biblioteca sigue navegable.

## Acción principal al seleccionar un juego

- **Jugar**: solo si el setup está instalado y el juego es lanzable.
- **Ver guía**: si falta el setup. La guía explica cómo instalarlo.
- **No compatible / No instalado / Sesión activa**: deshabilitado con motivo.
- Si hay una restauración pendiente de otro juego, se muestra un aviso aparte,
  sin bloquear la selección actual.

## Guía

La guía es el flujo principal y debe poder consultarse sin Steam cerrado:

- Estado del setup y motivo si hay conflicto.
- Comando exacto:

  ```text
  '<helper>' --donor -- %command%
  ```

- Botón **Copiar comando**.
- Pasos:
  1. Steam → Biblioteca → SpaceWar → Propiedades.
  2. Opciones de lanzamiento.
  3. Pegar el comando y guardar.
  4. Lanzar SpaceWar desde Steam.
  5. Aceptar el popup de Steam («Allow game launch?» / «¿Permitir iniciar el
     juego?»).
  6. Abrir el overlay (Shift+Tab) e invitar a Remote Play Together.
- Valor original de `LaunchOptions`, para que el usuario sepa qué se
  restaurará.
- Imágenes reales inline, justo después del paso que explican: opciones de
  lanzamiento tras el paso 2, comando pegado tras el 3, popup tras el 5 e
  invitación tras el 6. Si una captura no está disponible, se muestra una
  ilustración claramente etiquetada y se documenta la tarea de sustitución en
  `docs/guide-images.md`.
- Instalación automática al final de la guía: solo con Steam cerrado; con Steam
  abierto el botón está deshabilitado y explica que debe pegarse el comando o
  cerrar Steam manualmente.
- **Preparar guía**: registra el backup del valor actual sin escribir Steam, para
  poder restaurar después de que el usuario pegue el comando.

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
- Todos los textos viven en `internal/i18n`; no hay cadenas incrustadas en la UI.
- Las claves ES/EN deben tener el mismo conjunto.

## Accesibilidad

- Navegación completa por teclado: Tab por cabecera → búsqueda → filtro →
  biblioteca → pestañas → panel; Enter selecciona/activa; Ctrl+F enfoca la
  búsqueda; F5 refresca; Esc cierra diálogos.
- Nombres accesibles en botones e iconos; los estados incluyen texto, no solo
  color.
- Contraste y foco visibles.
- Lector de pantalla: Fyne 2.8 no lo implementa en Linux. No bloquea el MVP.

## Textos ES/EN (extracto)

| Clave | ES | EN |
|---|---|---|
| action.play | Jugar | Play |
| action.playAgain | Jugar de nuevo | Play again |
| action.openLast | Abrir última sesión | Open last session |
| action.viewGuide | Ver guía | View guide |
| action.copyCommand | Copiar comando | Copy command |
| action.installAuto | Instalar automáticamente | Install automatically |
| action.prepareGuide | Preparar guía | Prepare guide |
| action.restore | Restaurar | Restore |
| last.title | Última sesión | Last session |
| last.started | Inicio | Started |
| last.duration | Duración | Duration |
| last.hint | La biblioteca sigue disponible. | The library remains available. |
| guide.title | Guía de lanzamiento | Launch guide |
| guide.command | Comando para SpaceWar | SpaceWar command |
| guide.prepareHint | Preparar guía guarda el valor original antes de pegar el comando. | Prepare guide saves the original value before pasting the command. |
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

La lista completa está en `internal/i18n/i18n.go`; este extracto no sustituye la
fuente.
