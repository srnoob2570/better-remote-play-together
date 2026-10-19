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
		"selection.actionHint": "SpaceWar se restaura al terminar.",
		"selection.donorHint":  "SpaceWar debe estar instalado para iniciar la sesión.",

		"action.clear":         "Limpiar",
		"action.resetFilters":  "Quitar filtros",
		"action.clearSearch":   "Limpiar búsqueda",
		"action.refresh":       "Actualizar",
		"action.refreshing":    "Actualizando…",
		"action.prepare":       "Preparar y jugar",
		"action.cancel":        "Cancelar",
		"action.cancelRestore": "Cancelar y restaurar",
		"action.restore":       "Restaurar SpaceWar",
		"action.back":          "Volver a la biblioteca",
		"action.retry":         "Reintentar",
		"action.details":       "Detalles",
		"action.addSpacewar":   "Añadir SpaceWar",
		"action.openSteam":     "Abrir Steam",

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
		"donor.steamClosed":     "Steam está cerrado",
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
		"prep.closeSteam":    "Cerrar Steam",
		"prep.backup":        "Guardar copia de localconfig",
		"prep.launchOption":  "Instalar la opción de lanzamiento",
		"prep.startDonor":    "Iniciar Steam y SpaceWar",
		"prep.waitHelper":    "Esperar al helper y al juego",
		"prep.cancelHint":    "Puedes cancelar antes de que empiece la sesión.",

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

		"ended.title":       "Sesión terminada",
		"ended.duration":    "Duración",
		"ended.restoreHint": "Restaura SpaceWar para dejar su configuración como estaba.",

		"stale.title": "Hay una configuración pendiente",
		"stale.body":  "Better RPT encontró una opción de lanzamiento de una sesión anterior. Puedes restaurarla ahora; si no hay copia, no se sobrescribirá nada.",

		"error.title":           "No se pudo completar la operación",
		"error.body":            "Revisa el detalle técnico y vuelve a intentarlo.",
		"error.detail":          "Detalle técnico",
		"error.refresh":         "No se pudo leer la biblioteca de Steam.",
		"error.session":         "No se pudo iniciar o consultar la sesión.",
		"error.restoreConflict": "La opción de lanzamiento cambió fuera de Better RPT. No se sobrescribió.",
		"error.noBackup":        "No hay una copia asociada a esta sesión.",
		"error.steamMissing":    "No se encontró una instalación nativa de Steam.",
		"error.spacewarMissing": "SpaceWar (480) no está instalado.",

		"dialog.closeSteam.title":    "Cerrar Steam",
		"dialog.closeSteam.body":     "Better RPT necesita cerrar Steam para modificar la opción de lanzamiento. ¿Cerrar Steam y continuar?",
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
		"selection.actionHint": "SpaceWar is restored when you finish.",
		"selection.donorHint":  "SpaceWar must be installed before a session can start.",

		"action.clear":         "Clear",
		"action.resetFilters":  "Clear filters",
		"action.clearSearch":   "Clear search",
		"action.refresh":       "Refresh",
		"action.refreshing":    "Refreshing…",
		"action.prepare":       "Prepare and play",
		"action.cancel":        "Cancel",
		"action.cancelRestore": "Cancel and restore",
		"action.restore":       "Restore SpaceWar",
		"action.back":          "Back to library",
		"action.retry":         "Retry",
		"action.details":       "Details",
		"action.addSpacewar":   "Add SpaceWar",
		"action.openSteam":     "Open Steam",

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
		"donor.steamClosed":     "Steam is closed",
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
		"prep.closeSteam":    "Close Steam",
		"prep.backup":        "Back up localconfig",
		"prep.launchOption":  "Install the launch option",
		"prep.startDonor":    "Start Steam and SpaceWar",
		"prep.waitHelper":    "Wait for the helper and game",
		"prep.cancelHint":    "You can cancel before the session starts.",

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

		"ended.title":       "Session ended",
		"ended.duration":    "Duration",
		"ended.restoreHint": "Restore SpaceWar to leave its configuration as it was.",

		"stale.title": "A configuration is pending",
		"stale.body":  "Better RPT found a launch option from an earlier session. You can restore it now; without a backup, nothing will be overwritten.",

		"error.title":           "The operation could not be completed",
		"error.body":            "Review the technical detail and try again.",
		"error.detail":          "Technical detail",
		"error.refresh":         "The Steam library could not be read.",
		"error.session":         "The session could not be started or queried.",
		"error.restoreConflict": "The launch option changed outside Better RPT. It was not overwritten.",
		"error.noBackup":        "There is no backup associated with this session.",
		"error.steamMissing":    "No native Steam installation was found.",
		"error.spacewarMissing": "SpaceWar (480) is not installed.",

		"dialog.closeSteam.title":    "Close Steam",
		"dialog.closeSteam.body":     "Better RPT needs to close Steam to change the launch option. Close Steam and continue?",
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
