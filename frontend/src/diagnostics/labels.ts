import type { DiagnosticStatus } from "../bridge/types";
import type { MessageKey } from "../i18n";

export function diagnosticStatusLabel(
  t: (key: MessageKey) => string,
  status: DiagnosticStatus,
): string {
  switch (status) {
    case "ok":
      return t("diag.status.ok");
    case "warn":
      return t("diag.status.warn");
    case "fail":
      return t("diag.status.fail");
    case "skip":
      return t("diag.status.skip");
  }
}
