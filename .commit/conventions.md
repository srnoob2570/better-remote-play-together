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
- Las capturas de `frontend/public/guide/` son 1200×480 (2.5:1) y se muestran a
  ancho completo con `aspect-ratio: 5 / 2`; no volver a fijar una altura.
- Los fixtures de `testdata/steam/` son entradas de prueba versionadas; los
  scripts de `testdata/steam/` los materializan en directorios temporales.
- Los snapshots de Playwright en `frontend/e2e/visual.spec.ts-snapshots/` se
  versionan: son las baselines de `toHaveScreenshot` y sin ellos los tests
  visuales fallan en un clon limpio. Los artefactos de ejecución
  (`test-results/`, `playwright-report/`) sí están ignorados.
- Los worktrees de OpenCode pueden quedar en `HEAD` detached. Antes de
  commitear, confirmar con el usuario y crear una rama local (`fix/...`) en
  lugar de commitear sobre el detached.
- De `.codegraph/` solo se versiona su `.gitignore`; el índice (`codegraph.db`)
  es local de cada máquina y nunca debe entrar al repositorio.
- La versión publicada se define solo en `Makefile` (`VERSION`).
  `build/config.yml` es un espejo generado por `scripts/release.sh` que Wails
  necesita literal, y `packaging/nfpm.yaml` recibe la versión por `VERSION`. El
  paquete privado del frontend usa `0.0.0` fijo; no es una fuente de versión. Al
  preparar una versión nueva, mueve las entradas de `[Unreleased]` a la sección
  de la versión con su fecha y actualiza los enlaces de comparación del final de
  `CHANGELOG.md`.
- El updater vive en `internal/update`: el frontend nunca llama a la API de
  GitHub. La caché se guarda en `$XDG_CONFIG_HOME/better-rpt/update.json`
  (24 h). Al reiniciar tras actualizar, el proceso nuevo recibe
  `BETTER_RPT_WAIT_FOR_INSTANCE_LOCK=1` y espera a que el anterior libere el
  lock de instancia.
- El shell Wails requiere el CLI `wails3` instalado con `-tags gtk3`; `make build`
  lo busca en `PATH` y en `$(go env GOPATH)/bin`.
- `make setup` solo instala paquetes del sistema mediante `scripts/setup.sh`
  (apt, dnf, pacman o zypper); Go, Node/npm y el CLI de Wails se instalan
  aparte. `make setup-check` informa sin root y sin modificar el sistema.
- `go test ./...` compila `cmd/better-rpt/main_stub.go` por defecto; para compilar
  la entrada Wails hay que usar `-tags gtk3`. Los bindings generados en
  `frontend/bindings/` se versionan para que el frontend compile sin regenerarlos.
- Los tests de `internal/i18n` y `internal/ui/theme_tokens_test.go` comparan los
  catálogos y los tokens CSS con sus fuentes Go; si cambia la paleta o el copy,
  hay que actualizar ambos lados.
- En Vitest/jsdom, Radix Tabs cambia de pestaña con `fireEvent.mouseDown`; un
  `click` simple no dispara el cambio.
- El toast de `frontend/src/components/ui/toast.tsx` usa Radix (`radix-ui`);
  no reintroducir `@base-ui/react` para una sola primitiva.
- El wizard de setup se deriva de `setup.json`, `SetupSnapshot`, `Situation` y
  el informe de diagnóstico; no se añade un flag de primera vez ni un archivo
  de estado del wizard. `installed` y `restored` no lo abren automáticamente.
- El comando del dispatcher se construye solo en
  `config.BuildDispatcherLaunchOption`. La pestaña Guía y los wizards comparten
  `frontend/src/components/DispatcherSetup.tsx`; no duplicar el comando ni el
  orden de guardar el original antes de copiar.
- Steam puede reponer `configset_controller_<tipo>.vdf` con `"autosave" "1"`
  después de guardar un layout. Better RPT sustituye solo esa selección
  automática por `controller_<tipo>_gamepad_joystick.vdf`, también en
  `controller_mobile_touch`; el archivo `config/480/controller_<tipo>.vdf` es el
  autosave y no se modifica. La reparación se repite antes de cada lanzamiento.
- `Service.EnsureHelper` se ejecuta al arrancar la GUI y copia el helper
  empaquetado a la ruta XDG estable. La instalación y la restauración del
  dispatcher siguen exigiendo Steam cerrado; ningún wizard lo cierra.
- `make deb` depende de `make frontend` (bindings y assets), no de `make build`:
  compila en Ubuntu 22.04 con `-tags "gtk3 production"` y requiere Docker/nfpm.
  El binario del host no se usa en la release; el AppImage se construye con los
  binarios que deja el contenedor. `AppRun` valida GTK3/WebKit2GTK 4.1 en el
  host.
- Los ledgers de gates de las tareas de `unlazy` viven en `.unlazy/<tarea>/GATES.md`,
  que está ignorado; no se versiona ningún `GATES*.md` en la raíz.
- `.code-health-state.json` es estado local de las auditorías de `code-health`; está
  ignorado y no se versiona.
- `make dev` pasa por `scripts/dev.sh`, que aísla `wails3` en su propio grupo de
  procesos y retira restos huérfanos (Vite en 9245, `wails3 task run` y la app)
  antes de arrancar; `make dev-stop` fuerza esa limpieza. No volver a invocar
  `wails3 dev` directamente desde el Makefile.
- La versión de Go se repite en `go.mod`, `Makefile`, `packaging/Dockerfile.build`,
  `scripts/build-deb.sh`, `README.md` y `docs/`; actualízala en todos a la vez para
  que el toolchain del build sea el mismo que escanea `govulncheck`.
- `no-commit-to-branch` está desactivado en `.pre-commit-config.yaml`: se permiten
  los commits directos a `main`. `scripts/release.sh` conserva
  `SKIP=no-commit-to-branch`, que con el hook ausente es un no-op y mantiene el
  resto de hooks activos.
