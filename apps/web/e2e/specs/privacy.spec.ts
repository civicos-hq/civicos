import { test, expect } from '../fixtures/auth';

// Privacy notice — public page, must render all 12 clauses and preserve
// the mailto: anchors after the <Trans> component slot substitution.
test.describe('privacy page', () => {
  test('renders all 12 clauses with mailto anchors', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/privacy');

    await expect(page.locator('.privacy-title')).toContainText(/privacy notice/i);
    await expect(page.locator('.privacy-effective')).toContainText(/effective/i);

    // All 12 clauses numbered §01…§12.
    const clauses = page.locator('.privacy-clause');
    expect(await clauses.count()).toBe(12);
    for (let i = 1; i <= 12; i++) {
      const label = `§ ${String(i).padStart(2, '0')}`;
      await expect(page.locator('.privacy-clause-num').nth(i - 1)).toContainText(label);
    }

    // The <mail> and <security> Trans slots resolved to real anchors.
    await expect(page.locator('a[href="mailto:privacy@civicos.ng"]').first()).toBeVisible();
    await expect(page.locator('a[href="mailto:security@civicos.ng"]').first()).toBeVisible();

    await page.close();
  });

  test('nav back to homepage works from privacy', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/privacy');
    // TopNav Docket link now goes to /#docket — should navigate not just anchor.
    await page
      .getByRole('link', { name: /docket/i })
      .first()
      .click();
    await expect(page).toHaveURL(/\/#docket$/);
    await page.close();
  });
});
