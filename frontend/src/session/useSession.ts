import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";

import { toBridgeError } from "../bridge/client";
import { bridge } from "../bridge/wails";
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
  /** App ID whose launch is waiting for the backend or Steam confirmation. */
  launching: number | null;
  ready: boolean;
  now: number;
  dialog: SessionDialog;
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
  const [launching, setLaunching] = useState<number | null>(null);
  const [ready, setReady] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  const [dialog, setDialog] = useState<SessionDialog>(null);
  const warned = useRef(new Set<number>());

  const refreshContext = useCallback(async () => {
    const [nextSituation, nextSetup] = await Promise.all([
      bridge.Situation(),
      bridge.SetupSnapshot(),
    ]);
    setSituation(nextSituation);
    setSetup(nextSetup);
    if (nextSituation.last) setLast(nextSituation.last);
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
      try {
        const next = await bridge.SessionState(state.token);
        if (!alive) return;
        setState(next);
        if (isTerminal(next.phase)) await refreshContext();
      } catch {
        // A transient read failure must not discard the last known state.
      }
    };
    const timer = window.setInterval(() => void tick(), 1000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [refreshContext, state?.phase, state?.token]);

  useEffect(() => {
    if (state?.phase !== "active") return;
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [state?.phase]);

  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(() => setNotice(null), 5000);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const startNow = useCallback(
    async (game: Game) => {
      setBusy(true);
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
      }
    },
    [refreshContext],
  );

  const requestStart = useCallback(
    (game: Game) => {
      if (game.status === "unknown" && !warned.current.has(game.appId)) {
        setDialog({ kind: "unknown", game });
        return;
      }
      void startNow(game);
    },
    [startNow],
  );

  const confirmDialog = useCallback(async () => {
    const current = dialog;
    if (!current) return;
    setDialog(null);
    if (current.kind === "unknown") {
      warned.current.add(current.game.appId);
      await startNow(current.game);
      return;
    }
    setBusy(true);
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
    }
  }, [dialog, last?.token, refreshContext, setup?.hasBackup, situation?.token, state?.token, startNow]);

  const restore = useCallback(() => {
    if (!situation) return;
    if (situation.restore !== "available") {
      setNotice({ key: restoreReasonKey(situation.restore), tone: "amber" });
      return;
    }
    setDialog({ kind: "restore" });
  }, [situation]);

  const cancel = useCallback(async () => {
    if (!state || state.phase !== "preparing") return;
    setBusy(true);
    try {
      await bridge.CancelSession();
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
    }
  }, [refreshContext, state]);

  const replay = useCallback(async () => {
    setBusy(true);
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
    }
  }, [last?.appId, refreshContext, state?.targetAppId]);

  const installSetup = useCallback(async () => {
    setBusy(true);
    setProblem(null);
    try {
      await bridge.InstallSetup();
      setNotice({ key: "guide.installDone", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
    }
  }, [refreshContext]);

  const recordGuide = useCallback(async () => {
    setBusy(true);
    setProblem(null);
    try {
      await bridge.RecordGuideSetup();
      setNotice({ key: "guide.recordDone", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
    }
  }, [refreshContext]);

  const refreshSetup = useCallback(async () => {
    setBusy(true);
    setProblem(null);
    try {
      const next = await bridge.SetupSnapshot();
      setSetup(next);
      setNotice({ key: "guide.checkDone", tone: "mint" });
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
    }
  }, []);

  const ensureHelper = useCallback(async () => {
    setBusy(true);
    setProblem(null);
    try {
      await bridge.EnsureHelper();
      setNotice({ key: "wizard.helper.done", tone: "mint" });
      await refreshContext();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
    } finally {
      setBusy(false);
    }
  }, [refreshContext]);

  const recover = useCallback(async () => {
    setBusy(true);
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
    }
  }, [refreshContext]);

  const openSpaceWarInstall = useCallback(async () => {
    setProblem(null);
    try {
      await bridge.OpenSpaceWarInstall();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
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
      launching,
      ready,
      now,
      dialog,
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
      cancel,
      confirmDialog,
      dialog,
      ensureHelper,
      installSetup,
      last,
      launching,
      notice,
      now,
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
      setup,
      situation,
      state,
    ],
  );
}

export function isTerminal(phase: SessionState["phase"]): boolean {
  return phase === "ended" || phase === "error" || phase === "idle";
}

export function isPreparing(phase: SessionState["phase"] | undefined): boolean {
  return phase === "preparing" || phase === "donor_starting" || phase === "target_starting";
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
