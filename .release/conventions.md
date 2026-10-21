# Convenciones de release

## Por qué

- `0.1.0` es la primera release: no había tags, releases ni ramas `release/*`.
  Las entradas de `[Unreleased]` se integraron en `0.1.0` porque nunca se publicó
  una versión anterior.
- La versión se define en `Makefile` (`VERSION ?= 0.2.1`), se pasa a
  `packaging/nfpm.yaml` y `scripts/release.sh` la deduce de los commits.
- `make release` (o `scripts/release.sh`) es el flujo canónico: mueve
  `[Unreleased]`, actualiza referencias y documentación, compila y verifica los
  artefactos, crea commit, tag y rama, y publica en GitHub.
- El changelog sigue Keep a Changelog 1.1.0 en español y guarda los enlaces en el
  bloque de referencias al final.
- Los commits siguen Conventional Commits en español, según
  `.commit/conventions.md`.
- El tag usa el prefijo `v` y cada release conserva una rama `release/<tag>`.
- La release se publica en GitHub con los artefactos adjuntos. No hay CI de
  GitHub ni registro de paquetes; la compilación y la subida las hace
  `make release` en el host.

## Gotchas

- `dist/` está ignorado y contiene los artefactos; nunca se versiona.
- El AppImage se genera con los binarios que deja `make deb` en `dist/`
  (compilados en Ubuntu 22.04), no con los del host; así comparte la base glibc
  del `.deb`.
- El `.deb` se compila en Ubuntu 22.04 con `make deb VERSION=<versión>` para no
  enlazar contra una glibc más nueva. No se compila la release en el host.
- El AppImage se construye con `appimagetool` build 313 (`854e19e`) y el runtime
  type2 `fedfad5`. Sus hashes y los del artefacto están en `docs/appimage.md`;
  si cambia el runtime continuo, actualiza ambos.
- La cuenta `gh` activa `johanruizb` no tiene permiso de push en
  `srnoob2570/better-remote-play-together`. El remoto SSH `github-srnoob`
  autentica como `srnoob2570`; `scripts/release.sh` detecta el owner y usa su
  token automáticamente.
- Las herramientas del AppImage se cachean en
  `$XDG_CACHE_HOME/better-rpt/release-tools`; el script verifica sus hashes y
  las copia desde `/tmp/opencode/appimage-poc` si están disponibles.
- `scripts/release.sh` reanuda: no duplica un commit, tag o rama existentes y
  termina si la release ya está publicada. `--dry-run` muestra el plan y
  `--skip-build` reutiliza `dist/`.
- La verificación de reproducibilidad recompila solo el `.deb` con
  `scripts/build-deb.sh` y el AppImage; no repite bindings ni Vite porque el
  frontend ya está generado. El binario del host no se usa: el AppImage toma los
  binarios de Ubuntu 22.04 que deja el contenedor.
- `--repack` reconstruye una versión ya publicada y re-sube los assets con
  `gh release upload --clobber`; no toca commit, tag ni rama. Antes de subir
  compara los SHA-256 con los assets publicados y aborta si difieren. Guarda una
  copia local en `dist/repack-backup-<fecha>/` antes de recompilar.
- La primera release no lleva la línea `**Full changelog:**`; las siguientes sí
  incluyen el enlace de comparación.
- El hash del encabezado de la sección del changelog es el último commit antes
  del commit de release.
- `git tag -F` recorta las líneas que empiezan por `#`; usa
  `--cleanup=verbatim` para conservar los encabezados Markdown del changelog en
  las notas del tag y de la release.
- `gh release create --notes-from-tag` usa el cuerpo del tag anotado. El tag debe
  existir en el remoto antes de crear la release; usa `--verify-tag`.
- Antes de cada release, ejecuta `govulncheck ./...` y revisa los avisos de
  Dependabot. Actualiza las dependencias con `go get` y `go mod tidy`; los
  binarios publicados deben reconstruirse para incluir el parche.
  `scripts/release.sh` ejecuta `govulncheck` si está instalado y avisa si falta.
- Las releases de seguridad usan un parche (`0.1.x`) y una sección `Security` en
  el changelog.
- El commit de release puede no cambiar la versión si ya estaba fijada; el tag
  apunta al commit que contiene el changelog y `.release/`.
- `scripts/release.sh` deduce un patch cuando hay `feat` en 0.x. Si la versión
  ya está fijada en `Makefile`, `build/config.yml` y los docs, pásale
  `--version X.Y.Z` para no desincronizar las referencias.
- El SSH al puerto 22 de GitHub puede cerrar la conexión al publicar. El script
  hace un único `git push --atomic` de `main`, tag y rama, con
  `BatchMode=yes`, `ConnectTimeout=15` y tres reintentos. No pruebes
  `ssh.github.com:443` automáticamente: en la release 0.2.1 agotó 3 minutos. No
  cambies el remoto del repositorio. Una subida interrumpida de
  `gh release create` puede dejar un draft duplicado: bórralo por ID sin tocar
  el tag.
- El aviso alto de Dependabot sobre `braces` llega por la CLI de `shadcn`
  (`fast-glob`), una dependencia de compilación; no entra en el bundle ni en
  los artefactos publicados. Al publicar 0.2.0 no había versión parcheada.

## Preferencias

- Ejecutar `make release` desde el host; no usar GitHub Actions por ahora.
- El flujo completo pide confirmación solo dos veces: antes de crear y antes de
  publicar. `--yes` las omite para uso desatendido.
- Mantener el changelog y los mensajes de commit en español.
- Adjuntar siempre `SHA256SUMS` junto al `.deb` y el AppImage.
- No publicar en un registro de paquetes; GitHub Releases es el canal.
