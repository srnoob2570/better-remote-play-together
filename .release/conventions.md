# Convenciones de release

## Por qué

- `0.1.0` es la primera release: no había tags, releases ni ramas `release/*`.
  Las entradas de `[Unreleased]` se integraron en `0.1.0` porque nunca se publicó
  una versión anterior.
- La versión se define en `Makefile` (`VERSION ?= 0.1.0`) y se pasa a
  `packaging/nfpm.yaml`; no hay archivo `VERSION` ni herramienta de bump
  automático.
- El changelog sigue Keep a Changelog 1.1.0 en español y guarda los enlaces en el
  bloque de referencias al final.
- Los commits siguen Conventional Commits en español, según
  `.commit/conventions.md`.
- El tag usa el prefijo `v` y cada release conserva una rama `release/<tag>`.
- La release se publica en GitHub con los artefactos adjuntos. No hay CI de
  GitHub ni registro de paquetes; la compilación y la subida son manuales.

## Gotchas

- `dist/` está ignorado y contiene los artefactos; nunca se versiona.
- El `.deb` se compila en Ubuntu 22.04 con `make deb VERSION=<versión>` para no
  enlazar contra una glibc más nueva. No se compila la release en el host.
- El AppImage se construye con `appimagetool` build 313 (`854e19e`) y el runtime
  type2 `fedfad5`. Sus hashes y los del artefacto están en `docs/appimage.md`;
  si cambia el runtime continuo, actualiza ambos.
- La cuenta `gh` activa `johanruizb` no tiene permiso de push en
  `srnoob2570/better-remote-play-together`. El remoto SSH `github-srnoob`
  autentica como `srnoob2570`; para crear la release usa el token de esa cuenta
  o cambia la cuenta activa.
- La primera release no lleva la línea `**Full changelog:**`; las siguientes sí
  incluyen el enlace de comparación.
- El hash del encabezado de la sección del changelog es el último commit antes
  del commit de release.
- `git tag -F` recorta las líneas que empiezan por `#`; usa
  `--cleanup=verbatim` para conservar los encabezados Markdown del changelog en
  las notas del tag y de la release.
- `gh release create --notes-from-tag` usa el cuerpo del tag anotado. El tag debe
  existir en el remoto antes de crear la release; usa `--verify-tag`.
- El commit de release puede no cambiar la versión si ya estaba fijada; el tag
  apunta al commit que contiene el changelog y `.release/`.

## Preferencias

- Compilar y subir los artefactos manualmente; no usar GitHub Actions por ahora.
- Mantener el changelog y los mensajes de commit en español.
- Adjuntar siempre `SHA256SUMS` junto al `.deb` y el AppImage.
- No publicar en un registro de paquetes; GitHub Releases es el canal.
