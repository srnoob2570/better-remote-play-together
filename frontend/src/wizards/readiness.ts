import type {
  DiagnosticCheck,
  DiagnosticReport,
  SetupSnapshot,
  SetupStatus,
  Situation,
} from "../bridge/types";
import type { MessageKey } from "../i18n";

export type SetupStepId =
  | "welcome"
  | "steam"
  | "helper"
  | "dispatcher"
  | "dependencies"
  | "verify";

export type SetupStepStatus = "done" | "pending" | "blocked";

export interface SetupStep {
  id: SetupStepId;
  status: SetupStepStatus;
  reasonKey?: MessageKey;
}

export interface SetupAssessment {
  format: "appimage" | "deb" | "unknown";
  steps: SetupStep[];
  complete: boolean;
  blocked: boolean;
}

export const SETUP_STEP_LABEL: Record<SetupStepId, MessageKey> = {
  welcome: "wizard.step.welcome",
  steam: "wizard.step.steam",
  helper: "wizard.step.helper",
  dispatcher: "wizard.step.dispatcher",
  dependencies: "wizard.step.dependencies",
  verify: "wizard.step.verify",
};

export const SETUP_STEP_COUNT = Object.keys(SETUP_STEP_LABEL).length;

export function checkById(
  report: DiagnosticReport | null,
  id: string,
): DiagnosticCheck | undefined {
  return report?.checks.find((check) => check.id === id);
}

export function assessSetup(
  report: DiagnosticReport | null,
  setup?: SetupSnapshot,
  situation?: Situation,
): SetupAssessment {
  const format =
    report?.format === "deb"
      ? "deb"
      : report?.format === "appimage"
        ? "appimage"
        : "unknown";
  const activeSession = situation?.kind === "active";
  const steps: SetupStep[] = [];

  steps.push({ id: "welcome", status: "done" });

  const steam = checkById(report, "steam.install");
  const spacewar = checkById(report, "steam.spacewar");
  if (!report) {
    steps.push({ id: "steam", status: "pending" });
  } else if (steam?.status === "fail" || steam?.status === "skip") {
    steps.push({ id: "steam", status: "blocked" });
  } else if (spacewar?.status === "ok") {
    steps.push({ id: "steam", status: "done" });
  } else {
    steps.push({
      id: "steam",
      status: "pending",
      reasonKey: "wizard.steam.recheckHint",
    });
  }

  const helper = checkById(report, "app.helper");
  if (activeSession) {
    steps.push({
      id: "helper",
      status: "blocked",
      reasonKey: "wizard.helper.active",
    });
  } else if (
    helper?.evidence.includes("source=missing") &&
    helper.evidence.includes("exists=false")
  ) {
    steps.push({
      id: "helper",
      status: "blocked",
      reasonKey: "wizard.helper.sourceMissing",
    });
  } else if (helper?.status === "ok") {
    steps.push({ id: "helper", status: "done" });
  } else {
    steps.push({ id: "helper", status: "pending" });
  }

  const dispatcher = checkById(report, "steam.dispatcher");
  if (setup?.status === "conflict" || dispatcher?.status === "fail") {
    steps.push({
      id: "dispatcher",
      status: "blocked",
      reasonKey: "wizard.dispatcher.conflict",
    });
  } else if (setup?.status === "installed" && dispatcher?.status === "ok") {
    steps.push({ id: "dispatcher", status: "done" });
  } else {
    steps.push({ id: "dispatcher", status: "pending" });
  }

  const dependencyChecks = dependencyChecksFor(report);
  const dependencyFailure = dependencyChecks.find(
    (check) => check.status === "fail" || check.status === "warn",
  );
  if (format === "deb") {
    steps.push({ id: "dependencies", status: "done" });
  } else if (dependencyFailure) {
    steps.push({ id: "dependencies", status: "blocked" });
  } else {
    steps.push({
      id: "dependencies",
      status: report ? "done" : "pending",
    });
  }

  const required = steps.filter((step) => step.id !== "verify");
  const blocked = required.some((step) => step.status === "blocked");
  const pending = required.some((step) => step.status === "pending");
  steps.push({
    id: "verify",
    status: blocked ? "blocked" : pending ? "pending" : "done",
  });

  return {
    format,
    steps,
    complete: !blocked && !pending,
    blocked,
  };
}

export function firstPendingStep(steps: SetupStep[]): number {
  const index = steps.findIndex(
    (step) => step.id !== "welcome" && step.status !== "done",
  );
  return index < 0 ? steps.length - 1 : index;
}

export function setupAutoOpen(status: SetupStatus | undefined): boolean {
  return status !== "installed" && status !== "restored";
}

export function dependencyChecksFor(
  report: DiagnosticReport | null,
): DiagnosticCheck[] {
  if (!report) return [];
  const ids = [
    "deps.elf",
    "appimage.fuse",
    "system.glibc",
    "appimage.mount",
    "appimage.mac",
  ];
  return ids
    .map((id) => checkById(report, id))
    .filter((check): check is DiagnosticCheck => Boolean(check));
}

export function dependencyCommand(id: string): string | null {
  switch (id) {
    case "deps.elf":
      return "sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0";
    case "appimage.fuse":
      return "sudo apt install fuse3";
    default:
      return null;
  }
}

export function appImageFallback(version: string): string {
  const suffix = version ? `-${version}` : "";
  return `APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT${suffix}-x86_64.AppImage`;
}
