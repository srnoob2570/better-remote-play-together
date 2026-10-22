import { setTransport } from "@wailsio/runtime";

import type {
  Active,
  Catalog,
  DiagnosticReport,
  DiagnosticsResult,
  Game,
  LastSession,
  Problem,
  Recovery,
  SessionState,
  Setup,
  SetupSnapshot,
  Situation,
  Snapshot,
  UpdateCheck,
} from "../bridge/types";

type Scenario =
  | "library"
  | "many-games"
  | "loading"
  | "empty"
  | "error"
  | "preparing"
  | "launching"
  | "active"
  | "ended"
  | "stale"
  | "donor-missing"
  | "incompatible"
  | "details"
  | "dialog-unknown"
  | "session-error"
  | "setup-missing"
  | "update-available"
  | "update-development"
  | "start-pending";

const NOW = new Date("2026-10-20T12:00:00Z");

const games: Game[] = [
  game(480, "SpaceWar", "native", "ready"),
  game(553850, "HELLDIVERS™ 2", "proton", "unknown", "Proton 11.0"),
  game(1599600, "PlateUp!", "proton", "unknown", "Proton 9.0"),
  game(892970, "Valheim", "native", "ready"),
  game(883710, "Resident Evil 2", "proton", "unknown", "Proton 10.0"),
  game(1245620, "ELDEN RING", "proton", "incompatible", "Proton 9.0", {
    code: "anti_cheat",
    message: "Easy Anti-Cheat no es compatible con Remote Play Together.",
  }),
];

const manyGames: Game[] = Array.from({ length: 80 }, (_, index) =>
  game(900000 + index, `Test Game ${String(index + 1).padStart(2, "0")}`, "native", "ready"),
);

function game(
  appId: number,
  name: string,
  kind: Game["kind"],
  status: Game["status"],
  protonName?: string,
  problem?: Problem,
): Game {
  return {
    appId,
    name,
    installDir: `/steam/steamapps/common/${name.replaceAll(/[^A-Za-z0-9]+/g, "")}`,
    libraryPath: "/steam",
    sizeBytes: 10_000_000_000,
    kind,
    status,
    protonName,
    problem,
  };
}

function token(): string {
  return "0123456789abcdef0123456789abcdef";
}

function state(phase: SessionState["phase"], targetAppId = 553850): SessionState {
  const started = new Date(NOW.getTime() - 65_000).toISOString();
  return {
    token: token(),
    donorAppId: 480,
    targetAppId,
    phase,
    outcome: phase === "ended" ? "ok" : undefined,
    startedAt: phase === "idle" ? undefined : started,
    endedAt: phase === "ended" ? NOW.toISOString() : undefined,
    createdAt: started,
    updatedAt: NOW.toISOString(),
  };
}

function last(overrides: Partial<LastSession> = {}): LastSession {
  return {
    appId: 553850,
    name: "HELLDIVERS™ 2",
    token: token(),
    source: "better-rpt",
    startedAt: new Date(NOW.getTime() - 3_725_000).toISOString(),
    endedAt: NOW.toISOString(),
    outcome: "ok",
    ...overrides,
  };
}

function catalog(overrides: Partial<Catalog> = {}): Catalog {
  return { games, warnings: [], ...overrides };
}

function updateCheck(available: boolean): UpdateCheck {
  return {
    currentVersion: "0.2.0",
    latestVersion: available ? "0.2.1" : "0.2.0",
    tag: available ? "v0.2.1" : "v0.2.0",
    releaseUrl: `https://github.com/srnoob2570/better-remote-play-together/releases/tag/${
      available ? "v0.2.1" : "v0.2.0"
    }`,
    available,
    canInstall: available,
    installKind: "appimage",
    development: false,
    checkedAt: NOW.toISOString(),
    cached: true,
  };
}

function diagnostics(lang: "es" | "en"): DiagnosticsResult {
  const en = lang === "en";
  const report: DiagnosticReport = {
    schema_version: 1,
    generated_at: NOW.toISOString(),
    version: "0.2.0",
    format: "appimage",
    language: lang,
    summary: {
      verdict: "warn",
      counts: { ok: 2, warn: 1, fail: 0, skip: 1 },
      total: 4,
    },
    checks: [
      {
        id: "system.version",
        status: "ok",
        title: en ? "Better RPT version" : "Versión de Better RPT",
        message: "0.2.0",
        evidence: ["better-rpt 0.2.0"],
      },
      {
        id: "appimage.fuse",
        status: "warn",
        title: "FUSE",
        message: en
          ? "FUSE was not found: fusermount3 and /dev/fuse are missing."
          : "No se encontró FUSE: faltan fusermount3 y /dev/fuse.",
        evidence: ["fusermount3: not found", "/dev/fuse: missing"],
        nextStep: en
          ? "Install fuse3 and reopen the AppImage, or use --appimage-extract-and-run."
          : "Instala fuse3 y vuelve a abrir el AppImage, o usa --appimage-extract-and-run.",
        docs: ["docs/appimage.md"],
      },
      {
        id: "app.config",
        status: "ok",
        title: en ? "App configuration" : "Configuración de la app",
        message: en ? "config.json is valid." : "config.json es válido.",
        evidence: ["~/.config/better-rpt/config.json"],
      },
      {
        id: "limits.rpt",
        status: "skip",
        title: en ? "Diagnostics limits" : "Límites del diagnóstico",
        message: en
          ? "NAT, anti-cheat and per-game Remote Play compatibility are not diagnosed."
          : "No se diagnostican NAT, anti-cheat ni la compatibilidad de un juego concreto con Remote Play.",
        evidence: [],
        nextStep: en ? "See the compatibility guide." : "Consulta la guía de compatibilidad.",
        docs: ["docs/compatibility.md"],
      },
    ],
  };
  return {
    report,
    markdown: `# Better RPT diagnostics\n\n${report.summary.verdict.toUpperCase()} · OK 2 · WARN 1 · FAIL 0 · SKIP 1\n`,
  };
}

function baseSetup(): SetupSnapshot {
  return {
    status: "installed",
    hasBackup: true,
    originalValue: "gamemoderun %command%",
  };
}

function baseSituation(overrides: Partial<Situation> = {}): Situation {
  return {
    kind: "idle",
    setup: "installed",
    restore: "not_needed",
    steamRunning: false,
    ...overrides,
  };
}

function recovery(overrides: Partial<Recovery> = {}): Recovery {
  return {
    helperAlive: false,
    situation: baseSituation(),
    setup: baseSetup(),
    expiredPlans: [],
    ...overrides,
  };
}

function scenarioData(name: Scenario, lang: "es" | "en") {
  let snapshotCatalog: Catalog | undefined = catalog();
  let recoveryValue = recovery();
  let sessionState: SessionState | undefined;
  let setup = baseSetup();
  let situation = baseSituation();
  let lastValue: LastSession | undefined;
  let updateValue = updateCheck(false);

  switch (name) {
    case "many-games":
      snapshotCatalog = catalog({ games: manyGames });
      break;
    case "loading":
      snapshotCatalog = undefined;
      recoveryValue = recovery({ situation: undefined, setup: undefined });
      break;
    case "empty":
      snapshotCatalog = catalog({ games: [] });
      break;
    case "setup-missing":
      recoveryValue = recovery({ setup: undefined, situation: undefined });
      setup = { status: "missing", hasBackup: false };
      situation = baseSituation({ setup: "missing" });
      break;
    case "preparing":
      sessionState = state("preparing");
      situation = baseSituation({ kind: "prepared", state: sessionState });
      recoveryValue = recovery({ state: sessionState, situation });
      break;
    case "launching":
      sessionState = state("launching");
      situation = baseSituation({ kind: "prepared", state: sessionState });
      recoveryValue = recovery({ state: sessionState, situation });
      break;
    case "active":
      sessionState = state("active");
      situation = baseSituation({ kind: "active", restore: "session_active", state: sessionState });
      recoveryValue = recovery({ state: sessionState, situation });
      break;
    case "ended":
      sessionState = state("ended");
      lastValue = last();
      situation = baseSituation({
        kind: "finished",
        restore: "available",
        token: sessionState.token,
        targetAppId: sessionState.targetAppId,
        state: sessionState,
        last: lastValue,
      });
      recoveryValue = recovery({ state: sessionState, last: lastValue, situation });
      break;
    case "stale":
      lastValue = last({ outcome: "pending" });
      situation = baseSituation({
        kind: "restore_pending",
        restore: "available",
        token: lastValue.token,
        targetAppId: lastValue.appId,
        last: lastValue,
      });
      recoveryValue = recovery({ last: lastValue, situation });
      break;
    case "donor-missing":
      snapshotCatalog = catalog({ games: games.filter((item) => item.appId !== 480) });
      break;
    case "incompatible":
      snapshotCatalog = catalog({
        games: [games[5], games[3], games[4], games[0]],
      });
      break;
    case "dialog-unknown":
      situation = baseSituation({ setup: "installed" });
      break;
    case "session-error":
      sessionState = {
        ...state("error"),
        error: {
          code: "setup_conflict",
          message: "La configuración cambió fuera de Better RPT.",
          detail: "localconfig.vdf no coincide con el backup.",
        },
      };
      situation = baseSituation({ kind: "conflict", restore: "conflict", state: sessionState });
      recoveryValue = recovery({ state: sessionState, situation });
      break;
    case "error":
      snapshotCatalog = undefined;
      recoveryValue = recovery({ situation: undefined, setup: undefined });
      break;
    case "update-available":
      updateValue = updateCheck(true);
      break;
    case "update-development":
      updateValue = {
        ...updateCheck(true),
        currentVersion: "dev",
        development: true,
        canInstall: false,
        installKind: "development",
      };
      break;
    default:
      break;
  }

  return {
    scenario: name,
    lang,
    snapshot: (): Promise<Snapshot> => {
      if (name === "loading") return new Promise(() => {});
      return Promise.resolve({
        schemaVersion: 1,
        version: name === "update-development" ? "dev" : "0.2.0",
        language: lang,
        config: { language: lang },
        catalog: snapshotCatalog,
      });
    },
    refresh: (): Promise<Catalog> => {
      if (name === "error") {
        return Promise.reject({
          problem: { code: "error.refresh", message: "No se pudo leer Steam." },
        });
      }
      return Promise.resolve(snapshotCatalog ?? catalog());
    },
    getRecovery: () => recoveryValue,
    getSituation: () => ({ ...situation, setupSnapshot: setup }),
    getSetup: () => setup,
    getState: () => sessionState,
    getLast: () => lastValue,
    setState: (next: SessionState | undefined) => {
      sessionState = next;
    },
    setSituation: (next: Situation) => {
      situation = next;
    },
    setSetup: (next: SetupSnapshot) => {
      setup = next;
    },
    setLast: (next: LastSession | undefined) => {
      lastValue = next;
    },
    setLanguage: (next: "es" | "en") => {
      lang = next;
    },
    getLanguage: () => lang,
    getUpdate: () => updateValue,
  };
}

export function installMockTransport(scenarioName: string, language: "es" | "en"): void {
  const scenario = (scenarioName in SCENARIOS ? scenarioName : "library") as Scenario;
  const data = scenarioData(scenario, language);

  setTransport({
    async call(_objectID, _method, _windowName, payload: unknown) {
      const call = payload as { methodID?: number; args?: unknown[] } | null;
      const args = call?.args ?? [];
      switch (call?.methodID) {
        case 2461216770:
          return data.snapshot();
        case 1442126872:
          return data.refresh();
        case 1626599784:
          return;
        case 1417353986:
          if (scenario === "start-pending") return new Promise(() => {});
          return start(args[0] as { targetAppId: number });
        case 2503687275:
          return replay();
        case 2442709936:
          return cancel();
        case 1754684680:
          return restoreSession();
        case 3488450910:
          return data.getRecovery();
        case 2307575735:
          return data.getState() ?? state("idle");
        case 822314934:
          return data.getSituation();
        case 523681163:
          return data.getSetup();
        case 2513448886:
          return installSetup();
        case 2098282818:
          return recordGuide();
        case 349277801:
          return restoreSetup();
        case 459589820:
          return "better-rpt-helper --donor -- %command%";
        case 1080211332:
          data.setLanguage(args[0] as "es" | "en");
          return;
        case 1155290161:
        case 3814729318:
          return;
        case 1761850530:
          return "0.2.0";
        case 2429811133:
          return diagnostics(data.getLanguage());
        case 1614206993:
          return "/home/tester/.local/share/better-rpt/diagnostics/better-rpt-diagnostic-2026-10-20T12-00-00Z.md";
        case 1725841076:
        case 1746319122:
          return data.getUpdate();
        case 1967629556:
          return {
            status: "replaced",
            installKind: "appimage",
            path: "/home/tester/Apps/BetterRPT-0.2.0-x86_64.AppImage",
            restart: true,
          };
        case 2374902849:
          return;
        default:
          throw new Error(`unmocked bridge call ${String(call?.methodID)}`);
      }
    },
  });

  function target(appId: number): Game {
    return games.find((item) => item.appId === appId) ?? games[1];
  }

  function start(input: { targetAppId: number }): Active {
    const next = state("preparing", input.targetAppId);
    data.setState(next);
    data.setSituation(baseSituation({ kind: "prepared", state: next }));
    return { token: next.token, game: target(input.targetAppId), state: next };
  }

  function replay(): Active {
    const previous = data.getLast();
    const next = state("active", previous?.appId ?? 553850);
    data.setState(next);
    data.setSituation(baseSituation({ kind: "active", restore: "session_active", state: next }));
    return { token: next.token, game: target(next.targetAppId), state: next };
  }

  function cancel(): boolean {
    if (data.getState()?.phase !== "preparing") return false;
    data.setState(state("idle"));
    data.setSituation(baseSituation({ kind: "idle" }));
    return true;
  }

  function restoreSession(): void {
    data.setSituation(baseSituation({ kind: "restored", restore: "not_needed" }));
    data.setSetup({ status: "restored", hasBackup: false });
  }

  function restoreSetup(): Setup {
    restoreSession();
    return { status: "restored", hasBackup: false };
  }

  function installSetup(): Setup {
    const next: Setup = { status: "installed", hasBackup: true, originalValue: "gamemoderun %command%" };
    data.setSetup(next);
    data.setSituation(baseSituation({ kind: "idle", setup: "installed" }));
    return next;
  }

  function recordGuide(): Setup {
    const next: Setup = { status: "guide", hasBackup: true, originalValue: "gamemoderun %command%" };
    data.setSetup(next);
    data.setSituation(baseSituation({ kind: "idle", setup: "guide" }));
    return next;
  }
}

const SCENARIOS: Record<Scenario, true> = {
  library: true,
  "many-games": true,
  loading: true,
  empty: true,
  error: true,
  preparing: true,
  launching: true,
  active: true,
  ended: true,
  stale: true,
  "donor-missing": true,
  incompatible: true,
  details: true,
  "dialog-unknown": true,
  "session-error": true,
  "setup-missing": true,
  "update-available": true,
  "update-development": true,
  "start-pending": true,
};
