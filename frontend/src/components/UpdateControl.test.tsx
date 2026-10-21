import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UpdateCheck, UpdateInstallResult } from "../bridge/types";

vi.mock("@wailsio/runtime", () => ({
  Browser: {
    OpenURL: vi.fn().mockResolvedValue(undefined),
  },
  Events: {
    On: vi.fn(() => () => {}),
  },
}));

vi.mock("../bridge/wails", () => ({
  bridge: {
    CheckForUpdateOnStartup: vi.fn(),
    CheckForUpdate: vi.fn(),
    InstallUpdate: vi.fn(),
    RestartToUpdate: vi.fn(),
  },
}));

import { Browser, Events } from "@wailsio/runtime";

import { bridge } from "../bridge/wails";
import { UpdateControl } from "./UpdateControl";

const mocked = vi.mocked(bridge);
const events = vi.mocked(Events);

function updateCheck(overrides: Partial<UpdateCheck> = {}): UpdateCheck {
  return {
    currentVersion: "0.2.0",
    latestVersion: "0.2.1",
    tag: "v0.2.1",
    releaseUrl:
      "https://github.com/srnoob2570/better-remote-play-together/releases/tag/v0.2.1",
    available: true,
    canInstall: true,
    installKind: "appimage",
    development: false,
    checkedAt: "2026-10-21T12:00:00Z",
    cached: false,
    ...overrides,
  };
}

function installResult(overrides: Partial<UpdateInstallResult> = {}): UpdateInstallResult {
  return {
    status: "replaced",
    installKind: "appimage",
    path: "/home/test/Apps/BetterRPT-0.2.1-x86_64.AppImage",
    restart: true,
    ...overrides,
  };
}

function progressHandler(): ((event: { data: unknown }) => void) | undefined {
  const call = events.On.mock.calls.find(([name]) => name === "update.progress");
  return call?.[1] as unknown as ((event: { data: unknown }) => void) | undefined;
}

describe("update control", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    events.On.mockImplementation(() => () => {});
    mocked.CheckForUpdateOnStartup.mockRejectedValue({
      problem: { code: "update_network", message: "offline" },
    });
    mocked.CheckForUpdate.mockResolvedValue(updateCheck());
    mocked.InstallUpdate.mockResolvedValue(installResult());
    mocked.RestartToUpdate.mockResolvedValue();
  });

  it("shows a startup badge without opening a dialog", async () => {
    mocked.CheckForUpdateOnStartup.mockResolvedValue(updateCheck());
    render(<UpdateControl lang="es" version="0.2.0" />);

    expect(await screen.findByText("0.2.1")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Actualización 0.2.1 disponible" })).toBeEnabled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows up to date after a successful startup check", async () => {
    mocked.CheckForUpdateOnStartup.mockResolvedValue(
      updateCheck({ available: false, canInstall: false, latestVersion: "0.2.0" }),
    );
    render(<UpdateControl lang="es" version="0.2.0" />);

    expect(await screen.findByRole("button", { name: "Estás al día" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Estás al día con la versión 0.2.0.");
  });

  it("stays silent when the startup check fails", async () => {
    mocked.CheckForUpdateOnStartup.mockRejectedValue({
      problem: { code: "update_network", message: "offline" },
    });
    render(<UpdateControl lang="es" version="0.2.0" />);

    await waitFor(() => expect(mocked.CheckForUpdateOnStartup).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Buscar actualizaciones" })).toBeEnabled();
  });

  it("disables the button while checking", async () => {
    mocked.CheckForUpdate.mockReturnValue(new Promise(() => {}));
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));

    expect(
      screen.getByText("Comprobando…", { selector: "button" }),
    ).toBeDisabled();
    expect(await screen.findByText("Consultando GitHub Releases…")).toBeInTheDocument();
  });

  it("shows the current version when there is no update", async () => {
    mocked.CheckForUpdate.mockResolvedValue(
      updateCheck({ available: false, canInstall: false, latestVersion: "0.2.0" }),
    );
    render(<UpdateControl lang="es" version="0.2.0" />);

    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    expect(within(dialog).getByText("Versión actual: 0.2.0")).toBeInTheDocument();
    expect(
      within(dialog).getByText("Estás al día. La versión 0.2.0 es la más reciente."),
    ).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Actualizar ahora" })).not.toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Cerrar" }));
    expect(await screen.findByRole("button", { name: "Estás al día" })).toBeInTheDocument();
  });

  it("shows versions and release notes for an available update", async () => {
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));

    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    expect(within(dialog).getByText("Versión actual: 0.2.0")).toBeInTheDocument();
    expect(within(dialog).getByText("Nueva versión: 0.2.1")).toBeInTheDocument();
    expect(within(dialog).getByText("La versión 0.2.1 está disponible.")).toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: "Ver notas del release" })).toHaveAttribute(
      "href",
      updateCheck().releaseUrl,
    );
    expect(within(dialog).getByRole("button", { name: "Actualizar ahora" })).toBeInTheDocument();
  });

  it("moves through downloading and restart pending with progress", async () => {
    let resolveInstall: (value: UpdateInstallResult) => void = () => {};
    mocked.InstallUpdate.mockReturnValue(
      new Promise((resolve) => {
        resolveInstall = resolve;
      }),
    );
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Actualizar ahora" }));

    expect(mocked.InstallUpdate).toHaveBeenCalledTimes(1);
    expect(await screen.findByRole("progressbar")).toBeInTheDocument();
    const handler = progressHandler();
    expect(handler).toBeDefined();
    act(() => {
      handler?.({
        data: { stage: "downloading", downloaded: 42, total: 100, percent: 42 },
      });
    });
    expect(await screen.findByText("Descargando la actualización… 42%")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "42");

    act(() => {
      handler?.({
        data: { stage: "installing", downloaded: 0, total: 0, percent: 100 },
      });
    });
    expect(
      await screen.findByText("Instalando la actualización. No cierres la app."),
    ).toBeInTheDocument();
    expect(screen.getByText("Instalando…", { selector: "button" })).toBeDisabled();

    await act(async () => {
      resolveInstall(installResult());
    });
    expect(
      await screen.findByText("La actualización está lista. Reinicia para usar la versión 0.2.1."),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reiniciar ahora" }));
    expect(mocked.RestartToUpdate).toHaveBeenCalledTimes(1);
  });

  it("asks the user to reopen when restarting fails", async () => {
    mocked.RestartToUpdate.mockRejectedValue({
      problem: { code: "update_restart", message: "cannot start", detail: "permission denied" },
    });
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Actualizar ahora" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reiniciar ahora" }));

    expect(
      await within(dialog).findByText(
        "No se pudo reiniciar. Cierra y vuelve a abrir Better RPT.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("permission denied")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reiniciar ahora" })).toBeInTheDocument();
    expect(
      screen.getByText("No se pudo reiniciar. Cierra y vuelve a abrir Better RPT.", {
        selector: 'span[role="status"]',
      }),
    ).toBeInTheDocument();
  });

  it("shows a translated error with retry and a release link", async () => {
    mocked.CheckForUpdate.mockRejectedValue({
      problem: { code: "update_rate_limited", message: "rate limited", detail: "HTTP 403" },
    });
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));

    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    expect(
      within(dialog).getByText("GitHub ha limitado las peticiones. Espera un rato y reintenta."),
    ).toBeInTheDocument();
    expect(within(dialog).getByText("HTTP 403")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Reintentar" })).toBeInTheDocument();
    expect(within(dialog).getByRole("link", { name: "Abrir release" })).toHaveAttribute(
      "href",
      "https://github.com/srnoob2570/better-remote-play-together/releases/latest",
    );
    fireEvent.click(within(dialog).getByRole("button", { name: "Reintentar" }));
    await waitFor(() => expect(mocked.CheckForUpdate).toHaveBeenCalledTimes(2));
  });

  it("explains development builds without offering installation", async () => {
    mocked.CheckForUpdate.mockResolvedValue(
      updateCheck({
        available: false,
        canInstall: false,
        development: true,
        installKind: "development",
        latestVersion: "0.2.1",
      }),
    );
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));

    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    expect(within(dialog).getByText("Compilación de desarrollo")).toBeInTheDocument();
    expect(
      within(dialog).getByText(
        "Esta compilación no se actualiza automáticamente. Descarga la versión publicada desde GitHub.",
      ),
    ).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Actualizar ahora" })).not.toBeInTheDocument();
  });

  it("explains where a fallback AppImage was saved", async () => {
    mocked.InstallUpdate.mockResolvedValue(
      installResult({
        status: "saved",
        restart: false,
        path: "/home/test/Descargas/BetterRPT-0.2.1-x86_64.AppImage",
      }),
    );
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Actualizar ahora" }));

    expect(
      await screen.findByText(
        "El AppImage se guardó en /home/test/Descargas/BetterRPT-0.2.1-x86_64.AppImage. Ejecútalo para usar la versión 0.2.1.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reiniciar ahora" })).not.toBeInTheDocument();
  });

  it("opens the release page with the system browser", async () => {
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    fireEvent.click(within(dialog).getByRole("link", { name: "Ver notas del release" }));
    expect(Browser.OpenURL).toHaveBeenCalledWith(updateCheck().releaseUrl);
  });

  it("renders English copy and accessible status", async () => {
    mocked.CheckForUpdateOnStartup.mockResolvedValue(updateCheck());
    render(<UpdateControl lang="en" version="0.2.0" />);

    const button = await screen.findByRole("button", { name: "Update 0.2.1 available" });
    expect(button).toBeEnabled();
    expect(screen.getByRole("status")).toHaveTextContent("Update 0.2.1 available.");
    fireEvent.click(button);
    const dialog = await screen.findByRole("dialog", { name: "Better RPT update" });
    expect(within(dialog).getByText("Current version: 0.2.0")).toBeInTheDocument();
    expect(within(dialog).getByText("Version 0.2.1 is available.")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Update now" })).toBeInTheDocument();
  });

  it("moves focus into the dialog and supports Escape", async () => {
    render(<UpdateControl lang="es" version="0.2.0" />);
    fireEvent.click(screen.getByRole("button", { name: "Buscar actualizaciones" }));
    const dialog = await screen.findByRole("dialog", { name: "Actualización de Better RPT" });
    await waitFor(() => expect(dialog.contains(document.activeElement)).toBe(true));
    fireEvent.keyDown(dialog, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });
});
