import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

const wcag = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"];

test("library has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("update dialog has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=update-available&lang=es");
  await page.getByRole("button", { name: "Actualización 0.2.1 disponible" }).click();
  await expect(page.getByRole("dialog", { name: "Actualización de Better RPT" })).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("unknown-compatibility dialog has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=dialog-unknown&lang=es");
  await page.getByRole("button", { name: "Jugar" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("diagnostics has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
  await page.getByRole("button", { name: "Diagnóstico" }).click();
  await expect(page.getByText("No se encontró FUSE: faltan fusermount3 y /dev/fuse.")).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("setup wizard has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=setup-missing&lang=es");
  await expect(
    page.getByRole("region", { name: "Configuración inicial" }),
  ).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("repair wizard has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
  await page.getByRole("button", { name: "Diagnóstico" }).click();
  await page.getByRole("button", { name: "Diagnóstico y reparación" }).click();
  await expect(
    page.getByRole("region", { name: "Diagnóstico y reparación" }),
  ).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("session problem toast has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=session-error&lang=es");
  await expect(page.getByText("No se pudo completar la operación").last()).toBeVisible();

  const results = await new AxeBuilder({ page }).withTags(wcag).analyze();
  expect(results.violations).toEqual([]);
});

test("keyboard shortcuts and dialog focus stay inside the dialog", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

  const search = page.getByRole("searchbox", { name: "Buscar por nombre o App ID" });
  await page.keyboard.press("Control+f");
  await expect(search).toBeFocused();
  await page.keyboard.type("Valheim");
  await expect(page.getByRole("row", { name: /Valheim/ })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(search).toHaveValue("");

  await page.goto("/e2e.html?e2e=dialog-unknown&lang=es");
  await page.getByRole("button", { name: "Jugar" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  for (let i = 0; i < 6; i++) await page.keyboard.press("Tab");
  const inside = await page.evaluate(
    () => document.activeElement?.closest('[role="dialog"]') !== null,
  );
  expect(inside).toBe(true);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(page.getByRole("button", { name: "Jugar" })).toBeFocused();
});

test("compact header exposes every action and restores focus on Escape", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/e2e.html?e2e=library&lang=es");
  const menu = page.getByRole("button", { name: "Menú" });
  await menu.click();
  const panel = page.getByRole("group", { name: "Menú" });
  await expect(panel).toBeVisible();
  await expect(panel.getByRole("button", { name: "Diagnóstico" })).toBeVisible();
  await expect(
    panel.getByRole("button", { name: /Integración de SpaceWar/ }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(panel).toBeHidden();
  await expect(menu).toBeFocused();
});

test("update dialog survives a resize across the compact breakpoint", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/e2e.html?e2e=update-available&lang=es");
  await page.locator(".update-button").click();
  const dialog = page.getByRole("dialog", { name: "Actualización de Better RPT" });
  await expect(dialog).toBeVisible();

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(dialog).toBeVisible();

  await page.setViewportSize({ width: 1280, height: 800 });
  await expect(dialog).toBeVisible();
});

test("confirming the unknown-compatibility dialog focuses the session panel", async ({ page }) => {
  await page.goto("/e2e.html?e2e=dialog-unknown&lang=es");
  await page.getByRole("button", { name: "Jugar" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();

  await dialog.getByRole("button", { name: "Continuar" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByRole("tab", { name: "Sesión" })).toBeFocused();
});

test("update dialog returns focus to its trigger", async ({ page }) => {
  await page.goto("/e2e.html?e2e=update-available&lang=es");
  const trigger = page.locator(".update-button");
  await expect(trigger).toHaveAttribute(
    "aria-label",
    "Actualización 0.2.1 disponible",
  );
  await trigger.click();
  const dialog = page.getByRole("dialog", { name: "Actualización de Better RPT" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Cerrar" }).click();
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();
});
