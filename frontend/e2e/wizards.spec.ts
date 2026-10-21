import { expect, test, type Page } from "@playwright/test";

async function open(page: Page, scenario: string, lang = "es") {
  await page.goto(`/e2e.html?e2e=${scenario}&lang=${lang}`);
}

test.describe("setup and repair wizards", () => {
  test("the setup wizard opens on a missing integration and can be resumed", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "setup-missing");

    const view = page.getByRole("region", { name: "Configuración inicial" });
    await expect(view).toBeVisible();
    await expect(view.getByText("Paso 2 de 6")).toBeVisible();
    await expect(view.getByRole("navigation", { name: "Pasos de configuración" })).toBeVisible();
    await expect(view.getByText("Steam y SpaceWar").first()).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-setup-1280.png" });

    await view.locator(".wizard-header").getByRole("button", { name: "Cerrar" }).click();
    await expect(view).toBeHidden();
    await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

    await page.getByRole("button", { name: /Integración de SpaceWar/ }).click();
    await expect(view).toBeVisible();
    await view.locator(".wizard-header").getByRole("button", { name: "Cerrar" }).click();
    await expect(view).toBeHidden();

    await page.setViewportSize({ width: 1024, height: 600 });
    await page.getByRole("button", { name: /Integración de SpaceWar/ }).click();
    await expect(view).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-setup-1024.png" });
  });

  test("the setup wizard orders backup before copy and exposes dependency commands", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "setup-missing");
    const view = page.getByRole("region", { name: "Configuración inicial" });
    await expect(view).toBeVisible();

    await view.getByRole("button", { name: /Comando de SpaceWar/ }).click();
    await expect(view.getByText("better-rpt-helper --donor -- %command%")).toBeVisible();
    await expect(view.getByRole("button", { name: "Copiar comando" })).toBeDisabled();
    await view.getByRole("button", { name: "Guardar valor original" }).click();
    await expect(view.getByRole("button", { name: "Copiar comando" })).toBeEnabled();

    await view.getByRole("button", { name: /Dependencias/ }).click();
    await expect(view.getByText("sudo apt install fuse3")).toBeVisible();
    await expect(view.getByText("APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.2.0-x86_64.AppImage")).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-dependencies-1280.png" });
  });

  test("the setup wizard opened from diagnostics returns to diagnostics", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "library");
    await page.getByRole("button", { name: "Diagnóstico" }).click();
    await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();

    await page.getByRole("button", { name: /Integración de SpaceWar/ }).click();
    const view = page.getByRole("region", { name: "Configuración inicial" });
    await expect(view).toBeVisible();
    await view.locator(".wizard-header").getByRole("button", { name: "Cerrar" }).click();
    const heading = page.getByRole("heading", { name: "Diagnóstico", exact: true });
    await expect(heading).toBeVisible();
    await expect(heading).toBeFocused();
  });

  test("the setup guide request is consumed once and does not stick after navigation", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "setup-missing");
    const view = page.getByRole("region", { name: "Configuración inicial" });
    await expect(view).toBeVisible();

    await view.getByRole("button", { name: /Comando de SpaceWar/ }).click();
    await view.getByRole("button", { name: "Abrir la guía completa" }).click();
    const guideTab = page.getByRole("tab", { name: "Guía" });
    await expect(guideTab).toHaveAttribute("aria-selected", "true");

    await page.getByRole("tab", { name: "Sesión" }).click();
    await page.getByRole("button", { name: "Diagnóstico" }).click();
    await page.getByRole("button", { name: "Volver a la biblioteca" }).click();
    await expect(page.getByRole("tab", { name: "Sesión" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  test("the repair wizard is read-only first and maps FUSE to a command", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "library");
    await page.getByRole("button", { name: "Diagnóstico" }).click();
    await page.getByRole("button", { name: "Diagnóstico y reparación" }).click();

    const view = page.getByRole("region", { name: "Diagnóstico y reparación" });
    await expect(view).toBeVisible();
    await expect(view.getByText("Solo lectura", { exact: true })).toBeVisible();
    await expect(view.getByText("Problemas detectados (1)")).toBeVisible();
    await expect(view.getByRole("heading", { name: "FUSE" })).toBeVisible();
    await expect(view.getByText("Copia el comando y ejecútalo en tu terminal.")).toBeVisible();
    await expect(view.getByRole("button", { name: "Copiar comando" })).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-repair-1280.png" });

    await view.locator(".wizard-header").getByRole("button", { name: "Cerrar" }).click();
    await expect(view).toBeHidden();
    await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();

    await page.setViewportSize({ width: 1024, height: 600 });
    await page.getByRole("button", { name: "Diagnóstico y reparación" }).click();
    await expect(view).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-repair-1024.png" });
  });
});
