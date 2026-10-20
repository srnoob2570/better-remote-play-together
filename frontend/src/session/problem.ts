import type { Problem } from "../bridge/types";
import { translate, type Language, type MessageKey } from "../i18n";

const CODE_KEYS: Record<string, MessageKey> = {
  setup_missing: "error.setupRequired",
  setup_conflict: "error.setupConflict",
  setup_invalid: "error.setupConflict",
  steam_running: "restore.reason.steamRunning",
  spacewar_missing: "error.spacewarMissing",
  steam_not_found: "error.steamMissing",
  steam_guard_missing: "error.steamMissing",
  account_not_found: "error.steamMissing",
  account_ambiguous: "error.steamMissing",
  session_active: "error.session",
  session_busy: "error.session",
  invalid_session: "error.session",
  local_config_changed: "error.restoreConflict",
  local_config_unsafe: "error.restoreConflict",
  backup_missing: "error.noBackup",
  backup_corrupt: "error.noBackup",
  session_canceled: "error.session",
  timeout: "error.session",
  library_unreadable: "error.refresh",
  helper_missing: "error.session",
  helper_source_missing: "error.session",
  helper_source_invalid: "error.session",
  no_launch_option: "error.session",
  invalid_launch_spec: "error.session",
  not_game: "error.session",
  info_state: "error.session",
  proton_not_found: "error.session",
  internal: "error.generic",
};

export function problemKey(code: string): MessageKey | null {
  return CODE_KEYS[code] ?? null;
}

export function problemText(lang: Language, problem: Problem): string {
  const key = problemKey(problem.code);
  if (key) return translate(lang, key);
  return problem.message || translate(lang, "error.generic");
}
