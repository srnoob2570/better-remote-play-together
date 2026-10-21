import type { Game, GameStatus } from "../bridge/types";
import { translate, type Language, type MessageKey } from "../i18n";

export const GAME_STATUS_KEY: Record<GameStatus, MessageKey> = {
  ready: "status.ready",
  unknown: "status.unknown",
  incompatible: "status.incompatible",
  missing: "status.missing",
  error: "status.error",
  active: "status.active",
};

export const GAME_STATUS_TONE: Record<GameStatus, string> = {
  ready: "success",
  unknown: "warning",
  incompatible: "error",
  missing: "neutral",
  error: "error",
  active: "info",
};

export function gameKindLabel(lang: Language, kind: Game["kind"]): string {
  if (kind === "native") return translate(lang, "kind.native");
  if (kind === "proton") return translate(lang, "kind.proton");
  return translate(lang, "common.unknown");
}
