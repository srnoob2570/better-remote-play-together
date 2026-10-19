# Convenciones de commits

## Por qué

- No hay `CHANGELOG.md`: las versiones y los cambios se documentan en `README.md`
  y en `docs/`. No crear uno salvo que el proyecto lo adopte.
- Los mensajes van en español, igual que la documentación del proyecto.
- Se usa Conventional Commits con alcance opcional (`config`, `session`, `ui`,
  `launch`, `steam`, `packaging`, etc.).

## Gotchas

- `dist/`, `.scratch/` y `references/` no se versionan; están en `.gitignore`.
- `dist/` contiene binarios y paquetes generados por `make build`, `make helper`
  y el empaquetado; nunca deben entrar al repositorio.
- `references/` contiene copias de proyectos de terceros usadas solo como
  consulta.
- Los fixtures de `testdata/steam/` son entradas de prueba versionadas; los
  scripts de `testdata/steam/` los materializan en directorios temporales.
