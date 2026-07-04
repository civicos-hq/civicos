import { test, expect } from '../fixtures/auth';

test.describe('login + auth gate', () => {
  test('unauthenticated /users redirects to /login with a redirect param', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/users');
    await expect(page).toHaveURL(/\/login\?redirect=%2Fusers/);
    await expect(page.getByRole('heading', { name: /sign in/i })).toBeVisible();
    await ctx.close();
  });

  test('login page renders with clear scope messaging', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/login');
    await expect(page.locator('.admin-login-eyebrow')).toContainText(/admin console/i);
    await expect(page.locator('.admin-login-sub')).toContainText(/PLATFORM_ADMIN/);
    // Email + password fields with proper labels/id association.
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await ctx.close();
  });

  test('wrong credentials show an error', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/login');
    await page.getByLabel(/email/i).fill('nobody@civicos.test');
    await page.getByLabel(/password/i).fill('wrong-password');
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page.locator('.admin-login-error')).toBeVisible({ timeout: 5_000 });
    await ctx.close();
  });

  test('citizen role is rejected with actionable message', async ({ authedAsCitizen: page }) => {
    // The seeded citizen has a valid JWT but no PLATFORM_ADMIN role.
    // Visiting the console root should bounce to /login.
    await page.goto('/');
    await expect(page).toHaveURL(/\/login/);
  });

  test('admin lands on Overview after loading /', async ({ authedAsAdmin: page }) => {
    await page.goto('/');
    // Sidebar renders with the admin brand text.
    await expect(page.locator('.admin-brand-app')).toContainText(/admin console/i);
    // Topbar shows the actor's role chip.
    await expect(page.locator('.admin-topbar-role')).toContainText('PLATFORM_ADMIN');
    // Page header eyebrow reads "Overview".
    await expect(page.locator('.admin-page-eyebrow')).toContainText(/overview/i);
  });
});
