# Convenciones de commits

## Por qué

- `CHANGELOG.md` sigue Keep a Changelog 1.1.0: las entradas nuevas van bajo
  `## [Unreleased]`, en la categoría correspondiente y en español.
- Solo los commits que cambian el comportamiento visible para el usuario
  (`feat`, `fix`, `perf`, retiradas, seguridad y reversiones) añaden una entrada.
  `docs`, `chore`, `test`, `build` y `ci` no la añaden.
- Los mensajes van en español, igual que la documentación del proyecto.
- Se usa Conventional Commits con alcance opcional (`config`, `session`, `ui`,
  `launch`, `steam`, `packaging`, etc.).

## Gotchas

- `dist/`, `.scratch/` y `references/` no se versionan; están en `.gitignore`.
- `dist/` contiene binarios y paquetes generados por `make build`, `make helper`
  y el empaquetado; nunca deben entrar al repositorio.
- `references/` contiene copias de proyectos de terceros usadas solo como
  consulta.
- `.feedback/` guarda contratos y capturas de las iteraciones visuales; está
  ignorado y no se versiona.
- Los fixtures de `testdata/steam/` son entradas de prueba versionadas; los
  scripts de `testdata/steam/` los materializan en directorios temporales.
- De `.codegraph/` solo se versiona su `.gitignore`; el índice (`codegraph.db`)
  es local de cada máquina y nunca debe entrar al repositorio.
- La versión publicada se define en `Makefile` (`VERSION`) y en
  `packaging/nfpm.yaml`; al preparar una versión nueva, mueve las entradas de
  `[Unreleased]` a la sección de la versión con su fecha y actualiza los enlaces
  de comparación del final de `CHANGELOG.md`.
- El shell Wails requiere el CLI `wails3` instalado con `-tags gtk3`; `make build`
  lo busca en `PATH` y en `$(go env GOPATH)/bin`.
- `go test ./...` compila `cmd/better-rpt/main_stub.go` por defecto; para compilar
  la entrada Wails hay que usar `-tags gtk3`. Los bindings generados en
  `frontend/bindings/` se versionan para que el frontend compile sin regenerarlos.
- Los tests de `internal/i18n` y `internal/ui/theme_tokens_test.go` comparan los
  catálogos y los tokens CSS con sus fuentes Go; si cambia la paleta o el copy,
  hay que actualizar ambos lados.
