import { expect, test, type Locator, type Page } from "@playwright/test";

type Box = { x: number; y: number; width: number; height: number };

async function box(locator: Locator): Promise<Box> {
  return locator.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
  });
}

function centerY(value: Box): number {
  return value.y + value.height / 2;
}

async function openGuide(page: Page): Promise<void> {
  await page.goto("/e2e.html?e2e=details&lang=es");
  await page.getByRole("tab", { name: "Guía" }).click();
  await expect(page.locator(".guide-command-block .guide-command")).toHaveText(
    "better-rpt-helper --donor -- %command%",
  );
}

test("guide keeps the command hint and copy action separated", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openGuide(page);

  const command = page.locator(".guide-command-block .guide-command");
  const hint = page.locator(".guide-command-block .hint");
  const copy = page.getByRole("button", { name: "Copiar comando" });
  const [commandBox, hintBox, copyBox] = await Promise.all([
    box(command),
    box(hint),
    box(copy),
  ]);

  expect(hintBox.y - (commandBox.y + commandBox.height)).toBeGreaterThanOrEqual(8);
  expect(copyBox.y - (hintBox.y + hintBox.height)).toBeGreaterThanOrEqual(8);
});

test("guide refresh action shares the command heading row", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await openGuide(page);

  const heading = page.getByRole("heading", { name: "Comando para SpaceWar" });
  const refresh = page.getByRole("button", { name: "Verificar de nuevo" });
  const copy = page.getByRole("button", { name: "Copiar comando" });
  const [headingBox, refreshBox, copyBox] = await Promise.all([
    box(heading),
    box(refresh),
    box(copy),
  ]);

  expect(Math.abs(centerY(headingBox) - centerY(refreshBox))).toBeLessThanOrEqual(2);
  expect(
    Math.abs(refreshBox.x + refreshBox.width - (copyBox.x + copyBox.width)),
  ).toBeLessThanOrEqual(1);
});

test("repair wizard stays centered inside the viewport", async ({ page }) => {
  const viewport = { width: 1280, height: 800 };
  await page.setViewportSize(viewport);
  await page.goto("/e2e.html?e2e=library&lang=es");
  await page.getByRole("button", { name: "Diagnóstico" }).click();
  await page.getByRole("button", { name: "Revisar" }).click();

  const dialog = page.locator('[data-slot="dialog-content"]');
  await expect(dialog).toBeVisible();
  await page.waitForTimeout(250);

  const dialogBox = await box(dialog);
  expect(Math.abs(dialogBox.x + dialogBox.width / 2 - viewport.width / 2)).toBeLessThanOrEqual(1);
  expect(Math.abs(dialogBox.y + dialogBox.height / 2 - viewport.height / 2)).toBeLessThanOrEqual(1);
  expect(dialogBox.x).toBeGreaterThanOrEqual(0);
  expect(dialogBox.y).toBeGreaterThanOrEqual(0);
  expect(dialogBox.x + dialogBox.width).toBeLessThanOrEqual(viewport.width);
  expect(dialogBox.y + dialogBox.height).toBeLessThanOrEqual(viewport.height);
});

test("library search and filter toolbar keep breathing room", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

  const heading = page.locator(".library-heading");
  const search = page.getByRole("searchbox", { name: "Buscar por nombre o App ID" });
  const toolbar = page.locator(".library-toolbar");
  const [headingBox, searchBox, toolbarBox] = await Promise.all([
    box(heading),
    box(search),
    box(toolbar),
  ]);

  expect(searchBox.y - (headingBox.y + headingBox.height)).toBeGreaterThanOrEqual(10);
  expect(toolbarBox.y - (searchBox.y + searchBox.height)).toBeGreaterThanOrEqual(4);
});

test("selection privacy and play action stay close", async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto("/e2e.html?e2e=library&lang=es");

  const privacy = page.locator(".session-body > .hint").filter({ hasText: "Privacidad:" });
  const play = page.getByRole("button", { name: "Jugar" });
  await expect(privacy).toBeVisible();
  await expect(play).toBeVisible();

  const [privacyBox, playBox] = await Promise.all([box(privacy), box(play)]);
  const gap = playBox.y - (privacyBox.y + privacyBox.height);
  expect(gap).toBeGreaterThanOrEqual(12);
  expect(gap).toBeLessThanOrEqual(24);
});

test("selection last session stays close to play action", async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto("/e2e.html?e2e=stale&lang=es");
  await page.getByRole("row", { name: /Resident Evil 2/ }).click();

  const play = page.getByRole("button", { name: "Jugar" });
  const last = page.locator(".last-compact");
  await expect(play).toBeVisible();
  await expect(last).toBeVisible();

  const [playBox, lastBox, labelBox] = await Promise.all([
    box(play),
    box(last),
    box(last.locator(".last-compact-label")),
  ]);
  const playBottom = playBox.y + playBox.height;
  expect(lastBox.y - playBottom).toBeLessThanOrEqual(20);
  expect(labelBox.y - playBottom).toBeLessThanOrEqual(36);
});

test("toast text and icons share one vertical center", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
  await page.getByRole("tab", { name: "Guía" }).click();
  await expect(page.locator(".guide-command-block .guide-command")).toHaveText(
    "better-rpt-helper --donor -- %command%",
  );
  await page.getByRole("button", { name: "Copiar comando" }).click();

  const toast = page.locator('[data-slot="toast"]').last();
  await expect(toast).toBeVisible();
  await expect(toast.locator('[data-slot="toast-title"]')).toHaveText("Copiado");
  await page.waitForTimeout(650);

  const content = toast.locator('[data-slot="toast-content"]');
  const icon = toast.locator('[data-slot="toast-icon"]');
  const title = toast.locator('[data-slot="toast-title"]');
  const close = toast.locator('[data-slot="toast-close"]');
  const [contentBox, iconBox, titleBox, closeBox] = await Promise.all([
    box(content),
    box(icon),
    box(title),
    box(close),
  ]);
  const contentCenter = centerY(contentBox);

  expect(Math.abs(centerY(iconBox) - contentCenter)).toBeLessThanOrEqual(1);
  expect(Math.abs(centerY(titleBox) - contentCenter)).toBeLessThanOrEqual(1);
  expect(Math.abs(centerY(closeBox) - contentCenter)).toBeLessThanOrEqual(1);
});

test("multi-line toast keeps its icon and close centered", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/e2e.html?e2e=session-error&lang=es");

  const toast = page.locator('[data-slot="toast"]').last();
  await expect(toast).toBeVisible();
  await expect(toast).toContainText("No se pudo completar la operación");
  await expect(toast.locator('[data-slot="toast-description"]')).toBeVisible();
  await page.waitForTimeout(650);

  const content = toast.locator('[data-slot="toast-content"]');
  const icon = toast.locator('[data-slot="toast-icon"]');
  const close = toast.locator('[data-slot="toast-close"]');
  const [contentBox, iconBox, closeBox] = await Promise.all([
    box(content),
    box(icon),
    box(close),
  ]);
  const contentCenter = centerY(contentBox);

  expect(Math.abs(centerY(iconBox) - contentCenter)).toBeLessThanOrEqual(1);
  expect(Math.abs(centerY(closeBox) - contentCenter)).toBeLessThanOrEqual(1);
});
