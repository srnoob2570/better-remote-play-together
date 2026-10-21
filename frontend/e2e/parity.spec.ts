import { expect, test, type Page } from "@playwright/test";

async function open(page: Page, scenario: string, lang = "es") {
  await page.goto(`/e2e.html?e2e=${scenario}&lang=${lang}`);
}

test.describe("parity states", () => {
  test("library shows the catalog, filters and search", async ({ page }) => {
    await open(page, "library");
    await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
    await expect(page.getByText("5 juegos")).toBeVisible();
    await expect(page.getByRole("row", { name: /HELLDIVERS/ })).toBeVisible();
    await expect(page.getByRole("row", { name: /PlateUp/ })).toBeVisible();

    await page.getByRole("radio", { name: "Listos" }).click();
    await expect(page.getByRole("row", { name: /Valheim/ })).toBeVisible();
    await expect(page.getByRole("row", { name: /PlateUp/ })).toBeHidden();
    await expect(page.getByText("1 juego")).toBeVisible();

    await page.getByRole("radio", { name: "Todos" }).click();
    const search = page.getByRole("searchbox", { name: "Buscar por nombre o App ID" });
    await search.fill("1599600");
    await expect(page.getByRole("row", { name: /PlateUp/ })).toBeVisible();
    await search.fill("zzzz");
    await expect(page.getByText("Ningún juego coincide con la búsqueda.")).toBeVisible();
  });

  test("details and guide tabs expose the selected game", async ({ page }) => {
    await open(page, "details");
    await expect(page.getByRole("row", { name: /HELLDIVERS/ })).toBeVisible();
    await page.getByRole("tab", { name: "Detalles" }).click();
    await expect(page.getByText("Proton 11.0")).toBeVisible();
    await expect(page.getByText("App ID")).toBeVisible();

    await page.getByRole("tab", { name: "Guía" }).click();
    await expect(page.getByText("Comando para SpaceWar")).toBeVisible();
    await expect(page.getByText("better-rpt-helper --donor -- %command%")).toBeVisible();
  });

  test("guide images keep their 2.5:1 aspect ratio", async ({ page }) => {
    await open(page, "details");
    await page.getByRole("tab", { name: "Guía" }).click();
    const images = page.locator(".guide-image");
    await expect(images).toHaveCount(5);

    for (let index = 0; index < 5; index += 1) {
      const box = await images.nth(index).boundingBox();
      expect(box).not.toBeNull();
      const ratio = box!.width / box!.height;
      expect(ratio).toBeGreaterThan(2.45);
      expect(ratio).toBeLessThan(2.55);
    }
  });

  test("empty and error states", async ({ page }) => {
    await open(page, "empty");
    await expect(page.getByText("No hay juegos instalados.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Abrir Steam" })).toBeVisible();

    await open(page, "error");
    await expect(page.getByText("No se pudo leer la biblioteca de Steam.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Reintentar" })).toBeVisible();
  });

  test("preparing progresses and can be cancelled", async ({ page }) => {
    await open(page, "preparing");
    await expect(page.getByText("Preparando la sesión")).toBeVisible();
    await expect(page.getByText("Paso 1 de 5")).toBeVisible();
    await expect(
      page.getByText("Steam puede pedirte que confirmes la ejecución.", { exact: false }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Cancelar" }).click();
    await expect(page.getByRole("button", { name: "Jugar" })).toBeVisible();
  });

  test("active session shows timer and invite guide", async ({ page }) => {
    await open(page, "active");
    await expect(page.getByRole("heading", { name: "Sesión activa", exact: true })).toBeVisible();
    await expect(page.getByText("Abre el overlay de Steam con Shift+Tab.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sesión activa" })).toBeDisabled();
  });

  test("ended session restores SpaceWar", async ({ page }) => {
    await open(page, "ended");
    await expect(page.getByText("Última sesión")).toBeVisible();
    await page.getByRole("button", { name: "Restaurar SpaceWar" }).click();
    const dialog = page.getByRole("dialog", { name: "Restaurar SpaceWar" });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText("Restaurar SpaceWar")).toBeVisible();
    await dialog.getByRole("button", { name: "Restaurar", exact: true }).click();
    await expect(dialog).toBeHidden();
  });

  test("stale configuration offers restore", async ({ page }) => {
    await open(page, "stale");
    await expect(page.getByText("Configuración anterior")).toBeVisible();
    await expect(page.getByRole("button", { name: "Restaurar SpaceWar" })).toBeVisible();
  });

  test("missing donor and incompatible game disable launch", async ({ page }) => {
    await open(page, "donor-missing");
    await expect(page.getByText("SpaceWar debe estar instalado para iniciar la sesión.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Añadir SpaceWar" })).toBeVisible();

    await open(page, "incompatible");
    await expect(page.getByRole("heading", { name: "ELDEN RING" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Incompatible" })).toBeDisabled();
  });

  test("unknown compatibility warning is shown once and starts a session", async ({ page }) => {
    await open(page, "dialog-unknown");
    await page.getByRole("button", { name: "Jugar" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText("Compatibilidad desconocida")).toBeVisible();
    await dialog.getByRole("button", { name: "Continuar" }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText("Preparando la sesión")).toBeVisible();
  });

  test("session error keeps the technical detail available", async ({ page }) => {
    await open(page, "session-error");
    const panel = page.getByRole("tabpanel", { name: "Sesión" });
    await expect(
      panel.getByRole("heading", { name: "No se pudo completar la operación" }),
    ).toBeVisible();
    await expect(
      panel.getByText(
        "La configuración de SpaceWar cambió fuera de Better RPT. No se sobrescribió.",
      ),
    ).toBeVisible();
    await expect(panel.getByText("Detalle técnico")).toBeVisible();
  });

  test("language switch updates the whole UI", async ({ page }) => {
    await open(page, "library");
    await page.getByRole("button", { name: "Idioma" }).click();
    await page.getByRole("menuitemradio", { name: "English" }).click();
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
    await expect(page.getByText("5 games")).toBeVisible();
    await expect(page.getByRole("tab", { name: "Session" })).toBeVisible();
  });

  test("language switch does not re-announce an existing error", async ({ page }) => {
    await open(page, "session-error", "en");
    const toasts = page.locator('[data-slot="toast"]');
    await expect(toasts).toContainText("The operation could not be completed");

    await page.locator(".language-trigger").click();
    await page.getByRole("menuitemradio", { name: "Español" }).click();
    await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

    // The existing toast may live or expire, but the switch must not raise it again.
    await page.waitForTimeout(300);
    expect(
      await toasts.filter({ hasText: "No se pudo completar la operación" }).count(),
    ).toBe(0);
  });

  test("keyboard shortcuts focus and clear search", async ({ page }) => {
    await open(page, "library");
    await page.keyboard.press("Control+f");
    const search = page.getByRole("searchbox", { name: "Buscar por nombre o App ID" });
    await expect(search).toBeFocused();
    await search.fill("Valheim");
    await page.keyboard.press("Escape");
    await expect(search).toHaveValue("");
  });

  test("diagnostics shows the report and returns to the library", async ({ page }) => {
    await open(page, "library");
    await page.getByRole("button", { name: "Diagnóstico" }).click();
    await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();
    await expect(page.getByText("No se encontró FUSE: faltan fusermount3 y /dev/fuse.")).toBeVisible();
    await expect(page.getByRole("link", { name: "docs/appimage.md" })).toHaveAttribute(
      "href",
      "https://github.com/srnoob2570/better-remote-play-together/blob/main/docs/appimage.md",
    );
    await page.getByRole("button", { name: "Volver a la biblioteca" }).click();
    await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
  });
});
