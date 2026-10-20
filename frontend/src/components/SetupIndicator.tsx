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
}: {
  lang: Language;
  status?: SetupStatus;
}) {
  if (!status) return null;

  return (
    <span className="setup-indicator" data-tone={setupTone(status)}>
      <span className="setup-indicator-label">{translate(lang, "setup.title")}</span>
      <span className="setup-indicator-value">
        {translate(lang, SETUP_KEY[status])}
      </span>
    </span>
  );
}

function setupTone(status: SetupStatus): "mint" | "amber" | "coral" {
  if (status === "installed") return "mint";
  if (status === "conflict") return "coral";
  return "amber";
}
