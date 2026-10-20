import * as generated from "../../bindings/github.com/srnoob2570/better-rpt/internal/bridge/bridge";
import type * as generatedModels from "../../bindings/github.com/srnoob2570/better-rpt/internal/bridge/models";
import type { Bridge } from "./client";
import type {
  Active,
  Catalog,
  Game,
  Language,
  Recovery,
  SessionState,
  Setup,
  SetupSnapshot,
  Situation,
  Snapshot,
  StartSessionInput,
  Warning,
} from "./types";

// Wails models slices as nullable even though the Go bridge always emits [].
// Keep that normalization here, at the generated-binding boundary.
function toCatalog(value: generatedModels.Catalog): Catalog {
  return {
    games: (value.games ?? []) as unknown as Game[],
    warnings: (value.warnings ?? []) as unknown as Warning[],
  };
}

export const bridge: Bridge = {
  async Snapshot(): Promise<Snapshot> {
    const value = await generated.Snapshot();
    return {
      ...value,
      language: value.language as Language,
      config: { language: value.config.language as Language },
      catalog: value.catalog ? toCatalog(value.catalog) : undefined,
    };
  },
  async RefreshCatalog(): Promise<Catalog> {
    return toCatalog(await generated.RefreshCatalog());
  },
  async StartSession(input: StartSessionInput): Promise<Active> {
    return (await generated.StartSession(input)) as unknown as Active;
  },
  async ReplaySession(): Promise<Active> {
    return (await generated.ReplaySession()) as unknown as Active;
  },
  CancelSession(): Promise<boolean> {
    return generated.CancelSession();
  },
  RestoreSession(token: string): Promise<void> {
    return generated.RestoreSession(token);
  },
  async RecoverSession(): Promise<Recovery> {
    return (await generated.RecoverSession()) as unknown as Recovery;
  },
  async SessionState(token: string): Promise<SessionState> {
    return (await generated.SessionState(token)) as unknown as SessionState;
  },
  async Situation(): Promise<Situation> {
    return (await generated.Situation()) as unknown as Situation;
  },
  async SetupSnapshot(): Promise<SetupSnapshot> {
    return (await generated.SetupSnapshot()) as unknown as SetupSnapshot;
  },
  async InstallSetup(): Promise<Setup> {
    return (await generated.InstallSetup()) as unknown as Setup;
  },
  async RecordGuideSetup(): Promise<Setup> {
    return (await generated.RecordGuideSetup()) as unknown as Setup;
  },
  async RestoreSetup(): Promise<Setup> {
    return (await generated.RestoreSetup()) as unknown as Setup;
  },
  DispatcherCommand(): Promise<string> {
    return generated.DispatcherCommand();
  },
  SetLanguage(language: Language): Promise<void> {
    return generated.SetLanguage(language);
  },
  OpenSpaceWarInstall(): Promise<void> {
    return generated.OpenSpaceWarInstall();
  },
  OpenSteam(): Promise<void> {
    return generated.OpenSteam();
  },
  Version(): Promise<string> {
    return generated.Version();
  },
};
