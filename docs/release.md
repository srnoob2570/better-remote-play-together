# Release

`make release` publica una versión completa desde el host. El flujo manual
anterior (bump en seis archivos, mover el changelog, compilar, recalcular hashes,
commit, tag, rama, push y subida) queda en un solo comando.

## Requisitos

- Go 1.27.2, Docker y `nfpm` v2.48.0 (los usa `make deb`).
- `gh` con una sesión para el owner del remoto. El script detecta el owner y,
  si la cuenta activa no coincide, usa `gh auth token --user <owner>`.
- `appimagetool` y `runtime-x86_64` en
  `$XDG_CACHE_HOME/better-rpt/release-tools`. El script verifica sus hashes y,
  si existen en `/tmp/opencode/appimage-poc`, los copia a la caché.
- `govulncheck` es opcional; si está instalado, el script lo ejecuta.

## Uso

```bash
make release-plan                 # plan sin cambios
make release                      # flujo completo con dos confirmaciones
make release ARGS="--version 0.1.3"   # fuerza la versión
make release ARGS="--yes"         # sin confirmaciones (CI o uso experto)
make release ARGS="--no-push"     # commit, tag y rama, sin publicar
```

`release.sh` deduce la versión de los commits desde el último tag:

- `fix`/`perf` → patch.
- `feat` → patch en `0.x`, minor desde `1.0`.
- `!` o `BREAKING CHANGE` → minor en `0.x`, major desde `1.0`.

Si no hay commits `feat`, `fix`, `perf` o `revert`, aborta.

## Qué hace

1. Mueve `[Unreleased]` a `## [versión] - fecha (hash)` y actualiza los enlaces
   de comparación.
2. Reemplaza la versión anterior en `Makefile`, `scripts/build-deb.sh`, README y
   documentación de build, distribución y AppImage.
3. Ejecuta `make test`, `make vet` y `govulncheck` (si existe).
4. Compila el `.deb` en Ubuntu 22.04 con `make deb` y el AppImage con los
   binarios resultantes. Repite ambas compilaciones y compara los SHA-256.
   `--no-verify` omite la comprobación de reproducibilidad.
5. Genera `dist/SHA256SUMS` y actualiza el hash del AppImage y los tamaños del
   `.deb` en la documentación.
6. Actualiza `.release/config.json`, crea el commit `chore(release): vX.Y.Z`,
   el tag anotado y la rama `release/vX.Y.Z`.
7. Hace push de `main`, del tag y de la rama, y crea la release de GitHub con
   `.deb`, AppImage y `SHA256SUMS` adjuntos.

`--verify-install` añade una instalación real del `.deb` en un contenedor
Ubuntu 22.04 limpio.

## Reanudar

El script es idempotente:

- Si el commit de release ya existe, reutiliza los artefactos de `dist/`
  (`--skip-build` fuerza esa vía) y continúa con tag, rama y publicación.
- No duplica un tag o una rama existentes; comprueba que apunten al commit de
  release.
- Si la release de GitHub ya existe, termina sin tocar nada.

Si una ejecución se interrumpe **antes** del commit de release, deja el árbol
sucio y el script se detiene con un mensaje: revisa los cambios y crea el commit
manualmente, o descártalos. No intenta adivinar un estado parcial. En una
reanudación, los artefactos deben seguir en `dist/`; el script verifica
`SHA256SUMS` antes de publicar.

## Límites

- No actualiza dependencias ni revisa Dependabot: eso sigue siendo un paso
  previo (`go get`, `go mod tidy`, `govulncheck`).
- No usa GitHub Actions; la compilación y la subida ocurren en el host.
- No publica en un registro de paquetes; GitHub Releases es el canal.
