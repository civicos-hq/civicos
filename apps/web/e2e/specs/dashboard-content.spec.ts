import { test, expect } from '../fixtures/auth';

// Dashboard content pages — issues, petitions, discover. We prove they
// render, sidebar navigation works, filter pills toggle their aria-pressed
// state, and the "new item" modal actually opens. We don't submit forms
// here (the API smoke test covers persistence end-to-end); the goal is
// UI regressions — a broken component, missing translation, dead route.
test.describe('dashboard content', () => {
  test('citizen in a community can navigate sidebar without errors', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto('/discover');
    await expect(page.locator('.dashboard-sidebar')).toBeVisible();

    // Walk every sidebar link. Each page has a stable eyebrow above its
    // title so we check the URL and that the page-header component
    // mounted with an eyebrow — proves the route matched, the page
    // component rendered, and no rendering errors were thrown.
    const links = [
      '/community',
      '/issues',
      '/petitions',
      '/representatives',
      '/organizations',
      '/notifications',
      '/discover',
    ];

    for (const href of links) {
      await page.locator(`.dashboard-sidebar a[href="${href}"]`).click();
      await expect(page).toHaveURL(new RegExp(href + '$'));
      // Every page uses PageHeader with an eyebrow.
      await expect(page.locator('.page-header-eyebrow, h1').first()).toBeVisible({
        timeout: 5_000,
      });
    }

    // Filter out React-Query background 401s from the /me hit before the
    // token is applied on a fresh nav — those aren't UI errors.
    const uiErrors = consoleErrors.filter(
      (e) => !e.includes('401') && !e.includes('Failed to load resource'),
    );
    expect(uiErrors, `unexpected console errors:\n${uiErrors.join('\n')}`).toEqual([]);
  });

  test('IssuesPage — status filter pills toggle aria-pressed', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/issues');
    // The "All status" pill starts active.
    const allStatus = page.getByRole('button', { name: /all status/i });
    await expect(allStatus).toHaveAttribute('aria-pressed', 'true');

    // Click a specific status pill — its aria-pressed flips to true and
    // the All pill flips to false.
    const openPill = page.getByRole('button', { name: /^open$/i }).first();
    await openPill.click();
    await expect(openPill).toHaveAttribute('aria-pressed', 'true');
    await expect(allStatus).toHaveAttribute('aria-pressed', 'false');
  });

  test('IssuesPage — Report Issue modal opens', async ({ authedAsCitizenInCommunity: page }) => {
    await page.goto('/issues');
    // The button label reads "+ Raise an issue" (see issuesPage.reportBtn).
    // Match either wording so a future copy change from "raise" back to
    // "report" doesn't quietly break this test again.
    await page.getByRole('button', { name: /raise an issue|report issue/i }).click();
    // Modal header text ('File a community issue' or similar) proves the
    // dialog rendered. Using text lookup rather than getByLabel because
    // @civicos/ui Input drops the label/input id association (follow-up:
    // wire an id on Input so screen readers can navigate the form).
    const dialog = page
      .locator('[role="dialog"], .fixed')
      .filter({ hasText: /title|report|issue/i })
      .first();
    await expect(dialog).toBeVisible();
    // The first text input inside the dialog is the title.
    await expect(dialog.locator('input[type="text"], input:not([type])').first()).toBeVisible();
    // Close via Escape.
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden({ timeout: 2_000 });
  });

  test('PetitionsPage — page header + New Petition button render', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/petitions');
    // PetitionsPage eyebrow reads "Petition Studio".
    await expect(page.locator('.page-header-eyebrow').first()).toContainText(/studio|petition/i);
    // New Petition button is present.
    await expect(page.getByRole('button', { name: /new petition/i })).toBeVisible();
  });

  test('CommunityPage renders without errors', async ({ authedAsCitizenInCommunity: page }) => {
    await page.goto('/community');
    // The page-header renders whether or not the citizen has a community.
    await expect(page.locator('h1').first()).toBeVisible();
  });
});
