# ADR 0001: guía primero y dispatcher persistente

- Estado: aceptado
- Fecha: 2026-10-19
- Contexto: lanzamiento de juegos y recuperación de sesiones

## Contexto

El flujo anterior cerraba Steam, editaba `localconfig.vdf`, lo reabría y lanzaba
SpaceWar. Era disruptivo y, además, la app infería una restauración pendiente a
partir de que `LaunchOptions` no tuviera el valor de fábrica. Lanzar otro juego
podía dejar el estado anterior como si siguiera pendiente.

La API pública de Steam no permite lanzar un target arbitrario como hijo del
contexto de SpaceWar. `steam -applaunch <target>` arranca el target normalmente,
y `steam://run/480//<args>` solo pasa argumentos a SpaceWar. Editar
`localconfig.vdf` con Steam abierto no es fiable porque Steam puede
sobrescribirlo.

## Decisión

1. **La guía es el flujo principal.** La app muestra el comando exacto y los
   pasos para pegarlo en la UI de Steam, que sí aplica cambios con Steam
   abierto. Better RPT no escribe el archivo en ese caso.
2. **El lanzamiento automático usa un dispatcher persistente**, instalado una
   sola vez como `'<helper>' --donor -- %command%`. El helper lee una intención
   durable y lanza el target; sin intención ejecuta `%command%`.
3. **No se cierra ni reinicia Steam** en preparar, lanzar, cancelar o restaurar.
   La restauración automática solo se ofrece con Steam cerrado; con Steam abierto
   se muestra la guía.
4. **«Pendiente» exige evidencia durable**: marcador legacy + backup + estado con
   token/target coherentes. Un `LaunchOptions` no estándar ajeno a Better RPT es
   inactivo. Un marcador sin sesión/backup se clasifica como huérfano.
5. **La última sesión es independiente de la restauración.** Se persiste App ID,
   token, parámetros, fechas y resultado; permite Jugar de nuevo y Restaurar con
   motivos visibles si no hay nada que deshacer.

## Consecuencias

- Una instalación nueva requiere un paso de setup explícito y reversible.
- El setup queda instalado hasta que el usuario restaure; no se reescribe por
  sesión.
- Con Steam abierto, la primera instalación es manual por diseño.
- El popup de Steam es parte del contrato de lanzamiento; si una versión de
  Steam no lo soporta, Better RPT deshabilita la opción automática con motivo y
  no simula el lanzamiento.
- La app conserva el backup y la restauración legacy; no migra ni borra estados
  válidos.
- Los tests de no-shutdown, clasificación, última sesión y regresión A→B son
  obligatorios.

## Alternativas rechazadas

- **Cerrar Steam por sesión:** disruptivo y contrario al requisito.
- **Escribir `localconfig.vdf` con Steam abierto:** riesgo de pérdida de cambios.
- **Lanzar el target con `steam -applaunch <target>`:** no preserva el contexto
  donante.
- **Inferir «pendiente» del contenido de `LaunchOptions`:** produce falsos
  positivos al lanzar otro juego.
- **Simular el popup o el paso de argumentos:** oculta una limitación real.
