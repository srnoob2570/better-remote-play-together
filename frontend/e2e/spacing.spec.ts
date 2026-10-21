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
  const refresh = page
    .locator(".guide-command-block")
    .getByRole("button", { name: "Actualizar" });
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

test("repair wizard opens as a modal over the app", async ({ page }) => {
  const viewport = { width: 1280, height: 800 };
  await page.setViewportSize(viewport);
  await page.goto("/e2e.html?e2e=library&lang=es");
  await page.getByRole("button", { name: "Diagnóstico" }).click();
  await page.getByRole("button", { name: "Revisar" }).click();

  const overlay = page.locator('[data-slot="dialog-overlay"]');
  const dialog = page.getByRole("dialog", { name: "Diagnóstico y reparación" });
  const view = page.getByRole("region", { name: "Diagnóstico y reparación" });
  await expect(dialog).toBeVisible();
  await expect(view).toBeVisible();
  await expect(
    view
      .locator(".wizard-header")
      .getByRole("button", { name: "Actualizar" })
      .locator("svg.lucide-refresh-cw"),
  ).toBeVisible();
  await page.waitForTimeout(250);

  const [overlayBox, viewBox] = await Promise.all([box(overlay), box(view)]);
  expect(overlayBox).toEqual({ ...viewport, x: 0, y: 0 });
  expect(viewBox.x).toBeGreaterThan(0);
  expect(viewBox.y).toBeGreaterThan(0);
  expect(viewBox.x + viewBox.width).toBeLessThan(viewport.width);
  expect(viewBox.y + viewBox.height).toBeLessThan(viewport.height);
  expect(viewBox.width).toBeLessThanOrEqual(1280);
  expect(Math.abs(viewBox.x + viewBox.width / 2 - viewport.width / 2)).toBeLessThanOrEqual(1);
  expect(Math.abs(viewBox.y + viewBox.height / 2 - viewport.height / 2)).toBeLessThanOrEqual(1);

  await page.keyboard.press("Escape");
  await expect(view).toBeHidden();
  await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeFocused();
});

test("library search and filter toolbar keep breathing room", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/e2e.html?e2e=library&lang=es");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

  const heading = page.locator(".library-heading");
  const search = page.getByRole("searchbox", { name: "Buscar por nombre o App ID" });
  const toolbar = page.locator(".library-toolbar");
  await expect(
    toolbar
      .getByRole("button", { name: "Actualizar" })
      .locator("svg.lucide-refresh-cw"),
  ).toBeVisible();
  const [headingBox, searchBox, toolbarBox] = await Promise.all([
    box(heading),
    box(search),
    box(toolbar),
  ]);

  expect(searchBox.y - (headingBox.y + headingBox.height)).toBeGreaterThanOrEqual(10);
  expect(toolbarBox.y - (searchBox.y + searchBox.height)).toBeGreaterThanOrEqual(4);
});

test("selection play action sits in the title block above privacy guidance", async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto("/e2e.html?e2e=library&lang=es");

  const title = page.locator(".session-body .game-title");
  const meta = page.locator(".session-body .game-meta");
  const privacy = page.locator(".session-body > .hint").filter({ hasText: "Privacidad:" });
  const play = page.getByRole("button", { name: "Jugar" });
  await expect(title).toBeVisible();
  await expect(meta).toBeVisible();
  await expect(privacy).toBeVisible();
  await expect(play).toBeVisible();

  const [titleBox, metaBox, privacyBox, playBox] = await Promise.all([
    box(title),
    box(meta),
    box(privacy),
    box(play),
  ]);
  expect(playBox.y).toBeGreaterThan(titleBox.y + titleBox.height);
  expect(playBox.y - (metaBox.y + metaBox.height)).toBeLessThanOrEqual(24);
  expect(playBox.y + playBox.height).toBeLessThanOrEqual(privacyBox.y);
});

test("selection play action is visible without scrolling at 390px", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/e2e.html?e2e=library&lang=es");

  const play = page.getByRole("button", { name: "Jugar" });
  await expect(play).toBeVisible();
  await expect(play).toBeInViewport();

  const [playBox, dockBox] = await Promise.all([box(play), box(page.locator(".app-dock"))]);
  expect(playBox.y + playBox.height).toBeLessThanOrEqual(dockBox.y);
});

test("compact controls meet the 44px touch target", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/e2e.html?e2e=library&lang=es");
  await page.locator(".compact-menu-button").click();

  const controls = [
    ".compact-menu-button",
    ".setup-indicator",
    ".diagnostics-button",
    ".update-button",
    ".language-trigger",
    ".search-field",
    ".refresh-button",
    ".session-tab",
  ];
  for (const selector of controls) {
    const control = page.locator(selector).first();
    await expect(control).toBeVisible();
    expect((await box(control)).height).toBeGreaterThanOrEqual(44);
  }
  for (const filter of await page.locator(".filter-button").all()) {
    expect((await box(filter)).height).toBeGreaterThanOrEqual(44);
  }
});

test("diagnostics controls meet the 44px touch target at 390px", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/e2e.html?e2e=library&lang=es");
  await page.locator(".compact-menu-button").click();
  await page.getByRole("button", { name: "Diagnóstico" }).click();
  await expect(page.getByRole("heading", { name: "Diagnóstico", exact: true })).toBeVisible();

  const controls = [
    ".diagnostics-back",
    ".diagnostic-repair",
    ".diagnostic-docs a",
    ".diagnostics-actions .refresh-button",
  ];
  for (const selector of controls) {
    const control = page.locator(selector).first();
    await expect(control).toBeVisible();
    expect((await box(control)).height).toBeGreaterThanOrEqual(44);
  }
});

test("selection last session remains visible below the primary action", async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto("/e2e.html?e2e=stale&lang=es");
  await page.getByRole("row", { name: /Resident Evil 2/ }).click();

  const play = page.getByRole("button", { name: "Jugar" });
  const last = page.locator(".last-compact");
  await expect(play).toBeVisible();
  await expect(last).toBeVisible();

  const [playBox, lastBox] = await Promise.all([box(play), box(last)]);
  expect(playBox.y).toBeLessThan(lastBox.y);
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

test("loading button keeps its label and shows a spinner", async ({ page }) => {
  await page.goto("/e2e.html?e2e=loading&lang=es");

  const refresh = page.getByRole("button", { name: "Actualizar" });
  await expect(refresh).toBeDisabled();
  await expect(refresh.locator('[data-slot="spinner"]')).toBeVisible();
  await expect(refresh).toContainText("Actualizar");
});

test("launching button keeps its label and shows a spinner", async ({ page }) => {
  await page.goto("/e2e.html?e2e=start-pending&lang=es");
  await page.getByRole("row", { name: /Valheim/ }).click();
  const play = page.getByRole("button", { name: "Jugar" });
  await play.click();

  await expect(play).toBeDisabled();
  await expect(play.locator('[data-slot="spinner"]')).toBeVisible();
  await expect(play).toContainText("Jugar");
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
