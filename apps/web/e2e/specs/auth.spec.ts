import { test, expect } from '../fixtures/auth';

// Auth-flow specs. We test the wrong-password error path and the
// citizen-lands-on-onboarding-then-dashboard flow via the pre-seeded
// storage state.
test.describe('auth', () => {
  test('logged-out user visiting /discover is redirected to /login', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/discover');
    await expect(page).toHaveURL(/\/login$/);
    // Login form is present.
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await page.close();
  });

  test('login form shows an error on wrong password', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/login');
    await page.getByLabel(/email/i).fill('nobody@civicos.test');
    await page.getByLabel(/password/i).fill('wrong-password');
    await page.getByRole('button', { name: /sign in|log in/i }).click();
    await expect(page.locator('.auth-error')).toBeVisible({ timeout: 5_000 });
    await page.close();
  });

  test('privacy link on login page navigates to /privacy', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/login');
    await page.locator('.auth-legal a[href="/privacy"]').click();
    await expect(page).toHaveURL(/\/privacy$/);
    await page.close();
  });

  test('authed citizen without a community sees the CommunityGate on /discover', async ({
    authedAsCitizen: page,
  }) => {
    // Discover page loads for authed users but shows the CommunityGate
    // notice until they pick a community from the sidebar/community page.
    await page.goto('/discover');
    await expect(page).toHaveURL(/\/discover$/);
    await expect(page.locator('.community-gate').first()).toBeVisible();
  });
});
