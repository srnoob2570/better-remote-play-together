# Glosario

Términos del dominio tal como los usa el producto. No contiene decisiones de
implementación.

## Better RPT

Aplicación de escritorio Linux que prepara una sesión de Remote Play Together
para un juego que no la incluye.

## Donante (donor)

Juego cuya sesión de Steam hospeda la sesión de Remote Play Together. En Better
RPT el donante es siempre **SpaceWar** (App ID 480); no es configurable.

## Juego objetivo (target)

Juego instalado que el usuario quiere jugar con Remote Play Together. Puede ser
un juego Linux nativo o un juego Windows ejecutado con Proton.

## Contexto donante

Entorno de Steam en el que se ejecuta el donante: su App ID, su contenedor de
Proton/runtime y el overlay asociado. El juego objetivo se lanza como hijo de
este contexto; usa su propio `SteamAppId` para SteamAPI/DRM y conserva
`SteamGameId`/`SteamOverlayGameId` del donante para que la sesión RPT y el
overlay sigan asociados a SpaceWar.

## Dispatcher

Helper persistente instalado como opción de lanzamiento de SpaceWar:

```text
'<helper>' --donor -- %command%
```

Steam lo ejecuta dentro del contexto donante. Si existe una intención de
lanzamiento válida, arranca el target; si no, ejecuta `%command%` para que
SpaceWar siga funcionando normalmente. Es la base del lanzamiento automático sin
cerrar Steam.

## Setup del dispatcher

Registro durable (`setup.json`) de que el dispatcher está instalado, junto con
el backup y el valor original de `LaunchOptions`. Es una instalación única y
reversible, no una edición por sesión. Estados: `missing`, `guide`, `installed`,
`restored`, `conflict`.

## Intención de lanzamiento

Autorización de un solo uso que Better RPT escribe antes de pedir a Steam que
arranque SpaceWar. Contiene token, target y caducidad. El dispatcher la consume
y la marca como `consumed`; una intención caducada no lanza el target.

## Sesión RPT

Periodo que empieza cuando el donante está activo y el juego objetivo ha sido
lanzado dentro de su contexto, y termina cuando el juego objetivo deja de
ejecutarse o el usuario cierra la sesión. Las invitaciones se envían desde el
overlay de Steam, no desde Better RPT.

## Última sesión

Registro durable del último juego lanzado (`last-launch.json`): App ID, nombre,
token, parámetros, origen, fechas y resultado. Es independiente de la
restauración. Permite **Jugar de nuevo** con los mismos parámetros y **Ver
guía** sin perder la selección.

## Preparación

Conjunto de acciones que dejan la sesión lista para lanzar: verificar Steam y
SpaceWar, instalar/actualizar el helper estable, comprobar el setup del
dispatcher, escribir la intención y arrancar el contexto donante. En el flujo
nuevo no incluye cerrar Steam.

## Restauración

Devolver la opción de lanzamiento de SpaceWar a su valor anterior y dejar el
donante en su estado normal. Solo se restaura si el valor actual sigue siendo el
que puso Better RPT. Con Steam abierto se muestra la guía; Better RPT no cierra
Steam.

## Opción de lanzamiento (launch option)

Cadena de configuración por juego que Steam ejecuta al arrancar ese juego.
Better RPT usa una opción persistente en SpaceWar para invocar su dispatcher.
No se reescribe en cada sesión.

## Helper

Binario sin interfaz gráfica `better-rpt-helper` que Steam ejecuta dentro del
contexto donante. En modo dispatcher lee la intención y lanza el target; en modo
legacy acepta `--target`/`--session`. La app lo instala en una ruta estable del
usuario para que la opción de lanzamiento siga funcionando aunque se mueva o
actualice el AppImage.

## Guía

Flujo principal de la interfaz: muestra el comando exacto, los pasos, el popup
de Steam y las imágenes para que el usuario instale el dispatcher o lance sin
que la app cierre Steam. Está disponible en español e inglés.

## Configuración anterior

Configuración de SpaceWar que sigue apuntando a una sesión legacy de Better RPT.
Si existe un backup y un estado/token coherentes, se ofrece como restauración
pendiente; no se presenta como la última sesión.

## Configuración huérfana

Marcador de Better RPT en SpaceWar sin sesión, backup o setup coherentes. Se
detecta y se muestra por separado; no se clasifica como última sesión y no se
restaura sin backup.

## Restauración pendiente

Estado que solo existe cuando hay evidencia durable de una sesión preparada por
Better RPT cuya restauración no se ha completado: marcador legacy, backup y
estado con token/target coherentes. La mera presencia de un `LaunchOptions` no
predeterminado no crea este estado.

## Estado de compatibilidad

Valoración de si un juego puede lanzarse con RPT en las condiciones actuales:
listo, desconocido, no compatible, no instalado, error o sesión activa.

## Estados del juego

- **Listo:** tiene una configuración de lanzamiento válida y ningún bloqueo
  conocido.
- **Desconocido:** parece lanzable, pero la compatibilidad con RPT no está
  verificada; se permite intentarlo con aviso.
- **No compatible:** hay una condición verificable que impide lanzarlo.
- **No instalado:** el juego desapareció de la biblioteca después de cargar el
  catálogo.
- **Error:** la comprobación o el lanzamiento falló por una causa no
  identificada.
- **Sesión activa:** es el juego objetivo de la sesión actual.

## Compatibilidad desconocida

El juego parece lanzable, pero no hay garantía de que RPT funcione (anti-cheat,
launcher, overlay, Proton). Se permite intentarlo con un aviso.

## Herramienta/runtime

Componente de Steam que no es un juego jugable (Proton, Steam Linux Runtime,
Steamworks Shared). No se ofrece como objetivo.

## Overlay de Steam

Capa de Steam dentro del juego desde la que se envían las invitaciones de
Remote Play Together. Better RPT no implementa invitaciones propias.

## Estado obsoleto (legacy)

Término histórico para una opción de lanzamiento de sesión única que quedó
apuntando a Better RPT. En el flujo nuevo se clasifica como restauración
pendiente, configuración huérfana o conflicto según la evidencia durable.
