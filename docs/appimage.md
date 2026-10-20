# AppImage

Decisión del ticket 10. El AppImage es el artefacto secundario para distribuciones sin `.deb`; el `.deb` sigue siendo el principal.

## Decisión principal: helper estable

El launch option de SpaceWar **nunca apunta al AppImage ni a `$APPDIR`**: esas rutas cambian al mover o actualizar el archivo. Tampoco apunta a `/tmp`.

La ruta canónica del helper es:

```text
$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper
# alternativa: ~/.local/share/better-rpt/bin/better-rpt-helper
```

Es un binario sin interfaz gráfica `better-rpt-helper`, sin Fyne ni CGO, compartido por `.deb` y AppImage. La app lo instala o actualiza al arrancar:

- **AppImage:** origen `$APPDIR/usr/lib/better-rpt/better-rpt-helper`.
- **`.deb`:** origen `/usr/lib/better-rpt/better-rpt-helper`.
- **Desarrollo:** `dist/better-rpt-helper` o junto al binario, solo para pruebas.

Instalación atómica: directorio 0700, archivo 0755, temporal en el mismo filesystem, `fsync`, `rename`, comparación de hash y verificación de propietario. Nunca se reemplaza el helper mientras hay una sesión activa: se conserva el que está usando esa sesión y se actualiza al terminar.

El launch option resultante se cita con comillas simples para soportar espacios:

```text
'/home/usuario/.local/share/better-rpt/bin/better-rpt-helper' --donor --target <appid> --session <32hex> # %command%
```

El helper valida `SteamAppId=480`/`SteamGameId=480`, el plan y el token, como se define en el ticket 03. La app usa la misma ruta en `.deb` y AppImage, de modo que cambiar de formato no deja una opción de lanzamiento huérfana. Esto revisa la ruta `/usr/bin/better-rpt --donor` del ticket 01; el resto de su transacción (backup, atomicidad, restauración exacta) no cambia.

## Estructura del AppDir

```text
BetterRPT.AppDir/
├── AppRun
├── better-rpt.desktop
├── better-rpt.png
└── usr/
    ├── bin/better-rpt
    └── lib/better-rpt/better-rpt-helper
```

`AppRun` resuelve su propio directorio y ejecuta `usr/bin/better-rpt`. El `.desktop` usa `Exec=better-rpt`, `Icon=better-rpt` y `StartupWMClass=Better RPT`.

## Compilación

El binario Fyne final se construye en Ubuntu 22.04 (glibc 2.35), igual que el `.deb`:

```bash
CGO_ENABLED=1 GOTOOLCHAIN=go1.27.2 go build -trimpath -buildvcs=false \
  -ldflags '-s -w -X main.version=0.1.1' -o dist/better-rpt ./cmd/better-rpt

CGO_ENABLED=0 GOTOOLCHAIN=go1.27.2 go build -trimpath -buildvcs=false \
  -ldflags '-s -w -X main.version=0.1.1' -o dist/better-rpt-helper ./cmd/better-rpt-helper
```

AppDir y AppImage con las herramientas fijadas:

```bash
mkdir -p dist/BetterRPT.AppDir/usr/bin \
  dist/BetterRPT.AppDir/usr/lib/better-rpt
cp dist/better-rpt dist/BetterRPT.AppDir/usr/bin/better-rpt
cp dist/better-rpt-helper \
  dist/BetterRPT.AppDir/usr/lib/better-rpt/better-rpt-helper
cp packaging/AppRun dist/BetterRPT.AppDir/AppRun
cp packaging/better-rpt.desktop dist/BetterRPT.AppDir/better-rpt.desktop
cp packaging/better-rpt.png dist/BetterRPT.AppDir/better-rpt.png
chmod 0755 dist/BetterRPT.AppDir/AppRun \
  dist/BetterRPT.AppDir/usr/bin/better-rpt \
  dist/BetterRPT.AppDir/usr/lib/better-rpt/better-rpt-helper

ARCH=x86_64 SOURCE_DATE_EPOCH=1700000000 APPIMAGE_EXTRACT_AND_RUN=1 \
  /tmp/opencode/appimage-poc/appimagetool --no-appstream \
  --runtime-file /tmp/opencode/appimage-poc/runtime-x86_64 \
  dist/BetterRPT.AppDir dist/BetterRPT-0.1.1-x86_64.AppImage
```

Herramientas verificadas:

- `appimagetool` continuous, git `854e19e`, build 313 (2026-10-04), SHA-256 `95cbe7cce9717fce90c484e34052ee7c7f1d7635b33c12525b4776826a7d29b6`.
- Runtime type2 commit `fedfad5`, SHA-256 `617aa77314d57c8b8e4fae9c3f0b35af7b30d5529cb69e98f64f92510ed98a46`, 944.632 bytes.
- Para reproducibilidad, guardar ambos binarios en el repositorio de compilación o en caché con esos hashes; no depender de `continuous` ni de la descarga en cada compilación.
- Dos compilaciones con el mismo AppDir y `SOURCE_DATE_EPOCH=1700000000` dieron el mismo SHA-256: `98be3ed19b246901ba523accc549d44a6232621be6b5fd79f9e120057fdb91fa` (13,7 MB).

## Runtime

- El runtime type2 verificado es `static-pie`; no contiene `libfuse.so.2`. Usa `fusermount`/`fusermount3` del sistema y es compatible con FUSE 3.
- Si FUSE no está disponible: ejecutar con `--appimage-extract-and-run` o `APPIMAGE_EXTRACT_AND_RUN=1`. Esa extracción temporal **no** se usa para el launch option.
- El AppImage no incluye glibc, GL, Wayland ni X11: se usan las bibliotecas del sistema. Requisitos mínimos: glibc ≥ 2.35, `libwayland-client`, `libwayland-cursor`, `libwayland-egl`, `libxkbcommon`, Mesa/GL y las bibliotecas X11 que use la sesión. Un contenedor mínimo sin ellas falla al cargar `libwayland-client.so.0`.
- Probado con el artefacto final: `--version` directo con FUSE, con `APPIMAGE_EXTRACT_AND_RUN=1` y con `--appimage-extract-and-run`; GUI bajo Xvfb con datos simulados; extracción con `--appimage-extract` verificada. También se comprobó `--version` en Fedora 44 y Arch tras instalar las bibliotecas gráficas.
- Linux Mint/Pop!_OS y otras familias con glibc ≥ 2.35 no se probaron una a una; no se declara una garantía por versión.
- **No hay pruebas de GUI de extremo a extremo en Fedora/Arch** ni una sesión real de Steam/SpaceWar: el entorno de prueba no tenía esa combinación. No se declara soporte de GUI en esas distribuciones.

## Matriz

| Sistema | AppImage | Notas |
|---|---|---|
| Ubuntu 22.04 | Sí, validado | FUSE, extracción, GUI y helper |
| Ubuntu 24.04+ / Debian 12+ | Esperado | misma base glibc; sin prueba de extremo a extremo por versión |
| Fedora 44 | Parcial | solo `--version` con libs de sistema |
| Arch | Parcial | solo `--version` con libs de sistema |
| Alpine/musl | No | glibc no disponible |
| glibc < 2.35 | No | compilado sobre Ubuntu 22.04 |
| 32-bit / arm64 | No | solo amd64 en el MVP |

## Integración y convivencia

- No se instala automáticamente un `.desktop` en el menú: el AppImage se ejecuta desde el archivo. El README puede explicar cómo añadir una entrada manual al menú o usar AppImageLauncher; no es obligatorio para el MVP.
- No hay actualización automática: el usuario reemplaza el AppImage. Al arrancar, la app actualiza el helper estable si la versión cambió.
- `.deb` y AppImage comparten helper y configuración; un bloqueo de instancia y el bloqueo de sesión impiden dos sesiones. Si hay una sesión activa, no se reemplaza el helper.
- Desinstalar el AppImage es borrar el archivo; configuración, backups y helper permanecen. El README explica el borrado manual.
- La app rechaza `euid 0`, igual que el `.deb`.

## Seguridad

- Sin actualización automática ni descarga de código en runtime.
- SHA-256 publicado para el AppImage; sin firma GPG en el MVP.
- El helper se copia desde una ruta del AppImage montado en solo lectura o desde `/usr/lib`, nunca desde `/tmp` ni desde una ruta arbitraria.
- Copia atómica, comparación de hash, permisos 0700/0755 y validación de sesión.
- El launch option se cita; se valida el AppID numérico y el token de 32 hex.
- El modo `--appimage-extract-and-run` solo sirve para arrancar la GUI si falta FUSE; no se persiste en la configuración de Steam.

## Limitaciones

- AppImage no es universal: no cubre musl/Alpine, glibc < 2.35, arm64/32-bit.
- Fedora 44 y Arch solo se probaron con `--version` y bibliotecas gráficas; no hay pruebas de GUI de extremo a extremo ni soporte declarado.
- El binario final se construyó en Ubuntu 22.04 y usa como máximo `GLIBC_2.34`; el paquete declara `libc6 >= 2.35` como mínimo soportado.
- Sin integración automática de menú ni actualización automática.
