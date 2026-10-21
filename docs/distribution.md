# Distribución

Contrato de distribución del paquete `.deb` del MVP. El AppImage es un artefacto secundario y se concreta en `docs/appimage.md`.

## Artefactos

- **`.deb` principal**: `better-rpt_0.2.3-1_amd64.deb`, para Ubuntu 22.04+, Debian 12+ y derivados. Instalación limpia probada en Ubuntu 22.04.
- **AppImage secundario**: `BetterRPT-0.2.3-x86_64.AppImage`, con el mismo helper estable; probado con FUSE, extracción y GUI en Ubuntu. Se detalla en `docs/appimage.md`.
- Sin repositorio APT, firma GPG ni actualización automática en el MVP. `dist/SHA256SUMS` publica los hashes de ambos artefactos.

## Matriz de soporte

| Sistema | `.deb` | AppImage | Notas |
|---|---|---|---|
| Ubuntu 22.04 | Sí, validado | Sí, validado | Base de compilación; instalación limpia y GUI |
| Debian 12+ | Esperado | Esperado | glibc 2.36+; sin prueba de extremo a extremo por versión |
| Linux Mint 21+ | Esperado | Esperado | Derivados de Ubuntu |
| Pop!_OS 22.04+ | Esperado | Esperado | Derivados de Ubuntu |
| Fedora 44 / Arch | No | Parcial | solo `--version` con libs gráficas |
| Alpine / musl | No | No | Sin soporte en el MVP |
| 32-bit / arm64 | No | No | Solo amd64 |

La matriz es de familias, no una garantía por versión concreta. El `.deb` se construye una vez sobre Ubuntu 22.04 y declara `libc6 >= 2.35`; no se publican paquetes separados por distro en el MVP. La cobertura completa probada es Ubuntu 22.04; Debian, Mint y Pop!_OS comparten la base glibc, pero no se probaron de extremo a extremo.

## Compilación reproducible

El artefacto final se construyó en un contenedor `ubuntu:22.04` (amd64). El
binario usa como máximo `GLIBC_2.34`; `dpkg-shlibdeps` pidió `libc6 (>= 2.34)`.
El paquete declara `libc6 (>= 2.35)` como mínimo soportado. La receta está
en `docs/build.md`.

- Go 1.27.3, Wails v3.0.0-beta.33, GTK3/WebKit2GTK 4.1, CGO,
  `-trimpath -buildvcs=false`, `-ldflags '-s -w'`.
- nfpm v2.48.0 y `SOURCE_DATE_EPOCH=1700000000`.
- El AppImage se construye aparte sobre la misma base Ubuntu 22.04 y con el
  mismo helper; dos compilaciones consecutivas del AppImage y del `.deb` dieron el
  mismo SHA-256 en cada formato. Su receta está en `docs/appimage.md`.
- `dist/SHA256SUMS` contiene los hashes publicados.

## Metadatos del paquete

- Nombre: `better-rpt`
- Versión Debian: `0.2.3-1`
- Arquitectura: `amd64`
- Sección: `games`; prioridad `optional`
- Maintainer: `srnoob2570 <302417634+srnoob2570@users.noreply.github.com>`
- Licencia: MIT
- Vendor: Better RPT

## Estructura del paquete

```text
/usr/bin/better-rpt
/usr/lib/better-rpt/better-rpt-helper
/usr/share/applications/better-rpt.desktop
/usr/share/icons/hicolor/256x256/apps/better-rpt.png
/usr/share/doc/better-rpt/README.md
/usr/share/doc/better-rpt/LICENSE
/usr/share/doc/better-rpt/copyright
```

Sin servicios, unidades systemd, setuid ni configuración de usuario instalada por el paquete. El helper sin interfaz gráfica es la fuente que la app copia a `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper` al arrancar; el launch option apunta a esa copia estable, no a `/usr/lib`.

## Dependencias

`Depends` parte de lo detectado en el binario y se fija para Ubuntu 22.04:

```text
libc6 (>= 2.35), libgl1, libegl1,
libwayland-client0 (>= 1.20.0), libwayland-cursor0, libwayland-egl1,
libxkbcommon0, libx11-6, libx11-xcb1, libxcursor1, libxi6,
libxinerama1, libxrandr2, libxxf86vm1
```

`Recommends: libdecor-0-0, libdecor-0-plugin-1-gtk` para decoraciones Wayland. El plugin arrastra GTK3, no WebKitGTK. `dpkg-shlibdeps -O` sobre el binario final devolvió `libc6 (>= 2.34), libgl1, libwayland-client0 (>= 1.20.0)`; la lista manual añade las bibliotecas que GLFW carga con `dlopen`. Ajustar solo con evidencia de `ldd`/`strace`/`dpkg-shlibdeps`; no añadir dependencias a ciegas.

## Instalación

```bash
sudo apt install ./better-rpt_0.2.3-1_amd64.deb
```

Si `apt` no resuelve el archivo local:

```bash
sudo dpkg -i ./better-rpt_0.2.3-1_amd64.deb
sudo apt-get install -f
```

Verificar el SHA-256 publicado con `sha256sum -c SHA256SUMS`. No se requiere
compilar, clonar ni editar archivos de Steam a mano. La instalación limpia,
`--version`, el arranque GUI como usuario normal y el rechazo de root se
probaron en Ubuntu 22.04. La app se abre desde el menú de aplicaciones o con
`better-rpt`.

## Configuración y datos

- Configuración: `$XDG_CONFIG_HOME/better-rpt/config.json` o `~/.config/better-rpt/config.json` (idioma, preferencias).
- Helper estable: `$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper` o `~/.local/share/better-rpt/bin/better-rpt-helper` (dir 0700, archivo 0755).
- Backups: `$XDG_DATA_HOME/better-rpt/backups/` o `~/.local/share/better-rpt/backups/` (dir 0700, archivos 0600, retención 10; `Kind=session` o `dispatcher`).
- Setup del dispatcher: `$XDG_DATA_HOME/better-rpt/setup.json` (0600), con el backup y el valor original de SpaceWar.
- Sesiones: `$XDG_DATA_HOME/better-rpt/sessions/` con plan/estado, `launch-intent.json` y `last-launch.json` (0600).
- Log persistente: `$XDG_DATA_HOME/better-rpt/logs/better-rpt.log`, rotado a 1 MiB con tres copias y saneado.
- Informes de diagnóstico: `$XDG_DATA_HOME/better-rpt/diagnostics/` (Markdown `0600`).
- Runtime: `$XDG_RUNTIME_DIR/better-rpt/` para el bloqueo de sesión; no es dato persistente.
- El paquete no crea ni modifica estos directorios; los crea la app al ejecutarse como usuario.

## Instalación, upgrade y desinstalación

- **Upgrade**: `sudo apt install ./better-rpt_<nueva>_amd64.deb` reemplaza binario, `.desktop`, icono y docs; conserva configuración y backups. No hay migración de datos en el MVP.
- **Desinstalación**: `sudo apt remove better-rpt` elimina solo archivos del sistema. No hay `postrm` que toque archivos del usuario.
- **Aviso**: un `prerm` no interactivo comprueba si `better-rpt` está en ejecución y escribe un aviso por stderr. No bloquea la desinstalación ni modifica nada.
- **Datos**: configuración y backups permanecen tras `remove` y `purge`. El README explica el borrado manual.
- **Sesión activa, setup instalado o backup pendiente**: el usuario debe abrir Better RPT y pulsar **Restaurar** antes de desinstalar. Con Steam abierto, la app muestra la guía para quitar el comando; no cierra Steam. Si ya desinstaló, puede reinstalar la misma versión, restaurar y volver a desinstalar. No se edita `localconfig.vdf` desde el paquete.

Texto del aviso:

> Better RPT está en ejecución. Ciérralo y restaura SpaceWar antes de desinstalar. Si preparaste una sesión y ya no está instalado, reinstala esta misma versión para restaurar.

## Root

La app detecta `os.Geteuid() == 0` y se niega a arrancar con un mensaje claro. El paquete instala archivos del sistema; la app se ejecuta siempre como el usuario de Steam. No se admiten configuraciones ni backups con propietario root.

## AppImage

El AppImage es el artefacto secundario para distribuciones con glibc ≥ 2.35,
sin root. Usa el mismo helper estable que el `.deb`
(`$XDG_DATA_HOME/better-rpt/bin/better-rpt-helper`), runtime type2 estático con
FUSE 3 y la alternativa `--appimage-extract-and-run`. No empaqueta Wayland, GL ni X11:
el sistema debe tener esas bibliotecas. Se probó con FUSE, extracción y GUI en
Ubuntu; en Fedora 44 y Arch solo se verificó `--version` con las bibliotecas
gráficas. No cubre musl/Alpine, glibc antigua, 32-bit ni arm64, y no se declara
soporte de GUI en Fedora/Arch.

La decisión completa, la estructura del AppDir, los hashes de las herramientas,
la matriz y las limitaciones están en [`docs/appimage.md`](appimage.md).

## Fuera de alcance del paquete

- Repositorio APT, firma GPG y actualización automática.
- Flatpak/Snap de Steam y ejecución del helper dentro de un sandbox.
- Restauración automática desde `postrm`.
- Paquetes por distro o arquitecturas distintas de amd64.
