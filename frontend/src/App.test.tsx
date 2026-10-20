import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Catalog, Game, Snapshot } from "./bridge/types";
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

describe("library UI", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocked.Snapshot.mockResolvedValue(snapshot());
    mocked.RefreshCatalog.mockResolvedValue(catalog);
    mocked.SetLanguage.mockResolvedValue();
    mocked.OpenSteam.mockResolvedValue();
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

  it("renders skeletons while the snapshot is loading", () => {
    mocked.Snapshot.mockReturnValue(new Promise(() => {}));
    render(<App />);

    expect(document.querySelector('[aria-busy="true"]')).toBeInTheDocument();
  });
});
