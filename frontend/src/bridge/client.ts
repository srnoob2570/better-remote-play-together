import type {
  Active,
  Catalog,
  DiagnosticsResult,
  Language,
  Problem,
  Recovery,
  SessionState,
  Setup,
  SetupSnapshot,
  Situation,
  Snapshot,
  StartSessionInput,
  UpdateCheck,
  UpdateInstallResult,
} from "./types";

/**
 * Manual facade over the Wails-generated bindings. Ticket 08 supplies the
 * implementation; components depend on this interface so tests can mock it.
 */
export interface Bridge {
  Snapshot(): Promise<Snapshot>;
  RefreshCatalog(): Promise<Catalog>;
  EnsureHelper(): Promise<void>;
  StartSession(input: StartSessionInput): Promise<Active>;
  ReplaySession(): Promise<Active>;
  CancelSession(): Promise<boolean>;
  RestoreSession(token: string): Promise<void>;
  RecoverSession(): Promise<Recovery>;
  SessionState(token: string): Promise<SessionState>;
  Situation(): Promise<Situation>;
  SetupSnapshot(): Promise<SetupSnapshot>;
  InstallSetup(): Promise<Setup>;
  RecordGuideSetup(): Promise<Setup>;
  RestoreSetup(): Promise<Setup>;
  DispatcherCommand(): Promise<string>;
  SetLanguage(language: Language): Promise<void>;
  OpenSpaceWarInstall(): Promise<void>;
  OpenSteam(): Promise<void>;
  RunDiagnostics(): Promise<DiagnosticsResult>;
  SaveDiagnostics(markdown: string): Promise<string>;
  Version(): Promise<string>;
  CheckForUpdate(): Promise<UpdateCheck>;
  CheckForUpdateOnStartup(): Promise<UpdateCheck>;
  InstallUpdate(): Promise<UpdateInstallResult>;
  RestartToUpdate(): Promise<void>;
}

export interface BridgeError {
  problem: Problem;
}

/** Normalizes any rejection into the shared Problem shape. */
export function toBridgeError(error: unknown): BridgeError {
  if (isRecord(error)) {
    const problem = error.problem;
    if (isProblem(problem)) {
      return { problem };
    }
    if (typeof error.code === "string" && typeof error.message === "string") {
      return {
        problem: {
          code: error.code,
          message: error.message,
          detail: typeof error.detail === "string" ? error.detail : undefined,
        },
      };
    }
  }
  return {
    problem: {
      code: "internal",
      message: error instanceof Error ? error.message : String(error),
    },
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isProblem(value: unknown): value is Problem {
  return isRecord(value) && typeof value.code === "string" && typeof value.message === "string";
}
