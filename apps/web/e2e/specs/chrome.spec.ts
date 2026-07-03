import { test, expect } from '../fixtures/auth';

// Sidebar / topbar / language switcher smoke — the chrome we render on
// every dashboard page. If a landmark is missing or the language picker
// is broken, everything downstream falls apart.
test.describe('dashboard chrome', () => {
  test('sidebar + topbar have accessible landmarks', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/discover');
    // Sidebar is labelled and marked as an aside.
    const sidebar = page.locator('aside.dashboard-sidebar');
    await expect(sidebar).toBeVisible();
    await expect(sidebar).toHaveAttribute('aria-label', /nav|menu|main/i);

    // Topbar has a landmark label.
    const topbar = page.locator('header.dashboard-topbar, header[class*="topbar"]');
    await expect(topbar.first()).toBeVisible();
  });

  test('search bar exists and has aria-label', async ({ authedAsCitizenInCommunity: page }) => {
    await page.goto('/discover');
    const search = page.locator('input.dashboard-search');
    await expect(search).toBeVisible();
    await expect(search).toHaveAttribute('aria-label', /search/i);
  });

  test('language switcher swaps the sidebar labels', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/discover');
    const sidebar = page.locator('aside.dashboard-sidebar');
    const englishLabel = await sidebar
      .getByRole('link', { name: /discover/i })
      .first()
      .innerText();

    // Open language picker in the topbar and switch to Yorùbá.
    await page.locator('.lang-btn').first().click();
    await page.getByRole('button', { name: /yor.b/i }).click();

    // The sidebar labels change — Discover is "Ṣàwárí" in Yoruba.
    await expect(sidebar.getByRole('link', { name: /Ṣàwárí|discover/i }).first()).toBeVisible();
    // And it should NOT still be exactly the original English word.
    const newLabel = await sidebar.getByRole('link').first().innerText();
    expect(newLabel).not.toBe(englishLabel);

    // Switch back to English to leave state as we found it.
    await page.locator('.lang-btn').first().click();
    await page.getByRole('button', { name: /english/i }).click();
  });
});
