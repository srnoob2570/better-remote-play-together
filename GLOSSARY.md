# Glosario

Términos del dominio tal como los usa el producto. No contiene decisiones de implementación.

## Better RPT

Aplicación de escritorio Linux que prepara una sesión de Remote Play Together para un juego que no la incluye.

## Donante (donor)

Juego cuya sesión de Steam hospeda la sesión de Remote Play Together. En Better RPT el donante es siempre **SpaceWar** (App ID 480); no es configurable.

## Juego objetivo (target)

Juego instalado que el usuario quiere jugar con Remote Play Together. Puede ser un juego Linux nativo o un juego Windows ejecutado con Proton.

## Contexto donante

Entorno de Steam en el que se ejecuta el donante: su App ID, su contenedor de Proton/runtime y el overlay asociado. El juego objetivo se lanza como hijo de este contexto.

## Sesión RPT

Periodo que empieza cuando el donante está activo y el juego objetivo ha sido lanzado dentro de su contexto, y termina cuando el juego objetivo deja de ejecutarse o el usuario cierra la sesión. Las invitaciones se envían desde el overlay de Steam, no desde Better RPT.

## Preparación

Conjunto de acciones que dejan la sesión lista para lanzar: verificar Steam y SpaceWar, instalar/actualizar el helper estable, respaldar la configuración, instalar la opción de lanzamiento del donante y arrancar el contexto donante.

## Restauración

Devolver la opción de lanzamiento de SpaceWar a su valor anterior y dejar el donante en su estado normal. Solo se restaura si el valor actual sigue siendo el que puso Better RPT.

## Opción de lanzamiento (launch option)

Cadena de configuración por juego que Steam ejecuta al arrancar ese juego. Better RPT usa una opción temporal en SpaceWar para invocar su helper.

## Helper

Binario sin interfaz gráfica `better-rpt-helper` que Steam ejecuta dentro del contexto donante. Recibe el juego objetivo y lo lanza como hijo. La app lo instala en una ruta estable del usuario para que el launch option siga funcionando aunque se mueva o actualice el AppImage.

## Biblioteca de Steam

Directorio raíz de una instalación de Steam que contiene `steamapps`. Puede haber varias bibliotecas en discos distintos.

## Variante de Steam

Forma de instalación de Steam: nativa, Flatpak o Snap. El MVP solo admite la nativa.

## Estado de compatibilidad

Valoración de si un juego puede lanzarse con RPT en las condiciones actuales: listo, desconocido, no compatible, no instalado, error o sesión activa.

## Estados del juego

- **Listo:** tiene una configuración de lanzamiento válida y ningún bloqueo conocido.
- **Desconocido:** parece lanzable, pero la compatibilidad con RPT no está verificada; se permite intentarlo con aviso.
- **No compatible:** hay una condición verificable que impide lanzarlo.
- **No instalado:** el juego desapareció de la biblioteca después de cargar el catálogo.
- **Error:** la comprobación o el lanzamiento falló por una causa no identificada.
- **Sesión activa:** es el juego objetivo de la sesión actual.

## Compatibilidad desconocida

El juego parece lanzable, pero no hay garantía de que RPT funcione (anti-cheat, launcher, overlay, Proton). Se permite intentarlo con un aviso.

## Herramienta/runtime

Componente de Steam que no es un juego jugable (Proton, Steam Linux Runtime, Steamworks Shared). No se ofrece como objetivo.

## Overlay de Steam

Capa de Steam dentro del juego desde la que se envían las invitaciones de Remote Play Together. Better RPT no implementa invitaciones propias.

## Estado obsoleto

Opción de lanzamiento de SpaceWar que sigue apuntando a Better RPT cuando ya no hay una sesión activa. La app lo detecta y ofrece restaurar.
