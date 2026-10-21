import { expect, test, type Page } from "@playwright/test";

const scenarios = [
  "library",
  "loading",
  "empty",
  "error",
  "preparing",
  "active",
  "ended",
  "stale",
  "donor-missing",
  "incompatible",
  "details",
  "dialog-unknown",
  "session-error",
  "update-available",
  "update-development",
  "start-pending",
] as const;

type Scenario = (typeof scenarios)[number];

async function prepare(page: Page, scenario: Scenario, lang: "es" | "en" = "es") {
  await page.clock.install({ time: new Date("2026-10-20T12:00:00Z") });
  await page.goto(`/e2e.html?e2e=${scenario}&lang=${lang}`);
  await page.waitForTimeout(250);
  if (scenario === "details") {
    await page.getByRole("tab", { name: lang === "es" ? "Detalles" : "Details" }).click();
    await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
  }
  if (scenario === "dialog-unknown") {
    await page.getByRole("button", { name: lang === "es" ? "Jugar" : "Play" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
  }
  if (scenario === "update-available" || scenario === "update-development") {
    await page
      .getByRole("button", {
        name: lang === "es" ? "Actualización 0.2.1 disponible" : "Update 0.2.1 available",
      })
      .click();
    await expect(page.getByRole("dialog")).toBeVisible();
  }
  if (scenario === "start-pending") {
    await page.getByRole("row", { name: /Valheim/ }).click();
    await page.getByRole("button", { name: lang === "es" ? "Jugar" : "Play" }).click();
  }
}

const expected = (scenario: Scenario, lang: "es" | "en"): RegExp | string => {
  const es = lang === "es";
  switch (scenario) {
    case "library":
      return es ? "Biblioteca" : "Library";
    case "loading":
      return es ? "Leyendo la biblioteca de Steam…" : "Reading the Steam library…";
    case "empty":
      return es ? "No hay juegos instalados." : "No games are installed.";
    case "error":
      return es ? "No se pudo leer la biblioteca de Steam." : "The Steam library could not be read.";
    case "preparing":
      return es ? "Preparando la sesión" : "Preparing the session";
    case "active":
      return es ? "Sesión activa" : "Session active";
    case "ended":
      return es ? "Última sesión" : "Last session";
    case "stale":
      return es ? "Configuración anterior" : "Previous configuration";
    case "donor-missing":
      return es ? "SpaceWar debe estar instalado para iniciar la sesión." : "SpaceWar must be installed to start the session.";
    case "incompatible":
      return "ELDEN RING";
    case "details":
      return "Proton 11.0";
    case "dialog-unknown":
      return es ? "Compatibilidad desconocida" : "Unknown compatibility";
    case "session-error":
      return es ? "No se pudo completar la operación" : "The operation could not be completed";
    case "update-available":
    case "update-development":
      return es ? "Actualización de Better RPT" : "Better RPT update";
    case "start-pending":
      return "Valheim";
  }
};

async function assertReady(page: Page, scenario: Scenario, lang: "es" | "en") {
  if (scenario === "loading") {
    await expect(page.locator('.library-table[aria-busy="true"]')).toBeVisible();
    return;
  }
  await expect(page.getByText(expected(scenario, lang), { exact: false }).first()).toBeVisible();
}

test.describe("visual parity", () => {
  for (const scenario of scenarios) {
    test(`${scenario} at 1280×800`, async ({ page }) => {
      await page.setViewportSize({ width: 1280, height: 800 });
      await prepare(page, scenario, "es");
      await assertReady(page, scenario, "es");
      await expect(page).toHaveScreenshot(`visual/${scenario}-1280x800.png`);
    });
  }

  for (const scenario of ["library", "preparing", "active", "ended"] as const) {
    for (const size of [
      { width: 1024, height: 600, name: "1024x600" },
      { width: 1920, height: 1080, name: "1920x1080" },
    ]) {
      test(`${scenario} at ${size.name}`, async ({ page }) => {
        await page.setViewportSize({ width: size.width, height: size.height });
        await prepare(page, scenario, "es");
        await assertReady(page, scenario, "es");
        await expect(page).toHaveScreenshot(`visual/${scenario}-${size.name}.png`);
      });
    }
  }

  test("library in English", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await prepare(page, "library", "en");
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
    await expect(page).toHaveScreenshot("visual/library-en-1280x800.png");
  });

  test("library with focused search", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await prepare(page, "library", "es");
    await page.getByRole("searchbox", { name: "Buscar por nombre o App ID" }).focus();
    await expect(page).toHaveScreenshot("visual/library-focus-1280x800.png");
  });

  test("library with hovered row", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await prepare(page, "library", "es");
    await page.getByRole("row", { name: /Valheim/ }).hover();
    await expect(page).toHaveScreenshot("visual/library-hover-1280x800.png");
  });

  for (const size of [
    { width: 1280, height: 800, name: "1280x800" },
    { width: 1024, height: 600, name: "1024x600" },
    { width: 1920, height: 1080, name: "1920x1080" },
  ]) {
    test(`diagnostics at ${size.name}`, async ({ page }) => {
      await page.setViewportSize({ width: size.width, height: size.height });
      await prepare(page, "library", "es");
      await page.getByRole("button", { name: "Diagnóstico" }).click();
      await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();
      await expect(page.getByText("No se encontró FUSE: faltan fusermount3 y /dev/fuse.")).toBeVisible();
      await expect(page).toHaveScreenshot(`visual/diagnostics-${size.name}.png`);
    });
  }

  for (const size of [
    { width: 1280, height: 800, name: "1280x800" },
    { width: 1024, height: 600, name: "1024x600" },
  ]) {
    test(`guide at ${size.name}`, async ({ page }) => {
      await page.setViewportSize({ width: size.width, height: size.height });
      await prepare(page, "details", "es");
      await page.getByRole("tab", { name: "Guía" }).click();
      await expect(page.getByRole("heading", { name: "Comando para SpaceWar" })).toBeVisible();
      await expect(page.locator(".guide-command-block .guide-command")).toHaveText(
        "better-rpt-helper --donor -- %command%",
      );
      await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
      await page.waitForTimeout(100);
      await expect(page).toHaveScreenshot(`visual/guide-${size.name}.png`);
    });
  }
});
