# Imágenes de la guía

La guía de la interfaz usa cuatro imágenes:

| Recurso | Contenido | Estado |
|---|---|---|
| `frontend/public/guide/launch-options.png` | Steam → SpaceWar → Propiedades → Opciones de lanzamiento | Captura real (20/10/2026, Steam build 1791918634) |
| `frontend/public/guide/command.png` | Comando pegado en el campo de opciones | Captura real (20/10/2026, Steam build 1791918634) |
| `frontend/public/guide/popup.png` | Popup «Allow game launch?» / «¿Permitir iniciar el juego?» | Placeholder / pendiente de captura real |
| `frontend/public/guide/invite.png` | Overlay Shift+Tab → Remote Play Together | Placeholder / pendiente de captura real |

Las dos capturas reales son recortes 1200×480 del diálogo de
Propiedades, pensados para el contenedor de 160 px de alto de la guía. La ruta
personal del comando se sustituyó por `/home/user` para no publicar el nombre
de la cuenta; la app muestra la ruta real en el bloque de comando.

## Regla

No se inventan capturas ni se presentan diagramas como capturas reales. Las
imágenes de `popup.png` e `invite.png` siguen siendo **ilustraciones
placeholder** generadas localmente y etiquetadas como tales en este documento.
Deben sustituirse por capturas reales antes de publicar una guía definitiva.

## Procedimiento de sustitución

1. Cerrar cualquier partida en curso.
2. Abrir Steam y la ventana de propiedades de SpaceWar.
3. Capturar con la herramienta de captura del escritorio, recortando la zona
   relevante a 1200×480 (proporción 2.5:1).
4. Ocultar nombres de usuario, rutas personales y datos de la cuenta.
5. Guardar como PNG en la ruta correspondiente.
6. Verificar que la imagen se vea en ES/EN y en 1024/1280 px.
7. Actualizar esta tabla con la fecha y la versión de Steam.

## Verificación pendiente

- [x] Captura real de Opciones de lanzamiento.
- [x] Captura real del comando pegado.
- [ ] Captura real del popup de confirmación. Steam ya no lo muestra en esta
  cuenta (`LaunchOptionTipsShown` está agotado); hace falta otra cuenta o
  restablecer el aviso.
- [ ] Captura real del overlay/invitación. Requiere una partida en ejecución
  con el overlay de Steam disponible.
- [ ] Revisión visual ES/EN.
