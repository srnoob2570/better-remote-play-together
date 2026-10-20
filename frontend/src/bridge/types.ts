export type Language = "es" | "en";
export type GameKind = "native" | "proton" | "unknown";
export type GameStatus = "ready" | "unknown" | "incompatible" | "missing" | "error" | "active";
export type SessionPhase =
  | "idle"
  | "preparing"
  | "donor_starting"
  | "target_starting"
  | "active"
  | "ended"
  | "error"
  | "restoring";
export type LastOutcome = "pending" | "ok" | "error" | "timeout" | "canceled" | "unknown";
export type LastSource = "better-rpt" | "steam";
export type SetupStatus = "missing" | "guide" | "installed" | "restored" | "conflict";
export type SituationKind =
  | "idle"
  | "prepared"
  | "active"
  | "finished"
  | "restore_pending"
  | "restored"
  | "orphan"
  | "conflict";
export type RestoreState =
  | "available"
  | "not_needed"
  | "steam_running"
  | "session_active"
  | "conflict"
  | "no_backup";

export interface Config {
  language: Language;
}

export interface Problem {
  code: string;
  message: string;
  detail?: string;
}

export interface Snapshot {
  schemaVersion: number;
  version: string;
  language: Language;
  config: Config;
  catalog?: Catalog;
}

export interface Catalog {
  games: Game[];
  warnings: Warning[];
}

export interface Game {
  appId: number;
  name: string;
  installDir: string;
  libraryPath: string;
  sizeBytes: number;
  kind: GameKind;
  status: GameStatus;
  protonName?: string;
  problem?: Problem;
}

export interface Warning {
  path: string;
  problem: Problem;
}

export interface StartSessionInput {
  targetAppId: number;
  closeSteam: boolean;
}

export interface Active {
  token: string;
  game: Game;
  state: SessionState;
}

export interface SessionState {
  token: string;
  donorAppId: number;
  targetAppId: number;
  phase: SessionPhase;
  outcome?: LastOutcome;
  error?: Problem;
  exitCode?: number;
  createdAt?: string;
  updatedAt?: string;
  startedAt?: string;
  endedAt?: string;
}

export interface Recovery {
  state?: SessionState;
  helperAlive: boolean;
  situation?: Situation;
  setup?: SetupSnapshot;
  last?: LastSession;
  stale?: Stale;
  expiredPlans: string[];
}

export interface Stale {
  token: string;
  targetAppId: number;
  active: boolean;
}

export interface Situation {
  kind: SituationKind;
  setup: SetupStatus;
  restore: RestoreState;
  token?: string;
  targetAppId?: number;
  steamRunning: boolean;
  state?: SessionState;
  last?: LastSession;
}

export interface LastSession {
  appId: number;
  name?: string;
  token?: string;
  source: LastSource;
  startedAt?: string;
  endedAt?: string;
  outcome: LastOutcome;
  error?: string;
  kind?: GameKind;
}

export interface Setup {
  status: SetupStatus;
  originalValue?: string;
  hasBackup: boolean;
}

export interface SetupSnapshot {
  status: SetupStatus;
  originalValue?: string;
  hasBackup: boolean;
  currentValue?: string;
}
