# Better RPT — guía de uso

## La marca

**Relay Link** representa una sesión de Remote Play Together: el play es el
juego; las flechas son el vídeo que sale (ámbar) y el input que vuelve (teal).
Las puntas quedan a la misma distancia del play y los ejes pasan por debajo:
el enlace se lee como una sola pieza, no como tres objetos separados.

## Archivos

| Archivo | Uso |
|---|---|
| `icon.svg` / `icon-small.svg` | Tile de la app; el corte small usa la marca un 8 % más grande para 16–32 px |
| `icon-16 … icon-512.png` | Icono de escritorio, ventana y empaquetado |
| `icon-small-128.png` | Corte small para el encabezado de la app (28 px a 2×) |
| `mark.svg` | Marca suelta para fondo oscuro (play crema) |
| `mark-light.svg` | Marca suelta para fondo claro (play ink) |
| `mark-mono.svg` / `mark-white.svg` | Una tinta / reverso |
| `wordmark.svg` / `wordmark-white.svg` | Logotipo trazado, sin texto vivo |
| `lockup-horizontal*.svg` / `lockup-stacked*.svg` | Bloque horizontal y vertical, claro/oscuro, una tinta y reverso |
| `lockup-*.png` | Exportes transparentes para fondos claros u oscuros |
| `favicon.ico`, `favicon-*.png` | Navegador, documentación y accesos |
| `brand-board.png` | Lámina de presentación del sistema |
| `PALETTE.md` | Valores, proporción y contraste |

## Espacio de respeto

Deja al menos **0,25× la altura de la marca** libre alrededor del bloque. Con la
marca a 100 px, son 25 px por lado. No metas texto, bordes ni otros iconos en
ese margen.

## Tamaños mínimos

- Icono de app: **16 px**; usa `icon-small` para 16, 24 y 32 px.
- Bloque horizontal: **160 px** de ancho.
- Bloque vertical: **96 px** de ancho.
- Logotipo suelto: **80 px** de ancho.
- Por debajo de 16 px, no uses la marca: usa solo el favicon simplificado.

## Fondos aprobados

- **Oscuro:** ink `#0B1418` con la marca principal (play crema) y bloque dark.
- **Claro:** cream `#F8FAFC` con `mark-light` y bloque light (play ink).
- **Una tinta:** ink sobre claro; crema sobre oscuro.
- **Reverso:** crema sobre ink o fotografía oscura con suficiente contraste.
- Evita fondos con detalle bajo la marca; si no puedes controlarlo, usa el tile
  ink del icono.

## Tipografía

El logotipo es **IBM Plex Sans SemiBold** (OFL), trazado a curvas. No lo
recompongas con texto vivo ni cambies el tracking: usa los SVG entregados. Si
necesitas más texto de marca, IBM Plex Sans Regular/SemiBold es la familia
aprobada; ya está incluida en la app.

## No

- No rotes, inclines, deformes ni recortes la marca.
- No cambies los colores, añadas gradientes, sombras, biseles o contornos.
- No separes el play de las flechas ni muevas una punta para “equilibrar”.
- No pongas la marca principal en un fondo claro sin el play ink.
- No escribas “Better RPT” con otra fuente para sustituir el logotipo.
- No uses el icono por debajo de 16 px ni el bloque por debajo de su mínimo.

## Nota legal

No se ha realizado una búsqueda de marca registrada. Antes de publicar, haz una
búsqueda profesional en registros de marcas y una búsqueda inversa de imagen.
