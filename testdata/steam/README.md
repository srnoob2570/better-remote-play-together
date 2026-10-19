# Steam fixtures — Better RPT

Árbol sintético para probar detección de Steam, parseo de VDF/appinfo, planificación de lanzamiento y backup/restore **sin tocar una instalación real**.

No contiene datos de juegos reales salvo SpaceWar (App ID 480), que es el donante fijo del producto. Los nombres y App IDs `9000xx` son ficticios.

## Estructura

```text
fixtures/
├── primary/                      Steam raíz principal
│   ├── appcache/
│   │   ├── appinfo.vdf           v41 válido
│   │   ├── appinfo_v40.vdf       versión no soportada
│   │   ├── appinfo_corrupt.vdf   offset de string table inválido
│   │   └── appinfo_bad_key.vdf   apps válidas + registro 900013 corrupto
│   ├── config/config.vdf         CompatToolMapping global y por app
│   ├── loginusers.vdf            dos cuentas; MostRecent = 11111111
│   ├── registry.vdf              AutoLoginUser y SteamPID 0
│   ├── steamapps/
│   │   ├── libraryfolders.vdf.tmpl
│   │   ├── appmanifest_*.acf
│   │   └── common/
│   ├── compatibilitytools.d/GE-Proton-test/
│   └── userdata/
│       ├── 11111111/config/      cuenta activa y variantes de localconfig
│       └── 22222222/config/      segunda cuenta
├── secondary/                    biblioteca secundaria (sin libraryfolders)
└── legacy/                       raíz con libraryfolders en formato antiguo
```

Las rutas absolutas de `libraryfolders.vdf` no se pueden fijar en un repo portable. Por eso los archivos terminan en `.tmpl` y usan `{{PRIMARY}}`, `{{SECONDARY}}` y `{{LEGACY}}`.

## Materializar

```bash
bash testdata/steam/generate.sh          # regenera fixtures/ y los appinfo
bash testdata/steam/materialize.sh       # crea una copia temporal y sustituye rutas
```

`materialize.sh` imprime el directorio temporal. Los tests deben hacer lo mismo con `t.TempDir()`: copiar `fixtures/` y sustituir los placeholders; nunca apuntar la app a `fixtures/` directamente.

## Apps

| App ID | Nombre | Tipo | Estado esperado |
|---|---|---|---|
| 480 | Spacewar | donante | instalado, launch Windows |
| 900001 | Native Game | juego | listo, launch Linux + Windows |
| 900002 | Proton Game | juego | Proton vía `GE-Proton-test` |
| 900003 | Proton Test | herramienta | excluida |
| 900004 | SteamLinuxRuntime Test | runtime | excluido |
| 900005 | Not Installed Game | juego | `StateFlags 2`, no instalado |
| 900006 | Missing Exe | juego | manifiesto presente, ejecutable ausente |
| 900007 | Secondary Native | juego | en biblioteca secundaria, mapping `native` |
| 900008 | Legacy Game | juego | en raíz con `libraryfolders` legacy |
| 900009 | No Launch Config | juego | sin `config.launch` |
| 900010 | Orphan Manifest | juego | manifiesto sin appinfo |
| 900011 | Template Launch | juego | `arguments` contiene `%command%` |
| 900012 | Path Escape | juego | ejecutable `../evil` |
| 900013 | Bad Key | juego | solo en `appinfo_bad_key.vdf`, key inválida |

## Appinfo

`gen_appinfo.go` es el generador reproducible. Escribe:

- `appinfo.vdf`: v41, 12 apps, con apps nativas, Windows, herramientas y sin launch.
- `appinfo_v40.vdf`: magic v40; el parser debe devolver `ErrUnsupportedVersion`.
- `appinfo_corrupt.vdf`: string-table offset fuera de rango; `ErrCorrupt`.
- `appinfo_bad_key.vdf`: todas las apps válidas más 900013 con key inválida; el parser debe omitir solo esa entrada y registrar un warning.

Detalles del fixture:

- `InfoState = 2`; `LastUpdated = 0`; token 0; change number 0.
- El primer SHA-1 (VDF) está a cero; el segundo es el SHA-1 real del binary VDF. El parser no debe exigir el primero en fixtures sintéticos.
- `appinfo` es el objeto raíz; las claves son índices a la string table y los valores string van inline.
- `appinfo.vdf` no contiene 900010 a propósito: el manifiesto existe, la appinfo no.

## Config de compatibilidad

- `config.vdf` usa `InstallConfigStore > Software > Valve > Steam > CompatToolMapping`.
- Global `0` → `GE-Proton-test` (prioridad 250).
- `900002` → `GE-Proton-test`.
- `900007` → `native`.
- `GE-Proton-test/toolmanifest.vdf` declara `require_tool_appid 900004`; el runtime de prueba está en `steamapps/common/SteamLinuxRuntime_test`.
- `Proton Test` también tiene `toolmanifest.vdf` para probar la exclusión de herramientas.

## Localconfig

La cuenta activa es `11111111` (SteamID64 `76561197971376839`), con `MostRecent 1`. La segunda cuenta es `22222222`.

- `localconfig.vdf`: 480 vacío, 900001 con launch option ajena, 900002 vacío; incluye bloques `apps` señuelo en `WebStorage` y en la raíz.
- `localconfig_duplicate.vdf`: dos entradas 480 bajo el `apps` correcto.
- `localconfig_missing.vdf`: no existe 480; sirve para probar la inserción del bloque.
- `localconfig_no_launchoptions.vdf`: 480 existe pero sin `LaunchOptions`.
- `localconfig_malformed.vdf`: VDF sin cerrar.

## Casos que los tests deben crear

El árbol de Git no puede representar de forma fiable permisos `000`, ACLs, xattrs ni symlinks que escapan. Los tests de integración deben crear esos casos sobre una copia temporal:

- `chmod 000` en `localconfig.vdf` o en el directorio de backups.
- symlink `common/Path Escape/evil` apuntando fuera de la biblioteca.
- backup ausente o metadata de backup corrupta.
- dos cuentas sin `MostRecent` y con el mismo `Timestamp` (ambigüedad).

## Regenerar

```bash
bash testdata/steam/generate.sh
```

El script borra y recrea `fixtures/`; no toca `$HOME/.local/share/Steam`. El generador de appinfo se ejecuta con Go y no requiere módulo: `go run ./testdata/steam/gen_appinfo.go`.
