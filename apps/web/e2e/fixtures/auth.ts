import { test as base, type Page } from '@playwright/test';
import { loadAdmin, createTestCitizen, createTestCitizenInCommunity, type TestUser } from './users';

interface Fixtures {
  admin: TestUser;
  citizen: TestUser;
  citizenInCommunity: TestUser;
  authedAsCitizen: Page;
  authedAsCitizenInCommunity: Page;
  authedAsAdmin: Page;
}

// Extends the default test with admin + citizen users and pre-signed-in
// page contexts. Each spec that uses `authedAsX` gets a Page with the
// access token already injected into localStorage before first navigation
// — no interactive login required, and no dependency on the /login form.
export const test = base.extend<Fixtures>({
  admin: async ({}, use) => {
    await use(loadAdmin());
  },
  citizen: async ({}, use) => {
    // Global teardown handles the bulk purge (issues, petitions, then
    // users) in FK-safe order. Trying to delete the citizen inline here
    // would trip RESTRICT constraints from any content they authored.
    await use(createTestCitizen());
  },
  citizenInCommunity: async ({}, use) => {
    // Same reasoning as `citizen` above — global teardown handles it.
    await use(createTestCitizenInCommunity());
  },
  authedAsCitizen: async ({ browser, citizen }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await injectToken(page, citizen.accessToken);
    await use(page);
    await ctx.close();
  },
  authedAsCitizenInCommunity: async ({ browser, citizenInCommunity }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await injectToken(page, citizenInCommunity.accessToken);
    await use(page);
    await ctx.close();
  },
  authedAsAdmin: async ({ browser, admin }, use) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await injectToken(page, admin.accessToken);
    await use(page);
    await ctx.close();
  },
});

export const expect = test.expect;

// Puts the access token in localStorage BEFORE any script runs on the
// page, matching how the app itself boots (hasAccessToken() is checked at
// route-mount time).
async function injectToken(page: Page, token: string) {
  await page.addInitScript((t) => {
    localStorage.setItem('accessToken', t);
  }, token);
}
