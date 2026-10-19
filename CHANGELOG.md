# Changelog

Todos los cambios relevantes de este proyecto se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y
el proyecto usa [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

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
