import type { SetupStatus } from "../bridge/types";
import { translate, type Language, type MessageKey } from "../i18n";

const SETUP_KEY: Record<SetupStatus, MessageKey> = {
  missing: "setup.missing",
  guide: "setup.guide",
  installed: "setup.installed",
  restored: "setup.restored",
  conflict: "setup.conflict",
};

export function SetupIndicator({
  lang,
  status,
  ready,
  onOpen,
}: {
  lang: Language;
  status?: SetupStatus;
  ready: boolean;
  onOpen: () => void;
}) {
  if (!ready) return null;
  const effective = status ?? "missing";
  return (
    <button
      type="button"
      className="setup-indicator"
      data-tone={setupTone(effective)}
      onClick={onOpen}
    >
      <span className="setup-indicator-label">{translate(lang, "setup.title")}</span>
      <span className="setup-indicator-value">
        {translate(lang, SETUP_KEY[effective])}
      </span>
    </button>
  );
}

function setupTone(status: SetupStatus): "mint" | "amber" | "coral" {
  if (status === "installed") return "mint";
  if (status === "conflict") return "coral";
  return "amber";
}
