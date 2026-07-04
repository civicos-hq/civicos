import { test as base, type Page } from '@playwright/test';
import { loadAdmin, createTestCitizen, type TestUser } from './users';

interface Fixtures {
  admin: TestUser;
  citizen: TestUser;
  authedAsAdmin: Page;
  authedAsCitizen: Page;
}

// Admin console stores its session under two keys — the access token
// and the user shape. Both must be present or getSession() returns null
// and RequireAdmin bounces to /login.
export const test = base.extend<Fixtures>({
  admin: async ({}, use) => {
    await use(loadAdmin());
  },
  citizen: async ({}, use) => {
    await use(createTestCitizen());
  },
  authedAsAdmin: async ({ browser, admin }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await injectAdminSession(page, admin);
    await use(page);
    await ctx.close();
  },
  authedAsCitizen: async ({ browser, citizen }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    // Citizens can hold a JWT but the admin console gates on role.
    // Injecting a citizen session tests the "wrong role bounce."
    await injectAdminSession(page, citizen);
    await use(page);
    await ctx.close();
  },
});

export const expect = test.expect;

async function injectAdminSession(page: Page, user: TestUser) {
  await page.addInitScript(
    ({ token, u }) => {
      localStorage.setItem('civicos-admin-token', token);
      localStorage.setItem(
        'civicos-admin-user',
        JSON.stringify({
          id: u.id,
          email: u.email,
          name: u.name,
          role: u.role,
          emailVerified: u.emailVerified,
        }),
      );
    },
    { token: user.accessToken, u: user },
  );
}
