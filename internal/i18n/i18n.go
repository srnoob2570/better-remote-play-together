// Package i18n contains the small, static Spanish/English copy catalogue used
// by the GUI. The Spanish catalogue is the fallback for unknown languages and
// for keys that are missing from the selected language.
package i18n

import (
	"os"
	"strings"
)

// Language is a supported UI language.
type Language string

const (
	Spanish Language = "es"
	English Language = "en"
)

var catalogs = map[Language]map[string]string{
	Spanish: {
		"app.title":       "Better RPT",
		"app.subtitle":    "Remote Play Together para juegos que no lo traen.",
		"app.bootstrap":   "Better RPT está listo.",
		"app.version":     "Versión",
		"app.language":    "Idioma",
		"app.language.es": "Español",
		"app.language.en": "English",

		"search.placeholder":  "Buscar por nombre o App ID",
		"search.empty":        "Ningún juego coincide con la búsqueda.",
		"filter.all":          "Todos",
		"filter.ready":        "Listos",
		"filter.unknown":      "Desconocidos",
		"filter.incompatible": "Incompatibles",

		"library.title":        "Biblioteca",
		"library.columnGame":   "Juego",
		"library.columnKind":   "Tipo",
		"library.columnStatus": "Estado",
		"selection.hint":       "Elige un juego de la biblioteca para preparar la sesión.",
		"selection.actionHint": "Se lanzará con el comando instalado; Steam no se cerrará.",
		"selection.guideHint":  "Instala una vez el comando de la guía para jugar sin cerrar Steam.",
		"selection.donorHint":  "SpaceWar debe estar instalado para iniciar la sesión.",

		"action.clear":           "Limpiar",
		"action.resetFilters":    "Quitar filtros",
		"action.clearSearch":     "Limpiar búsqueda",
		"action.refresh":         "Actualizar",
		"action.refreshing":      "Actualizando…",
		"action.prepare":         "Preparar y jugar",
		"action.play":            "Jugar",
		"action.prepareAgain":    "Volver a preparar",
		"action.playAgain":       "Jugar de nuevo",
		"action.openLast":        "Abrir última sesión",
		"action.viewGuide":       "Ver guía",
		"action.copyCommand":     "Copiar comando",
		"action.installAuto":     "Instalar automáticamente",
		"action.prepareGuide":    "Preparar guía",
		"action.refreshSetup":    "Verificar de nuevo",
		"action.restore":         "Restaurar",
		"action.restoreSpaceWar": "Restaurar SpaceWar",
		"action.cancel":          "Cancelar",
		"action.cancelRestore":   "Cancelar y restaurar",
		"action.retry":           "Reintentar",
		"action.details":         "Detalles",
		"action.addSpacewar":     "Añadir SpaceWar",
		"action.openSteam":       "Abrir Steam",

		"kind.native": "Nativo",
		"kind.proton": "Proton",

		"status.ready":        "Listo",
		"status.unknown":      "Desconocido",
		"compat.unknownHint":  "Remote Play Together no está confirmado.",
		"status.incompatible": "Incompatible",
		"status.missing":      "Falta",
		"status.error":        "Error",
		"status.active":       "Sesión activa",

		"library.loading":   "Leyendo la biblioteca de Steam…",
		"library.empty":     "No hay juegos instalados.",
		"library.emptyHint": "Instala un juego en Steam y pulsa Actualizar.",
		"library.warnings":  "Avisos de la biblioteca",
		"library.count":     "%d juegos",
		"library.countOne":  "1 juego",

		"details.title":      "Detalles",
		"details.appid":      "App ID",
		"details.library":    "Biblioteca",
		"details.path":       "Ruta",
		"details.kind":       "Tipo",
		"details.proton":     "Proton",
		"details.problem":    "Motivo",
		"details.emptyTitle": "Sin juego seleccionado",
		"details.emptyHint":  "Elige un juego de la biblioteca para ver sus detalles.",

		"donor.title":           "Donante: SpaceWar (480)",
		"donor.name":            "SpaceWar",
		"donor.role":            "Donante de Remote Play Together",
		"donor.self":            "Este es el donante",
		"donor.ready":           "Listo",
		"donor.spacewarMissing": "SpaceWar no está instalado",
		"donor.preparing":       "Preparando",
		"donor.active":          "Sesión activa",
		"donor.ended":           "Sesión terminada",
		"donor.error":           "Error",
		"donor.stale":           "Restauración pendiente",

		"prep.title":         "Preparando la sesión",
		"prep.step":          "Paso %d de %d",
		"prep.verify":        "Verificar Steam y SpaceWar",
		"prep.installHelper": "Instalar o actualizar el helper",
		"prep.armIntent":     "Preparar el lanzamiento",
		"prep.startDonor":    "Pedir a Steam que abra SpaceWar",
		"prep.waitHelper":    "Esperar al juego",
		"prep.cancelHint":    "Puedes cancelar antes de que empiece la sesión; Steam no se cerrará.",

		"session.title":      "Sesión activa",
		"session.section":    "Sesión",
		"session.donorRole":  "Donante",
		"session.targetRole": "Objetivo",
		"session.guide":      "Cómo invitar",
		"session.elapsed":    "Tiempo",
		"session.guide1":     "Abre el overlay de Steam con Shift+Tab.",
		"session.guide2":     "Invita a tus amigos a Remote Play Together.",
		"session.guide3":     "El juego se ejecuta dentro del contexto de SpaceWar.",
		"session.locked":     "La biblioteca sigue disponible, pero no se puede iniciar otra sesión.",
		"session.actionHint": "Cierra el juego para terminar la sesión.",
		"session.starting":   "Iniciando la sesión…",
		"session.tabGuide":   "Guía",

		"last.title":            "Última sesión",
		"last.playing":          "Jugando ahora",
		"last.started":          "Inicio",
		"last.ended":            "Fin",
		"last.duration":         "Duración",
		"last.source.betterRpt": "Iniciado por Better RPT",
		"last.source.steam":     "Iniciado desde Steam",
		"last.outcome.pending":  "Preparando",
		"last.outcome.ok":       "Terminada",
		"last.outcome.error":    "Error",
		"last.outcome.timeout":  "Sin respuesta de Steam",
		"last.outcome.canceled": "Cancelada",
		"last.outcome.unknown":  "Origen desconocido",
		"last.empty":            "Aún no hay una sesión registrada.",
		"last.hint":             "Puedes repetir el lanzamiento o restaurar SpaceWar.",
		"last.setupMissing":     "Falta instalar el comando de la guía para volver a jugar.",

		"guide.title":               "Guía de lanzamiento",
		"guide.intro":               "Better RPT no cierra Steam. Instala una vez el comando en SpaceWar y luego lanza el juego desde aquí.",
		"guide.command":             "Comando para SpaceWar",
		"guide.commandHint":         "Pégalo tal cual en Opciones de lanzamiento. Incluye %command%.",
		"guide.prepareHint":         "Preparar guía guarda el valor original de SpaceWar antes de que pegues el comando, para poder restaurarlo después.",
		"guide.step1.title":         "Abre SpaceWar en Steam",
		"guide.step1.body":          "Biblioteca → SpaceWar (480) → clic derecho → Propiedades.",
		"guide.step2.title":         "Abre Opciones de lanzamiento",
		"guide.step2.body":          "En la pestaña General, busca Opciones de lanzamiento.",
		"guide.step3.title":         "Pega el comando",
		"guide.step3.body":          "Sustituye el contenido actual por el comando de arriba. Better RPT guarda una copia del valor original antes de continuar.",
		"guide.step4.title":         "Guarda y lanza",
		"guide.step4.body":          "Cierra Propiedades y pulsa Jugar en Steam, o usa Jugar de nuevo en Better RPT.",
		"guide.step5.title":         "Acepta el aviso de Steam",
		"guide.step5.body":          "Steam puede mostrar «¿Permitir iniciar el juego?». Acepta para que el helper arranque el juego con los parámetros de la sesión.",
		"guide.step6.title":         "Invita a tus amigos",
		"guide.step6.body":          "Con el juego abierto, abre el overlay con Shift+Tab e invita a Remote Play Together.",
		"guide.steamOpen":           "Steam está abierto. Pega el comando desde la UI de Steam; Better RPT no lo cerrará. Si el estado no cambia, usa «Verificar de nuevo» o cierra Steam una vez para que guarde localconfig.",
		"guide.steamClosed":         "Steam está cerrado. Puedes instalar el comando automáticamente.",
		"guide.autoUnavailable":     "Cierra Steam manualmente para usar la instalación automática. Better RPT no lo cerrará.",
		"guide.installDone":         "Comando instalado. Ya puedes jugar sin cerrar Steam.",
		"guide.recordDone":          "Copia del valor original guardada. Pega el comando en Steam.",
		"guide.copyDone":            "Comando copiado.",
		"guide.setupMissing":        "Aún no hay una copia del valor original.",
		"guide.setupGuide":          "Guía preparada; falta pegar el comando en Steam.",
		"guide.setupInstalled":      "El comando ya está instalado.",
		"guide.setupRestored":       "SpaceWar está restaurado.",
		"guide.setupConflict":       "La configuración de SpaceWar cambió fuera de Better RPT. No se sobrescribirá.",
		"guide.image.launchOptions": "Opciones de lanzamiento de SpaceWar",
		"guide.image.command":       "Comando pegado",
		"guide.image.popup":         "Aviso de confirmación de Steam",
		"guide.image.invite":        "Overlay de Remote Play Together",
		"guide.image.placeholder":   "Ilustración pendiente de sustituir por una captura real de Steam.",
		"guide.originalValue":       "Valor original",
		"guide.originalEmpty":       "El valor original está vacío.",
		"guide.restoreHint":         "Para restaurar sin cerrar Steam, pega el valor original en Opciones de lanzamiento. También puedes usar Restaurar cuando Steam esté cerrado.",
		"guide.manualRestore":       "Restauración manual",

		"situation.prepared":       "Preparada",
		"situation.active":         "Activa",
		"situation.finished":       "Terminada",
		"situation.restorePending": "Restauración pendiente",
		"situation.restored":       "Restaurada",
		"situation.orphan":         "Configuración huérfana",
		"situation.conflict":       "Conflicto",
		"situation.idle":           "Inactiva",

		"setup.title":     "Integración de SpaceWar",
		"setup.missing":   "No instalada",
		"setup.guide":     "Guía preparada",
		"setup.installed": "Instalada",
		"setup.restored":  "Restaurada",
		"setup.conflict":  "Conflicto",

		"pending.title":       "Configuración anterior",
		"pending.body":        "SpaceWar conserva una configuración de una sesión anterior de %s. No es la última sesión; puedes restaurarla o consultar la guía.",
		"pending.orphanTitle": "Configuración huérfana",
		"pending.orphanBody":  "SpaceWar apunta a Better RPT, pero no hay una sesión o copia coherente. No se tocará; consulta la guía.",
		"pending.noBackup":    "No hay copia de seguridad; no se puede restaurar automáticamente.",
		"pending.restore":     "Restaurar configuración anterior",
		"pending.guide":       "Ver cómo restaurar",

		"restore.reason.steamRunning":  "Steam está abierto. Ciérralo manualmente o restaura desde la guía; Better RPT no lo cerrará.",
		"restore.reason.sessionActive": "La sesión sigue activa. Cierra el juego antes de restaurar.",
		"restore.reason.conflict":      "La configuración actual no coincide con la copia. No se sobrescribirá.",
		"restore.reason.noBackup":      "No hay una copia de seguridad asociada.",
		"restore.reason.notNeeded":     "No hay nada que restaurar.",

		"error.title":           "No se pudo completar la operación",
		"error.body":            "Revisa el detalle técnico y vuelve a intentarlo.",
		"error.detail":          "Detalle técnico",
		"error.refresh":         "No se pudo leer la biblioteca de Steam.",
		"error.session":         "No se pudo iniciar o consultar la sesión.",
		"error.restoreConflict": "La opción de lanzamiento cambió fuera de Better RPT. No se sobrescribió.",
		"error.noBackup":        "No hay una copia asociada a esta sesión.",
		"error.steamMissing":    "No se encontró una instalación nativa de Steam.",
		"error.spacewarMissing": "SpaceWar (480) no está instalado.",
		"error.setupRequired":   "Instala el comando desde la guía antes de jugar.",
		"error.setupConflict":   "La configuración de SpaceWar cambió fuera de Better RPT. No se sobrescribió.",
		"error.replay":          "No se pudo repetir la última sesión.",

		"dialog.unknown.title":       "Compatibilidad desconocida",
		"dialog.unknown.body":        "Este juego usa Proton y su compatibilidad con Remote Play Together no está verificada. ¿Intentar de todos modos?",
		"dialog.restore.title":       "Restaurar SpaceWar",
		"dialog.restore.body":        "Se restaurará la opción de lanzamiento original de SpaceWar. ¿Continuar?",
		"dialog.cancelRestore.title": "Cancelar la sesión",
		"dialog.cancelRestore.body":  "La opción de lanzamiento ya está instalada. Al cancelar se restaurará SpaceWar. ¿Continuar?",
		"dialog.addSpacewar.title":   "Añadir SpaceWar",
		"dialog.addSpacewar.body":    "Se abrirá Steam para instalar SpaceWar (480), el donante de Remote Play Together.",

		"restore.done.title": "SpaceWar restaurado",
		"restore.done.body":  "La configuración de lanzamiento de SpaceWar volvió a su estado original.",

		"common.continue": "Continuar",
		"common.cancel":   "Cancelar",
		"common.close":    "Cerrar",
		"common.ignore":   "Ignorar",
		"common.unknown":  "Desconocido",
		"common.yes":      "Sí",
		"common.no":       "No",

		"language.es": "Español",
		"language.en": "English",

		"game.reason.launcher":  "Puede requerir un launcher externo.",
		"game.reason.anticheat": "Puede usar anti-cheat y no ser compatible.",
		"primary.select":        "Selecciona un juego",

		"error.generic":         "Ocurrió un error.",
		"error.root":            "Better RPT no puede ejecutarse como root.",
		"error.instance":        "Better RPT ya está abierto.",
		"error.config":          "No se pudo leer la configuración.",
		"error.paths":           "No se pudieron resolver las carpetas de usuario.",
		"error.unsupported":     "Esta instalación de Steam no está soportada.",
		"helper.notImplemented": "La ejecución de sesiones todavía no está implementada.",
	},
	English: {
		"app.title":       "Better RPT",
		"app.subtitle":    "Remote Play Together for games that do not include it.",
		"app.bootstrap":   "Better RPT is ready.",
		"app.version":     "Version",
		"app.language":    "Language",
		"app.language.es": "Español",
		"app.language.en": "English",

		"search.placeholder":  "Search by name or App ID",
		"search.empty":        "No games match your search.",
		"filter.all":          "All",
		"filter.ready":        "Ready",
		"filter.unknown":      "Unknown",
		"filter.incompatible": "Incompatible",

		"library.title":        "Library",
		"library.columnGame":   "Game",
		"library.columnKind":   "Type",
		"library.columnStatus": "Status",
		"selection.hint":       "Choose a game from the library to prepare the session.",
		"selection.actionHint": "It will launch with the installed command; Steam will not be closed.",
		"selection.guideHint":  "Install the command from the guide once to play without closing Steam.",
		"selection.donorHint":  "SpaceWar must be installed before a session can start.",

		"action.clear":           "Clear",
		"action.resetFilters":    "Clear filters",
		"action.clearSearch":     "Clear search",
		"action.refresh":         "Refresh",
		"action.refreshing":      "Refreshing…",
		"action.prepare":         "Prepare and play",
		"action.play":            "Play",
		"action.prepareAgain":    "Prepare again",
		"action.playAgain":       "Play again",
		"action.openLast":        "Open last session",
		"action.viewGuide":       "View guide",
		"action.copyCommand":     "Copy command",
		"action.installAuto":     "Install automatically",
		"action.prepareGuide":    "Prepare guide",
		"action.refreshSetup":    "Check again",
		"action.restore":         "Restore",
		"action.restoreSpaceWar": "Restore SpaceWar",
		"action.cancel":          "Cancel",
		"action.cancelRestore":   "Cancel and restore",
		"action.retry":           "Retry",
		"action.details":         "Details",
		"action.addSpacewar":     "Add SpaceWar",
		"action.openSteam":       "Open Steam",

		"kind.native": "Native",
		"kind.proton": "Proton",

		"status.ready":        "Ready",
		"status.unknown":      "Unknown",
		"compat.unknownHint":  "Remote Play Together is not confirmed.",
		"status.incompatible": "Incompatible",
		"status.missing":      "Missing",
		"status.error":        "Error",
		"status.active":       "Session active",

		"library.loading":   "Reading the Steam library…",
		"library.empty":     "No games are installed.",
		"library.emptyHint": "Install a game in Steam and press Refresh.",
		"library.warnings":  "Library warnings",
		"library.count":     "%d games",
		"library.countOne":  "1 game",

		"details.title":      "Details",
		"details.appid":      "App ID",
		"details.library":    "Library",
		"details.path":       "Path",
		"details.kind":       "Kind",
		"details.proton":     "Proton",
		"details.problem":    "Reason",
		"details.emptyTitle": "No game selected",
		"details.emptyHint":  "Choose a game from the library to see its details.",

		"donor.title":           "Donor: SpaceWar (480)",
		"donor.name":            "SpaceWar",
		"donor.role":            "Remote Play Together donor",
		"donor.self":            "This is the donor",
		"donor.ready":           "Ready",
		"donor.spacewarMissing": "SpaceWar is not installed",
		"donor.preparing":       "Preparing",
		"donor.active":          "Session active",
		"donor.ended":           "Session ended",
		"donor.error":           "Error",
		"donor.stale":           "Restore pending",

		"prep.title":         "Preparing the session",
		"prep.step":          "Step %d of %d",
		"prep.verify":        "Verify Steam and SpaceWar",
		"prep.installHelper": "Install or update the helper",
		"prep.armIntent":     "Prepare the launch",
		"prep.startDonor":    "Ask Steam to open SpaceWar",
		"prep.waitHelper":    "Wait for the game",
		"prep.cancelHint":    "You can cancel before the session starts; Steam will not be closed.",

		"session.title":      "Session active",
		"session.section":    "Session",
		"session.donorRole":  "Donor",
		"session.targetRole": "Target",
		"session.guide":      "How to invite",
		"session.elapsed":    "Elapsed",
		"session.guide1":     "Open the Steam overlay with Shift+Tab.",
		"session.guide2":     "Invite your friends to Remote Play Together.",
		"session.guide3":     "The game runs inside the SpaceWar context.",
		"session.locked":     "The library stays available, but another session cannot start.",
		"session.actionHint": "Close the game to end the session.",
		"session.starting":   "Starting the session…",
		"session.tabGuide":   "Guide",

		"last.title":            "Last session",
		"last.playing":          "Playing now",
		"last.started":          "Started",
		"last.ended":            "Ended",
		"last.duration":         "Duration",
		"last.source.betterRpt": "Started by Better RPT",
		"last.source.steam":     "Started from Steam",
		"last.outcome.pending":  "Preparing",
		"last.outcome.ok":       "Finished",
		"last.outcome.error":    "Error",
		"last.outcome.timeout":  "No response from Steam",
		"last.outcome.canceled": "Canceled",
		"last.outcome.unknown":  "Unknown origin",
		"last.empty":            "No session has been recorded yet.",
		"last.hint":             "You can repeat the launch or restore SpaceWar.",
		"last.setupMissing":     "Install the command from the guide to play again.",

		"guide.title":               "Launch guide",
		"guide.intro":               "Better RPT does not close Steam. Install the command in SpaceWar once, then launch the game from here.",
		"guide.command":             "SpaceWar command",
		"guide.commandHint":         "Paste it exactly into Launch Options. It includes %command%.",
		"guide.prepareHint":         "Prepare guide saves SpaceWar's original value before you paste the command, so it can be restored later.",
		"guide.step1.title":         "Open SpaceWar in Steam",
		"guide.step1.body":          "Library → SpaceWar (480) → right click → Properties.",
		"guide.step2.title":         "Open Launch Options",
		"guide.step2.body":          "On the General tab, find Launch Options.",
		"guide.step3.title":         "Paste the command",
		"guide.step3.body":          "Replace the current contents with the command above. Better RPT keeps a copy of the original value first.",
		"guide.step4.title":         "Save and launch",
		"guide.step4.body":          "Close Properties and press Play in Steam, or use Play again in Better RPT.",
		"guide.step5.title":         "Accept Steam's prompt",
		"guide.step5.body":          "Steam may show “Allow game launch?”. Accept it so the helper can start the game with the session parameters.",
		"guide.step6.title":         "Invite your friends",
		"guide.step6.body":          "With the game open, open the overlay with Shift+Tab and invite them to Remote Play Together.",
		"guide.steamOpen":           "Steam is open. Paste the command from Steam's UI; Better RPT will not close it. If the status does not change, use “Check again” or close Steam once so it writes localconfig.",
		"guide.steamClosed":         "Steam is closed. You can install the command automatically.",
		"guide.autoUnavailable":     "Close Steam manually to use automatic installation. Better RPT will not close it.",
		"guide.installDone":         "Command installed. You can now play without closing Steam.",
		"guide.recordDone":          "Original value backed up. Paste the command into Steam.",
		"guide.copyDone":            "Command copied.",
		"guide.setupMissing":        "There is no copy of the original value yet.",
		"guide.setupGuide":          "Guide prepared; the command still needs to be pasted into Steam.",
		"guide.setupInstalled":      "The command is already installed.",
		"guide.setupRestored":       "SpaceWar is restored.",
		"guide.setupConflict":       "SpaceWar's configuration changed outside Better RPT. It will not be overwritten.",
		"guide.image.launchOptions": "SpaceWar Launch Options",
		"guide.image.command":       "Pasted command",
		"guide.image.popup":         "Steam confirmation prompt",
		"guide.image.invite":        "Remote Play Together overlay",
		"guide.image.placeholder":   "Illustration pending replacement with a real Steam capture.",
		"guide.originalValue":       "Original value",
		"guide.originalEmpty":       "The original value is empty.",
		"guide.restoreHint":         "To restore without closing Steam, paste the original value into Launch Options. You can also use Restore while Steam is closed.",
		"guide.manualRestore":       "Manual restore",

		"situation.prepared":       "Prepared",
		"situation.active":         "Active",
		"situation.finished":       "Finished",
		"situation.restorePending": "Restore pending",
		"situation.restored":       "Restored",
		"situation.orphan":         "Orphan configuration",
		"situation.conflict":       "Conflict",
		"situation.idle":           "Idle",

		"setup.title":     "SpaceWar integration",
		"setup.missing":   "Not installed",
		"setup.guide":     "Guide prepared",
		"setup.installed": "Installed",
		"setup.restored":  "Restored",
		"setup.conflict":  "Conflict",

		"pending.title":       "Previous configuration",
		"pending.body":        "SpaceWar still has a configuration from an earlier session for %s. It is not the last session; you can restore it or open the guide.",
		"pending.orphanTitle": "Orphan configuration",
		"pending.orphanBody":  "SpaceWar points at Better RPT, but there is no coherent session or backup. It will not be touched; open the guide.",
		"pending.noBackup":    "There is no backup; it cannot be restored automatically.",
		"pending.restore":     "Restore previous configuration",
		"pending.guide":       "See how to restore",

		"restore.reason.steamRunning":  "Steam is open. Close it manually or restore from the guide; Better RPT will not close it.",
		"restore.reason.sessionActive": "The session is still active. Close the game before restoring.",
		"restore.reason.conflict":      "The current configuration does not match the backup. It will not be overwritten.",
		"restore.reason.noBackup":      "There is no associated backup.",
		"restore.reason.notNeeded":     "There is nothing to restore.",

		"error.title":           "The operation could not be completed",
		"error.body":            "Review the technical detail and try again.",
		"error.detail":          "Technical detail",
		"error.refresh":         "The Steam library could not be read.",
		"error.session":         "The session could not be started or queried.",
		"error.restoreConflict": "The launch option changed outside Better RPT. It was not overwritten.",
		"error.noBackup":        "There is no backup associated with this session.",
		"error.steamMissing":    "No native Steam installation was found.",
		"error.spacewarMissing": "SpaceWar (480) is not installed.",
		"error.setupRequired":   "Install the command from the guide before playing.",
		"error.setupConflict":   "SpaceWar's configuration changed outside Better RPT. It was not overwritten.",
		"error.replay":          "The last session could not be repeated.",

		"dialog.unknown.title":       "Unknown compatibility",
		"dialog.unknown.body":        "This game uses Proton and its Remote Play Together compatibility has not been verified. Try anyway?",
		"dialog.restore.title":       "Restore SpaceWar",
		"dialog.restore.body":        "SpaceWar's original launch option will be restored. Continue?",
		"dialog.cancelRestore.title": "Cancel the session",
		"dialog.cancelRestore.body":  "The launch option is already installed. Cancelling will restore SpaceWar. Continue?",
		"dialog.addSpacewar.title":   "Add SpaceWar",
		"dialog.addSpacewar.body":    "Steam will open so you can install SpaceWar (480), the Remote Play Together donor.",

		"restore.done.title": "SpaceWar restored",
		"restore.done.body":  "SpaceWar's launch configuration is back to its original state.",

		"common.continue": "Continue",
		"common.cancel":   "Cancel",
		"common.close":    "Close",
		"common.ignore":   "Ignore",
		"common.unknown":  "Unknown",
		"common.yes":      "Yes",
		"common.no":       "No",

		"language.es": "Español",
		"language.en": "English",

		"game.reason.launcher":  "It may require an external launcher.",
		"game.reason.anticheat": "It may use anti-cheat and not be compatible.",
		"primary.select":        "Select a game",

		"error.generic":         "An error occurred.",
		"error.root":            "Better RPT cannot run as root.",
		"error.instance":        "Better RPT is already open.",
		"error.config":          "The configuration could not be read.",
		"error.paths":           "The user folders could not be resolved.",
		"error.unsupported":     "This Steam installation is not supported.",
		"helper.notImplemented": "Session execution is not implemented yet.",
	},
}

// Supported reports whether lang is one of the bundled languages. It accepts
// only the language code, not a full locale.
func Supported(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case string(Spanish), string(English):
		return true
	default:
		return false
	}
}

// Normalize maps English locales to English and everything else to Spanish.
func Normalize(lang string) Language {
	if strings.EqualFold(strings.TrimSpace(lang), string(English)) {
		return English
	}
	return Spanish
}

// Detect reads the usual POSIX locale variables and falls back to Spanish.
// LC_ALL takes precedence over LC_MESSAGES, which takes precedence over LANG.
func Detect() Language {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		switch {
		case strings.HasPrefix(value, string(English)):
			return English
		case strings.HasPrefix(value, string(Spanish)):
			return Spanish
		}
	}
	return Spanish
}

// Translator resolves copy for one language.
type Translator struct {
	lang Language
}

// New returns a translator for lang. Unknown languages fall back to Spanish.
func New(lang string) *Translator {
	return &Translator{lang: Normalize(lang)}
}

// Language returns the resolved language.
func (t *Translator) Language() Language {
	return t.lang
}

// T returns the requested key, falling back to Spanish and then to the key.
func (t *Translator) T(key string) string {
	if value, ok := catalogs[t.lang][key]; ok {
		return value
	}
	if value, ok := catalogs[Spanish][key]; ok {
		return value
	}
	return key
}
