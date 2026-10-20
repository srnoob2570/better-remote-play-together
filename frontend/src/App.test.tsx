import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  Catalog,
  Game,
  LastSession,
  Recovery,
  SessionState,
  Situation,
  Snapshot,
} from "./bridge/types";
import App from "./App";

vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: vi.fn(() => () => {}),
  },
}));

vi.mock("./bridge/wails", () => ({
  bridge: {
    Snapshot: vi.fn(),
    RefreshCatalog: vi.fn(),
    SetLanguage: vi.fn(),
    OpenSteam: vi.fn(),
    RecoverSession: vi.fn(),
    Situation: vi.fn(),
    SetupSnapshot: vi.fn(),
    StartSession: vi.fn(),
    ReplaySession: vi.fn(),
    CancelSession: vi.fn(),
    RestoreSession: vi.fn(),
    RestoreSetup: vi.fn(),
    InstallSetup: vi.fn(),
    RecordGuideSetup: vi.fn(),
    DispatcherCommand: vi.fn(),
    OpenSpaceWarInstall: vi.fn(),
    SessionState: vi.fn(),
  },
}));

import { bridge } from "./bridge/wails";

const mocked = vi.mocked(bridge);

function game(overrides: Partial<Game> & Pick<Game, "appId" | "name">): Game {
  return {
    installDir: "",
    libraryPath: "",
    sizeBytes: 0,
    kind: "proton",
    status: "unknown",
    ...overrides,
  };
}

const catalog: Catalog = {
  games: [
    game({ appId: 553850, name: "HELLDIVERS™ 2", status: "unknown" }),
    game({ appId: 1599600, name: "PlateUp!", status: "unknown" }),
    game({ appId: 892970, name: "Valheim", kind: "native", status: "ready" }),
  ],
  warnings: [],
};

function snapshot(value: Catalog = catalog): Snapshot {
  return {
    schemaVersion: 1,
    version: "0.1.2",
    language: "es",
    config: { language: "es" },
    catalog: value,
  };
}

function session(overrides: Partial<SessionState>): SessionState {
  return {
    token: "0123456789abcdef0123456789abcdef",
    donorAppId: 480,
    targetAppId: 553850,
    phase: "preparing",
    ...overrides,
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

function recovery(overrides: Partial<Recovery> = {}): Recovery {
  return {
    helperAlive: false,
    expiredPlans: [],
    ...overrides,
  };
}

function lastSession(overrides: Partial<LastSession> = {}): LastSession {
  return {
    appId: 553850,
    name: "HELLDIVERS™ 2",
    source: "better-rpt",
    outcome: "ok",
    ...overrides,
  };
}

describe("library UI", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocked.Snapshot.mockResolvedValue(snapshot());
    mocked.RefreshCatalog.mockResolvedValue(catalog);
    mocked.SetLanguage.mockResolvedValue();
    mocked.OpenSteam.mockResolvedValue();
    mocked.RecoverSession.mockResolvedValue({
      helperAlive: false,
      expiredPlans: [],
    });
    mocked.Situation.mockResolvedValue({
      kind: "idle",
      setup: "missing",
      restore: "not_needed",
      steamRunning: false,
    });
    mocked.SetupSnapshot.mockResolvedValue({ status: "missing", hasBackup: false });
    mocked.DispatcherCommand.mockResolvedValue("better-rpt-helper --donor -- %command%");
    mocked.SessionState.mockResolvedValue({
      token: "",
      donorAppId: 480,
      targetAppId: 0,
      phase: "idle",
    });
    mocked.OpenSpaceWarInstall.mockResolvedValue();
    mocked.CancelSession.mockResolvedValue(false);
    mocked.ReplaySession.mockRejectedValue(new Error("no last session"));
    mocked.StartSession.mockRejectedValue(new Error("not configured"));
    mocked.InstallSetup.mockResolvedValue({ status: "installed", hasBackup: true });
    mocked.RecordGuideSetup.mockResolvedValue({ status: "guide", hasBackup: true });
    mocked.RestoreSetup.mockResolvedValue({ status: "restored", hasBackup: false });
    mocked.RestoreSession.mockResolvedValue();
  });

  it("renders the catalog, count and lazy artwork", async () => {
    render(<App />);

    expect(await screen.findAllByText("HELLDIVERS™ 2")).not.toHaveLength(0);
    expect(screen.getByText("3 juegos")).toBeInTheDocument();
    expect(screen.getByText("Versión 0.1.2")).toBeInTheDocument();

    const image = document.querySelector<HTMLImageElement>("img.game-icon");
    expect(image).not.toBeNull();
    expect(image?.getAttribute("src")).toBe("/api/art/553850/icon");
    expect(image?.getAttribute("loading")).toBe("lazy");
    expect(image?.getAttribute("decoding")).toBe("async");
  });

  it("filters by status and searches by name or App ID", async () => {
    render(<App />);
    await screen.findAllByText("HELLDIVERS™ 2");

    fireEvent.click(screen.getByRole("radio", { name: "Listos" }));
    expect(screen.getByRole("row", { name: /Valheim/ })).toBeInTheDocument();
    expect(screen.queryByRole("row", { name: /PlateUp/ })).not.toBeInTheDocument();
    expect(screen.getByText("1 juego")).toBeInTheDocument();

    const search = screen.getByRole("searchbox", {
      name: "Buscar por nombre o App ID",
    });
    fireEvent.change(search, { target: { value: "1599600" } });
    expect(await screen.findByText("Ningún juego coincide con la búsqueda.")).toBeInTheDocument();
  });

  it("shows the empty state", async () => {
    mocked.Snapshot.mockResolvedValue(snapshot({ games: [], warnings: [] }));
    render(<App />);

    expect(await screen.findByText("No hay juegos instalados.")).toBeInTheDocument();
  });

  it("shows the error state and retries", async () => {
    mocked.RefreshCatalog.mockRejectedValueOnce({
      problem: { code: "error.refresh", message: "falló la lectura" },
    });
    render(<App />);
    await screen.findAllByText("HELLDIVERS™ 2");

    fireEvent.click(screen.getByRole("button", { name: "Actualizar" }));

    expect(
      await screen.findByText("No se pudo leer la biblioteca de Steam."),
    ).toBeInTheDocument();
    expect(screen.getByText("falló la lectura")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reintentar" })).toBeInTheDocument();
  });

  it("changes the language through the bridge", async () => {
    render(<App />);
    await screen.findAllByText("HELLDIVERS™ 2");

    fireEvent.click(screen.getByRole("button", { name: "English" }));

    expect(mocked.SetLanguage).toHaveBeenCalledWith("en");
    expect(await screen.findByRole("heading", { name: "Library" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Refresh" })).toBeInTheDocument();
  });

  it("shows the last session when the library is empty", async () => {
    mocked.Snapshot.mockResolvedValue(snapshot({ games: [], warnings: [] }));
    mocked.RecoverSession.mockResolvedValue(
      recovery({ last: lastSession({ startedAt: "2026-10-20T10:00:00Z", endedAt: "2026-10-20T10:30:00Z" }) }),
    );
    render(<App />);

    expect(await screen.findByRole("heading", { name: "Última sesión" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Jugar de nuevo" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Abrir última sesión" })).toBeInTheDocument();
  });

  it("cancels a session only while it is preparing", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        state: session({ phase: "preparing" }),
        situation: situation({ kind: "prepared", setup: "installed" }),
        setup: { status: "installed", hasBackup: true },
      }),
    );
    render(<App />);

    expect(await screen.findByText("Preparando la sesión")).toBeInTheDocument();
    expect(screen.getByText("Paso 1 de 5")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(mocked.CancelSession).toHaveBeenCalled();
  });

  it("renders the active session timer and guide", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        state: session({ phase: "active", startedAt: new Date().toISOString() }),
        situation: situation({ kind: "active", setup: "installed" }),
        setup: { status: "installed", hasBackup: true },
      }),
    );
    render(<App />);

    expect(await screen.findByRole("heading", { name: "Sesión activa" })).toBeInTheDocument();
    expect(screen.getByText("Cómo invitar")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sesión activa" })).toBeDisabled();
  });

  it("opens the restore dialog and restores the setup", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        state: session({
          phase: "ended",
          outcome: "ok",
          startedAt: "2026-10-20T10:00:00Z",
          endedAt: "2026-10-20T10:30:00Z",
        }),
        situation: situation({ kind: "finished", setup: "installed", restore: "available" }),
        setup: { status: "installed", hasBackup: true },
        last: lastSession({ startedAt: "2026-10-20T10:00:00Z", endedAt: "2026-10-20T10:30:00Z" }),
      }),
    );
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Restaurar SpaceWar" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Restaurar SpaceWar")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Restaurar" }));
    expect(mocked.RestoreSetup).toHaveBeenCalled();
  });

  it("warns once before starting an unknown game and then starts it", async () => {
    const donor: Game = {
      appId: 480,
      name: "SpaceWar",
      installDir: "/steam/spacewar",
      libraryPath: "/steam",
      sizeBytes: 1,
      kind: "native",
      status: "ready",
    };
    mocked.Snapshot.mockResolvedValue(
      snapshot({ ...catalog, games: [donor, ...catalog.games] }),
    );
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        setup: { status: "installed", hasBackup: true },
        situation: situation({ kind: "prepared", setup: "installed" }),
      }),
    );
    mocked.SetupSnapshot.mockResolvedValue({ status: "installed", hasBackup: true });
    mocked.StartSession.mockResolvedValue({
      token: "0123456789abcdef0123456789abcdef",
      game: catalog.games[0],
      state: session({ phase: "preparing" }),
    });
    render(<App />);
    await screen.findAllByText("HELLDIVERS™ 2");

    fireEvent.click(screen.getByRole("button", { name: "Jugar" }));
    expect(await screen.findByText("Compatibilidad desconocida")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continuar" }));
    expect(mocked.StartSession).toHaveBeenCalledWith({
      targetAppId: 553850,
      closeSteam: false,
    });
  });

  it("shows the guide command and setup state", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({ setup: { status: "guide", hasBackup: true } }),
    );
    render(<App />);
    await screen.findAllByText("HELLDIVERS™ 2");

    fireEvent.mouseDown(screen.getByRole("tab", { name: "Guía" }));
    expect(await screen.findByText("Comando para SpaceWar")).toBeInTheDocument();
    expect(screen.getByText("better-rpt-helper --donor -- %command%")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copiar comando" })).toBeInTheDocument();
  });

  it("renders a stale configuration notice with a restore action", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        situation: situation({
          kind: "restore_pending",
          setup: "installed",
          restore: "available",
        }),
        last: lastSession(),
      }),
    );
    render(<App />);

    expect(await screen.findByText("Configuración anterior")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Restaurar SpaceWar" }).length).toBeGreaterThan(0);
  });

  it("renders a session error with its technical detail", async () => {
    mocked.RecoverSession.mockResolvedValue(
      recovery({
        state: session({
          phase: "error",
          error: { code: "setup_missing", message: "missing setup", detail: "stack" },
        }),
        situation: situation({ kind: "finished", restore: "not_needed" }),
      }),
    );
    render(<App />);

    expect(
      await screen.findByRole("heading", { name: "No se pudo completar la operación" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Instala el comando desde la guía antes de jugar.")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Detalle técnico"));
    expect(screen.getByText("stack")).toBeInTheDocument();
  });

  it("renders skeletons while the snapshot is loading", () => {
    mocked.Snapshot.mockReturnValue(new Promise(() => {}));
    render(<App />);

    expect(document.querySelector('[aria-busy="true"]')).toBeInTheDocument();
  });
});
