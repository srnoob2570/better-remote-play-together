# Problemas conocidos de Steam

Estado: 2026-10-19. Son problemas del cliente de Steam que afectan a Better RPT
pero no se originan en su código. Se documentan para no confundirlos con fallos
del lanzador.

## Overlay lento, congelado o en bucle de crash

**Síntomas.** El overlay tarda en abrir, baja los FPS, se queda en negro o el
juego se congela al abrirlo o cerrarlo. Pueden aparecer varios procesos
`gameoverlayui` y algunos quedan zombis.

**Entorno observado.** Steam estable build 1791918634, overlay build
20261013184430, Ubuntu 26.04 con GNOME Wayland y AMD RX 6600. En una sesión de
RE2: 75 reaps de `gameoverlayui`, 48 asserts
`SerializedSurface.cpp (1437): OnChromePaintBufferClientDisconnect` y 5 asserts
`Failed to load "libnvidia-ml.so.1"` (AMD), con minidumps. CPU, GPU y RAM no
estaban saturadas: el overlay usaba 3–6 % de CPU y la GPU estaba al 60 %.

**Causa.** Regresión de Valve #13682; #13740 y #13744 son duplicados. Afecta a
juegos normales, no solo a SpaceWar ni a Better RPT.

**Mitigación.**

- Cerrar la ventana principal de Steam —sin hacer «Salir de Steam»— antes de
  jugar. El cliente deja de renderizar la biblioteca y la tienda, y el overlay
  suele responder mejor.
- Probar la beta de Steam 1792192922. Corrige el caso principal para muchos
  usuarios, pero #13766 sigue abierto y afecta a algunos juegos.
- Si no se necesitan invitaciones, desactivar el overlay es el workaround
  definitivo. No sirve para Remote Play Together porque las invitaciones salen
  del overlay.
- Desactivar el contador de FPS y Game Recording reduce los asserts de NVML en
  AMD, pero no corrige el fallo principal.
- `DISABLE_VK_LAYER_VALVE_steam_overlay_1=1` solo evita el crash de Vulkan y
  también elimina el overlay.

**No es** falta de recursos ni un bug de Better RPT.

## Mando conectado pero sin respuesta en el juego objetivo

**Síntomas.** El invitado ve el mando como conectado, pero el juego objetivo no
responde. Puede ocurrir con un mando físico o con los controles táctiles.

**Evidencia local (2026-10-19, RE2 vía SpaceWar).**

- El cliente envió un **GameSir Nova Lite**; Steam lo registró como Controller 1.
  Controller 0 era `controller_mobile_touch` (el teléfono).
- Para ambos cargó configuración de **App ID 480**:
  `.../config/480/controller_mobile_touch.vdf` y
  `.../workshop/content/241100/1761888411/797614364640018072_legacy.bin`.
- No cargó configuración de **883710** durante la sesión RPT. La configuración
  de 883710 (`controller_generic_gamepad_joystick.vdf`) solo aparece en sesiones
  locales anteriores.
- El foco de Steam se quedó en `AppID 480, 480`.
- La configuración de 480 es el action set de SpaceWar (`fire_lasers`,
  `menu_select`, `menu_up`…), no un gamepad genérico.
- `streaming_log.txt` registró `k_EStreamControlSetInputTemporarilyDisabled`
  al abrir el overlay. Si el overlay se queda colgado, el input puede seguir
  deshabilitado.

**Interpretación.** Las dos hipótesis apuntan al mismo mecanismo: al conservar
`SteamGameId` y `SteamOverlayGameId` del donante, Steam Input asocia el mando
del invitado a SpaceWar, carga su preset y mantiene el foco en 480. El juego
objetivo recibe poco o ningún input. El mando no está desconectado.

**Pruebas pendientes.**

1. En SpaceWar > Administrar > Configuración del controlador, elegir una
   plantilla **Gamepad** genérica (no el action set de SpaceWar) y repetir con
   el invitado. Es la prueba más directa de la hipótesis del preset.
2. Si la anterior no basta, probar con Steam Input desactivado para SpaceWar y
   confirmar que RPT sigue enviando el mando.
3. Cerrar el overlay antes de probar el mando. Si está en crash-loop, reiniciar
   Steam.
4. Comparar con un mando local en el host para separar RPT de Steam Input.

**No aplicado.** Cambiar `SteamGameId` al App ID del target es una hipótesis de
código atractiva, pero puede desasociar la sesión RPT del donante. No se cambia
sin una prueba real de invitación y mando.
