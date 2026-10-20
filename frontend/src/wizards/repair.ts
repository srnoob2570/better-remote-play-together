import type {
  DiagnosticCheck,
  DiagnosticReport,
  DiagnosticStatus,
  RestoreState,
  SetupSnapshot,
  Situation,
} from "../bridge/types";
import type { MessageKey } from "../i18n";
import { dependencyCommand } from "./readiness";

export type RepairActionKind =
  | "helper"
  | "spacewar"
  | "steam"
  | "dispatcher"
  | "deps"
  | "restore"
  | "recover"
  | "cancel"
  | "none";

export interface RepairIssue {
  key: string;
  checkId: string;
  status: DiagnosticStatus;
  title?: string;
  titleKey?: MessageKey;
  message?: string;
  messageKey?: MessageKey;
  evidence: string[];
  nextStep?: string;
  docs?: string[];
  action: RepairActionKind;
  actionLabelKey: MessageKey;
  command?: string;
  disabledReasonKey?: MessageKey;
  restoreReason?: RestoreState;
  hintKey?: MessageKey;
}

const ACTION_LABEL: Record<RepairActionKind, MessageKey> = {
  helper: "wizard.repair.action.helper",
  spacewar: "wizard.repair.action.spacewar",
  steam: "wizard.repair.action.steam",
  dispatcher: "wizard.repair.action.dispatcher",
  deps: "wizard.repair.action.deps",
  restore: "wizard.repair.action.restore",
  recover: "wizard.repair.action.recover",
  cancel: "wizard.repair.action.cancel",
  none: "wizard.repair.action.none",
};

export function buildRepairIssues(
  report: DiagnosticReport | null,
  setup?: SetupSnapshot,
  situation?: Situation,
): RepairIssue[] {
  const issues: RepairIssue[] = [];
  const checks = report?.checks.filter(
    (check) => check.status === "fail" || check.status === "warn",
  ) ?? [];

  for (const check of checks) {
    const issue = issueForCheck(check, situation);
    if (issue) issues.push(issue);
  }

  if (situation) {
    const sessionIssue = issueForSituation(situation);
    if (sessionIssue) {
      const index = issues.findIndex((issue) => issue.checkId === "session.state");
      if (index >= 0) issues.splice(index, 1, sessionIssue);
      else issues.push(sessionIssue);
    }
  }

  if (setup?.status === "conflict" && !issues.some((issue) => issue.checkId === "steam.dispatcher")) {
    issues.push({
      key: "setup-conflict",
      checkId: "steam.dispatcher",
      status: "fail",
      titleKey: "guide.title",
      messageKey: "guide.setupConflict",
      evidence: ["status=conflict"],
      docs: ["docs/compatibility.md"],
      action: "dispatcher",
      actionLabelKey: ACTION_LABEL.dispatcher,
      hintKey: "wizard.dispatcher.conflict",
    });
  }

  return dedupe(issues);
}

function issueForCheck(
  check: DiagnosticCheck,
  situation?: Situation,
): RepairIssue | null {
  switch (check.id) {
    case "app.helper": {
      const sourceMissing =
        check.evidence.includes("source=missing") &&
        check.evidence.includes("exists=false");
      const disabled =
        situation?.kind === "active"
          ? "wizard.repair.helperActive"
          : sourceMissing
            ? "wizard.helper.sourceMissing"
            : undefined;
      return baseIssue(check, "helper", disabled);
    }
    case "steam.spacewar":
      return baseIssue(check, "spacewar", undefined, "wizard.repair.spacewarHint");
    case "steam.install":
      return baseIssue(check, "steam", undefined, "wizard.repair.steamMissing");
    case "steam.dispatcher":
      return baseIssue(check, "dispatcher", undefined, "wizard.repair.dispatcherHint");
    case "appimage.fuse":
    case "deps.elf":
    case "appimage.mount":
    case "appimage.mac":
      return {
        ...baseIssue(check, "deps", undefined, "wizard.repair.depsHint"),
        command: dependencyCommand(check.id) ?? undefined,
      };
    case "system.glibc":
      return baseIssue(check, "deps", undefined, "wizard.repair.depsHint");
    case "steam.permissions":
      return baseIssue(check, "none", undefined, "wizard.repair.permissionsHint");
    case "session.state": {
      if (check.status !== "warn") return null;
      const stale = check.evidence.some((line) => line.includes("alive=false"));
      const issue = baseIssue(check, stale ? "recover" : "none");
      if (stale) issue.hintKey = "wizard.repair.recoverHint";
      return issue;
    }
    case "steam.backups":
      return baseIssue(check, "none", undefined, "wizard.repair.backupHint");
    default:
      return baseIssue(check, "none");
  }
}

function issueForSituation(situation: Situation): RepairIssue | null {
  const base = {
    checkId: "session.state",
    evidence: [
      `kind=${situation.kind}`,
      `setup=${situation.setup}`,
      `restore=${situation.restore}`,
      `steam_running=${situation.steamRunning}`,
    ],
    docs: ["docs/architecture.md"],
  };
  switch (situation.kind) {
    case "active": {
      const preparing = situation.state?.phase === "preparing";
      return {
        ...base,
        key: "session-active",
        status: "warn",
        titleKey: "session.title",
        messageKey: "wizard.repair.sessionActive",
        action: preparing ? "cancel" : "none",
        actionLabelKey: preparing
          ? ACTION_LABEL.cancel
          : ACTION_LABEL.none,
        hintKey: preparing
          ? "wizard.repair.cancelHint"
          : "wizard.repair.sessionActiveHint",
        disabledReasonKey: preparing
          ? undefined
          : "wizard.repair.sessionActiveHint",
      };
    }
    case "restore_pending":
      return {
        ...base,
        key: "session-restore",
        status: "warn",
        titleKey: "pending.title",
        messageKey: "wizard.repair.sessionRestore",
        action: situation.restore === "available" ? "restore" : "none",
        actionLabelKey:
          situation.restore === "available"
            ? ACTION_LABEL.restore
            : ACTION_LABEL.none,
        restoreReason: situation.restore,
        hintKey: "wizard.repair.restoreHint",
      };
    case "orphan":
      return {
        ...base,
        key: "session-orphan",
        status: "warn",
        titleKey: "pending.orphanTitle",
        messageKey: "wizard.repair.sessionOrphan",
        action:
          situation.restore === "available" && situation.token
            ? "restore"
            : "none",
        actionLabelKey:
          situation.restore === "available" && situation.token
            ? ACTION_LABEL.restore
            : ACTION_LABEL.none,
        restoreReason:
          situation.restore === "available" && !situation.token
            ? "no_backup"
            : situation.restore,
        hintKey: "wizard.repair.restoreHint",
      };
    case "conflict":
      return {
        ...base,
        key: "session-conflict",
        status: "fail",
        titleKey: "guide.title",
        messageKey: "guide.setupConflict",
        action: "dispatcher",
        actionLabelKey: ACTION_LABEL.dispatcher,
        hintKey: "wizard.dispatcher.conflict",
      };
    default:
      return null;
  }
}

function baseIssue(
  check: DiagnosticCheck,
  action: RepairActionKind,
  disabledReasonKey?: MessageKey,
  hintKey?: MessageKey,
): RepairIssue {
  return {
    key: check.id,
    checkId: check.id,
    status: check.status,
    title: check.title,
    message: check.message,
    evidence: check.evidence,
    nextStep: check.nextStep,
    docs: check.docs,
    action,
    actionLabelKey: ACTION_LABEL[action],
    disabledReasonKey,
    hintKey,
  };
}

function dedupe(issues: RepairIssue[]): RepairIssue[] {
  const seen = new Set<string>();
  return issues.filter((issue) => {
    const key = `${issue.key}:${issue.checkId}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}
