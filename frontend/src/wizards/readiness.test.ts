import { describe, expect, it } from "vitest";

import type {
  DiagnosticCheck,
  DiagnosticReport,
  SetupSnapshot,
  Situation,
} from "../bridge/types";
import { buildRepairIssues } from "./repair";
import { assessSetup, setupAutoOpen } from "./readiness";

function check(
  id: string,
  status: DiagnosticCheck["status"] = "ok",
  message = "",
  evidence: string[] = [],
): DiagnosticCheck {
  return { id, status, title: id, message, evidence };
}

function report(format: string, checks: DiagnosticCheck[]): DiagnosticReport {
  return {
    schema_version: 1,
    generated_at: "2026-10-20T12:00:00Z",
    version: "0.2.0",
    format,
    language: "es",
    summary: { verdict: "warn", counts: { ok: 0, warn: 0, fail: 0, skip: 0 }, total: checks.length },
    checks,
  };
}

function situation(overrides: Partial<Situation> = {}): Situation {
  return {
    kind: "idle",
    setup: "missing",
    restore: "not_needed",
    steamRunning: false,
    ...overrides,
  };
}

const setupInstalled: SetupSnapshot = { status: "installed", hasBackup: true };

describe("setup assessment", () => {
  it("treats a fresh install as incomplete and starts at Steam", () => {
    const assessment = assessSetup(null);
    expect(assessment.complete).toBe(false);
    expect(assessment.steps.find((step) => step.id === "helper")?.status).toBe("pending");
    expect(assessment.steps.find((step) => step.id === "dispatcher")?.status).toBe("pending");
    expect(setupAutoOpen(undefined)).toBe(true);
    expect(setupAutoOpen("missing")).toBe(true);
    expect(setupAutoOpen("installed")).toBe(false);
    expect(setupAutoOpen("restored")).toBe(false);
  });

  it("blocks helper replacement while a session is active", () => {
    const assessment = assessSetup(
      report("deb", [
        check("steam.install"),
        check("steam.spacewar"),
        check("app.helper", "warn", "helper missing"),
        check("steam.dispatcher", "warn", "guide"),
      ]),
      { status: "guide", hasBackup: true },
      situation({ kind: "active", setup: "guide", restore: "session_active" }),
    );
    expect(assessment.steps.find((step) => step.id === "helper")?.status).toBe("blocked");
  });

  it("blocks the helper step when the artifact has no packaged source", () => {
    const assessment = assessSetup(
      report("deb", [
        check("steam.install"),
        check("steam.spacewar"),
        check("app.helper", "warn", "helper missing", [
          "source=missing",
          "exists=false",
        ]),
        check("steam.dispatcher", "warn", "guide"),
      ]),
      { status: "missing", hasBackup: false },
    );
    const helper = assessment.steps.find((step) => step.id === "helper");
    expect(helper?.status).toBe("blocked");
    expect(helper?.reasonKey).toBe("wizard.helper.sourceMissing");
  });

  it("marks a verified .deb setup complete without AppImage dependencies", () => {
    const assessment = assessSetup(
      report("deb", [
        check("steam.install"),
        check("steam.spacewar"),
        check("app.helper"),
        check("steam.dispatcher"),
        check("appimage.fuse", "skip"),
      ]),
      setupInstalled,
    );
    expect(assessment.complete).toBe(true);
    expect(assessment.steps.find((step) => step.id === "dependencies")?.status).toBe("done");
  });

  it("blocks an AppImage when a dependency fails", () => {
    const assessment = assessSetup(
      report("appimage", [
        check("steam.install"),
        check("steam.spacewar"),
        check("app.helper"),
        check("steam.dispatcher"),
        check("appimage.fuse", "fail", "FUSE missing"),
      ]),
      setupInstalled,
    );
    expect(assessment.steps.find((step) => step.id === "dependencies")?.status).toBe("blocked");
  });
});

describe("repair issue mapping", () => {
  it("maps helper, dispatcher, Steam and dependency checks to safe actions", () => {
    const issues = buildRepairIssues(
      report("appimage", [
        check("app.helper", "warn", "helper missing"),
        check("steam.dispatcher", "fail", "conflict"),
        check("steam.spacewar", "fail", "missing"),
        check("steam.install", "fail", "steam missing"),
        check("appimage.fuse", "fail", "fuse missing"),
      ]),
      { status: "conflict", hasBackup: true },
      situation({ kind: "conflict", setup: "conflict", restore: "conflict" }),
    );
    const actions = Object.fromEntries(issues.map((issue) => [issue.checkId, issue.action]));
    expect(actions["app.helper"]).toBe("helper");
    expect(actions["steam.dispatcher"]).toBe("dispatcher");
    expect(actions["steam.spacewar"]).toBe("spacewar");
    expect(actions["steam.install"]).toBe("steam");
    expect(actions["appimage.fuse"]).toBe("deps");
  });

  it("offers restore only when the situation says it is available", () => {
    const available = buildRepairIssues(
      report("deb", []),
      setupInstalled,
      situation({
        kind: "restore_pending",
        setup: "installed",
        restore: "available",
        token: "0123456789abcdef0123456789abcdef",
      }),
    );
    expect(available.find((issue) => issue.checkId === "session.state")?.action).toBe("restore");

    const steamRunning = buildRepairIssues(
      report("deb", []),
      setupInstalled,
      situation({
        kind: "restore_pending",
        setup: "installed",
        restore: "steam_running",
      }),
    );
    expect(steamRunning.find((issue) => issue.checkId === "session.state")?.action).toBe("none");
  });
});
