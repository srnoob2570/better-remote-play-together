import { expect, test } from "@playwright/test";

test("document language follows the active UI language", async ({ page }) => {
  await page.goto("/e2e.html?e2e=library&lang=en");
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");

  await page.locator(".language-trigger").click();
  await page.getByRole("menuitemradio", { name: "Español" }).click();
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "es");
});

test("keyboard reaches rows beyond the rendered window", async ({ page }) => {
  await page.goto("/e2e.html?e2e=many-games");
  await expect(page.getByRole("heading", { name: "Biblioteca" })).toBeVisible();

  const table = page.getByRole("table", { name: "Biblioteca" });
  await expect(table).toHaveAttribute("aria-rowcount", "81");

  // A single roving tab stop keeps the table out of an 80-stop tab chain.
  await expect(page.locator(".game-row[tabindex='0']")).toHaveCount(1);

  const first = page.locator(".game-row[tabindex='0']");
  await first.focus();
  await expect(first).toHaveAttribute("data-app-id", "900000");

  for (let step = 1; step <= 20; step++) {
    await page.keyboard.press("ArrowDown");
    await expect(page.locator(".game-row[tabindex='0']")).toHaveAttribute(
      "data-app-id",
      String(900000 + step),
    );
  }

  const state = await page.evaluate(() => {
    const focused = document.activeElement as HTMLElement | null;
    const scroller = document.querySelector(".table-scroll") as HTMLElement | null;
    return {
      appId: focused?.getAttribute("data-app-id"),
      tabStop: focused?.getAttribute("tabindex"),
      scrollTop: scroller?.scrollTop ?? 0,
    };
  });
  expect(state.appId).toBe("900020");
  expect(state.tabStop).toBe("0");
  expect(state.scrollTop).toBeGreaterThan(0);

  await page.keyboard.press("End");
  await expect(page.locator(".game-row[tabindex='0']")).toHaveAttribute(
    "data-app-id",
    "900079",
  );
  await expect(page.locator(".game-row[data-app-id='900079']")).toBeFocused();
});
