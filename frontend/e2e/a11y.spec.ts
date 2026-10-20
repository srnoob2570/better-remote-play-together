import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

const wcag = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"];

test("library has no WCAG A/AA violations", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

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
    page.getByRole("dialog", { name: "Configuración inicial" }),
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
    page.getByRole("dialog", { name: "Diagnóstico y reparación" }),
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
});
