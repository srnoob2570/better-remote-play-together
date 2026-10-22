import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";

import { toBridgeError } from "../bridge/client";
import { bridge } from "../bridge/wails";
import { focusInvoker } from "../lib/focus";
import type {
  Game,
  LastSession,
  Problem,
  SessionState,
  SetupSnapshot,
  Situation,
} from "../bridge/types";
import type { MessageKey } from "../i18n";

export type SessionDialog =
  | { kind: "unknown"; game: Game }
  | { kind: "restore" }
  | null;

export type SessionAction =
  | "start"
  | "replay"
  | "cancel"
  | "restore"
  | "installSetup"
  | "recordGuide"
  | "refreshSetup"
  | "ensureHelper"
  | "recover"
  | "openSpaceWar";

export interface SessionNotice {
  key: MessageKey;
  args?: Array<string | number>;
  tone?: "amber" | "mint";
}

export interface SessionController {
  state?: SessionState;
  situation?: Situation;
  setup?: SetupSnapshot;
  last?: LastSession;
  problem: Problem | null;
  notice: SessionNotice | null;
  busy: boolean;
  busyAction: SessionAction | null;
  /** App ID whose launch is waiting for the backend or Steam confirmation. */
  launching: number | null;
  ready: boolean;
  dialog: SessionDialog;
  restoreDialogFocus: () => void;
  requestStart: (game: Game) => void;
  confirmDialog: () => Promise<void>;
  closeDialog: () => void;
  cancel: () => Promise<void>;
  restore: () => void;
  replay: () => Promise<void>;
  refreshContext: () => Promise<void>;
  installSetup: () => Promise<void>;
  recordGuide: () => Promise<void>;
  refreshSetup: () => Promise<void>;
  ensureHelper: () => Promise<void>;
  recover: () => Promise<void>;
  openSpaceWarInstall: () => Promise<void>;
}

export function useSession(): SessionController {
  const [state, setState] = useState<SessionState>();
  const [situation, setSituation] = useState<Situation>();
  const [setup, setSetup] = useState<SetupSnapshot>();
  const [last, setLast] = useState<LastSession>();
  const [problem, setProblem] = useState<Problem | null>(null);
  const [notice, setNotice] = useState<SessionNotice | null>(null);
  const [busy, setBusy] = useState(false);
  const [busyAction, setBusyAction] = useState<SessionAction | null>(null);
  const [launching, setLaunching] = useState<number | null>(null);
  const [ready, setReady] = useState(false);
  const [dialog, setDialog] = useState<SessionDialog>(null);
  const warned = useRef(new Set<number>());
  const dialogInvoker = useRef<HTMLElement | null>(null);
  const pollInFlight = useRef(false);

  const openDialog = useCallback((next: Exclude<SessionDialog, null>) => {
    const active = document.activeElement;
    dialogInvoker.current = active instanceof HTMLElement ? active : null;
    setDialog(next);
  }, []);

  const restoreDialogFocus = useCallback(() => {
    focusInvoker(dialogInvoker.current);
  }, []);

  const refreshContext = useCallback(async () => {
    const next = await bridge.Situation();
    setSituation(next);
    if (next.setupSnapshot) setSetup(next.setupSnapshot);
    if (next.last) setLast(next.last);
  }, []);

  useEffect(() => {
    let alive = true;
    const off = Events.On("session.updated", (event) => {
      if (!alive) return;
      const next = event.data as SessionState;
      setState(next);
      if (isTerminal(next.phase)) void refreshContext().catch(() => {});
    });

    async function load() {
      try {
        const recovery = await bridge.RecoverSession();
        if (!alive) return;
        setState(recovery.state);
        setSituation(recovery.situation);
        setSetup(recovery.setup);
        setLast(recovery.last ?? recovery.situation?.last);
      } catch (reason: unknown) {
        if (alive) setProblem(toBridgeError(reason).problem);
      } finally {
        if (alive) setReady(true);
      }
    }

    void load();
    return () => {
      alive = false;
      off();
    };
  }, [refreshContext]);

  useEffect(() => {
    if (!state?.token || isTerminal(state.phase)) return;
    let alive = true;
    const tick = async () => {
      if (pollInFlight.current || document.visibilityState === "hidden") return;
      pollInFlight.current = true;
      try {
        const next = await bridge.SessionState(state.token);
        if (!alive) return;
        setState((current) =>
          current && sameSessionState(current, next) ? current : next,
        );
        if (isTerminal(next.phase)) await refreshContext();
      } catch {
        // A transient read failure must not discard the last known state.
      } finally {
        pollInFlight.current = false;
      }
    };
    const onVisibility = () => {
      if (document.visibilityState === "visible") void tick();
    };
    const timer = window.setInterval(() => void tick(), 1000);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      alive = false;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [refreshContext, state?.phase, state?.token]);

  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(() => setNotice(null), 5000);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const startNow = useCallback(
    async (game: Game) => {
      setBusy(true);
      setBusyAction("start");
      setLaunching(game.appId);
      setProblem(null);
      setNotice(null);
      try {
        const active = await bridge.StartSession({ targetAppId: game.appId, closeSteam: false });
        setState(active.state);
        setLaunching(null);
        await refreshContext();
      } catch (reason: unknown) {
        setProblem(toBridgeError(reason).problem);
      } finally {
        setLaunching(null);
        setBusy(false);
        setBusyAction(null);
      }
    },
    [refreshContext],
  );

  const requestStart = useCallback(
    (game: Game) => {
      if (game.status === "unknown" && !warned.current.has(game.appId)) {
        openDialog({ kind: "unknown", game });
        return;
      }
      void startNow(game);
    },
    [openDialog, startNow],
  );

  const confirmDialog = useCallback(async () => {
    const current = dialog;
    if (!current) return;
    if (current.kind === "unknown") {
      setDialog(null);
      warned.current.add(current.game.appId);
      await startNow(current.game);
      return;
    }
    setBusy(true);
    setBusyAction("restore");
    setProblem(null);
    try {
      if (setup?.hasBackup) {
        await bridge.RestoreSetup();
      } else {
        const token = situation?.token ?? last?.token ?? state?.token;
        if (!token) return;
        await bridge.RestoreSession(token);
      }
      setNotice({ key: "restore.done.title", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
      setDialog(null);
    }
  }, [dialog, last?.token, refreshContext, setup?.hasBackup, situation?.token, state?.token, startNow]);

  const restore = useCallback(() => {
    if (!situation) return;
    if (situation.restore !== "available") {
      setNotice({ key: restoreReasonKey(situation.restore), tone: "amber" });
      return;
    }
    openDialog({ kind: "restore" });
  }, [openDialog, situation]);

  const cancel = useCallback(async () => {
    if (!state || state.phase !== "preparing") return;
    setBusy(true);
    setBusyAction("cancel");
    try {
      await bridge.CancelSession();
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, [refreshContext, state]);

  const replay = useCallback(async () => {
    setBusy(true);
    setBusyAction("replay");
    setLaunching(last?.appId ?? state?.targetAppId ?? null);
    setProblem(null);
    try {
      const active = await bridge.ReplaySession();
      setState(active.state);
      setLaunching(null);
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setLaunching(null);
      setBusy(false);
      setBusyAction(null);
    }
  }, [last?.appId, refreshContext, state?.targetAppId]);

  const installSetup = useCallback(async () => {
    setBusy(true);
    setBusyAction("installSetup");
    setProblem(null);
    try {
      await bridge.InstallSetup();
      setNotice({ key: "guide.installDone", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, [refreshContext]);

  const recordGuide = useCallback(async () => {
    setBusy(true);
    setBusyAction("recordGuide");
    setProblem(null);
    try {
      await bridge.RecordGuideSetup();
      setNotice({ key: "guide.recordDone", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, [refreshContext]);

  const refreshSetup = useCallback(async () => {
    setBusy(true);
    setBusyAction("refreshSetup");
    setProblem(null);
    try {
      const next = await bridge.SetupSnapshot();
      setSetup(next);
      setNotice({ key: "guide.checkDone", tone: "mint" });
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, []);

  const ensureHelper = useCallback(async () => {
    setBusy(true);
    setBusyAction("ensureHelper");
    setProblem(null);
    try {
      await bridge.EnsureHelper();
      setNotice({ key: "wizard.helper.done", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, [refreshContext]);

  const recover = useCallback(async () => {
    setBusy(true);
    setBusyAction("recover");
    setProblem(null);
    try {
      const recovery = await bridge.RecoverSession();
      setState(recovery.state);
      setSituation(recovery.situation);
      setSetup(recovery.setup);
      setLast(recovery.last ?? recovery.situation?.last);
      setNotice({ key: "wizard.repair.recovered", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, [refreshContext]);

  const openSpaceWarInstall = useCallback(async () => {
    setBusy(true);
    setBusyAction("openSpaceWar");
    setProblem(null);
    try {
      await bridge.OpenSpaceWarInstall();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
      setBusyAction(null);
    }
  }, []);

  return useMemo(
    () => ({
      state,
      situation,
      setup,
      last,
      problem,
      notice,
      busy,
      busyAction,
      launching,
      ready,
      dialog,
      restoreDialogFocus,
      requestStart,
      confirmDialog,
      closeDialog: () => setDialog(null),
      cancel,
      restore,
      replay,
      refreshContext,
      installSetup,
      recordGuide,
      refreshSetup,
      ensureHelper,
      recover,
      openSpaceWarInstall,
    }),
    [
      busy,
      busyAction,
      cancel,
      confirmDialog,
      dialog,
      ensureHelper,
      installSetup,
      last,
      launching,
      notice,
      openSpaceWarInstall,
      problem,
      ready,
      recordGuide,
      recover,
      refreshContext,
      refreshSetup,
      replay,
      requestStart,
      restore,
      restoreDialogFocus,
      setup,
      situation,
      state,
    ],
  );
}

export function isTerminal(phase: SessionState["phase"]): boolean {
  return phase === "ended" || phase === "error" || phase === "idle";
}

function sameSessionState(a: SessionState, b: SessionState): boolean {
  return (
    a.token === b.token &&
    a.phase === b.phase &&
    a.outcome === b.outcome &&
    a.exitCode === b.exitCode &&
    a.error?.code === b.error?.code &&
    a.error?.message === b.error?.message &&
    a.error?.detail === b.error?.detail &&
    a.startedAt === b.startedAt &&
    a.endedAt === b.endedAt &&
    a.updatedAt === b.updatedAt
  );
}

export function isPreparing(phase: SessionState["phase"] | undefined): boolean {
  return (
    phase === "preparing" ||
    phase === "launching" ||
    phase === "donor_starting" ||
    phase === "target_starting"
  );
}

export function restoreReasonKey(reason: Situation["restore"]): MessageKey {
  switch (reason) {
    case "steam_running":
      return "restore.reason.steamRunning";
    case "session_active":
      return "restore.reason.sessionActive";
    case "conflict":
      return "restore.reason.conflict";
    case "no_backup":
      return "restore.reason.noBackup";
    case "not_needed":
      return "restore.reason.notNeeded";
    default:
      return "restore.reason.notNeeded";
  }
}
