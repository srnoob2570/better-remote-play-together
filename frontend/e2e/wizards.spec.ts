import { expect, test, type Page } from "@playwright/test";

async function open(page: Page, scenario: string, lang = "es") {
  await page.goto(`/e2e.html?e2e=${scenario}&lang=${lang}`);
}

test.describe("setup and repair wizards", () => {
  test("the setup wizard opens on a missing integration and can be resumed", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "setup-missing");

    const dialog = page.getByRole("dialog", { name: "Configuración inicial" });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText("Paso 2 de 6")).toBeVisible();
    await expect(dialog.getByRole("navigation", { name: "Pasos de configuración" })).toBeVisible();
    await expect(dialog.getByText("Steam y SpaceWar").first()).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-setup-1280.png" });

    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();

    await page.getByRole("button", { name: /Integración de SpaceWar/ }).click();
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Cerrar" }).click();
    await expect(dialog).toBeHidden();

    await page.setViewportSize({ width: 1024, height: 600 });
    await page.getByRole("button", { name: /Integración de SpaceWar/ }).click();
    await expect(dialog).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-setup-1024.png" });
  });

  test("the setup wizard orders backup before copy and exposes dependency commands", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "setup-missing");
    const dialog = page.getByRole("dialog", { name: "Configuración inicial" });
    await expect(dialog).toBeVisible();

    await dialog.getByRole("button", { name: /Comando de SpaceWar/ }).click();
    await expect(dialog.getByText("better-rpt-helper --donor -- %command%")).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Copiar comando" })).toBeDisabled();
    await dialog.getByRole("button", { name: "Guardar valor original" }).click();
    await expect(dialog.getByRole("button", { name: "Copiar comando" })).toBeEnabled();

    await dialog.getByRole("button", { name: /Dependencias/ }).click();
    await expect(dialog.getByText("sudo apt install fuse3")).toBeVisible();
    await expect(dialog.getByText("APPIMAGE_EXTRACT_AND_RUN=1 ./BetterRPT-0.2.0-x86_64.AppImage")).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-dependencies-1280.png" });
  });

  test("the repair wizard is read-only first and maps FUSE to a command", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await open(page, "library");
    await page.getByRole("button", { name: "Diagnóstico" }).click();
    await page.getByRole("button", { name: "Diagnóstico y reparación" }).click();

    const dialog = page.getByRole("dialog", { name: "Diagnóstico y reparación" });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText("Solo lectura", { exact: true })).toBeVisible();
    await expect(dialog.getByText("Problemas detectados (1)")).toBeVisible();
    await expect(dialog.getByRole("heading", { name: "FUSE" })).toBeVisible();
    await expect(dialog.getByText("Copia el comando y ejecútalo en tu terminal.")).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Copiar comando" })).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-repair-1280.png" });

    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();

    await page.setViewportSize({ width: 1024, height: 600 });
    await page.getByRole("button", { name: "Diagnóstico y reparación" }).click();
    await expect(dialog).toBeVisible();
    await page.waitForTimeout(300);
    await page.screenshot({ path: "/tmp/wizard-repair-1024.png" });
  });
});
