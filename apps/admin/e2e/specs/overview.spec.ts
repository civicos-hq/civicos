import { test, expect } from '../fixtures/auth';

test.describe('overview + sidebar navigation', () => {
  test('health strip pings all 4 services and shows their state', async ({
    authedAsAdmin: page,
  }) => {
    await page.goto('/');
    // Each health item renders a dot + service name.
    const items = page.locator('.admin-health-item');
    await expect(items).toHaveCount(4, { timeout: 10_000 });
    // All 4 backend services should read "ok" since global-setup verified them.
    for (const name of ['gateway', 'identity', 'community', 'organization']) {
      await expect(page.locator('.admin-health-item').filter({ hasText: name })).toContainText(
        /ok/i,
      );
    }
  });

  test('overview stat grid renders 4 counter cards', async ({ authedAsAdmin: page }) => {
    await page.goto('/');
    await expect(page.locator('.admin-stat-card')).toHaveCount(4);
    // Card labels present.
    await expect(page.getByText(/pending flags/i)).toBeVisible();
    await expect(page.getByText(/hidden \(all time\)/i)).toBeVisible();
    await expect(page.getByText(/dismissed/i)).toBeVisible();
    await expect(page.getByText(/audit log entries/i)).toBeVisible();
  });

  test('every sidebar link navigates without console errors', async ({ authedAsAdmin: page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto('/');

    const links: Array<[string, RegExp]> = [
      ['/users', /user management/i],
      ['/organizations', /organization management/i],
      ['/flags', /moderation queue/i],
      ['/audit', /audit log/i],
      ['/', /platform overview/i],
    ];

    for (const [href, titleRe] of links) {
      await page.locator(`.admin-nav-link[href="${href}"]`).click();
      await expect(page).toHaveURL(new RegExp(href.replace('/', '\\/') + '$'));
      await expect(page.locator('.admin-page-title')).toContainText(titleRe);
    }

    const noise = consoleErrors.filter(
      (e) => !e.includes('401') && !e.includes('Failed to load resource'),
    );
    expect(noise, `unexpected console errors:\n${noise.join('\n')}`).toEqual([]);
  });

  test('sign-out clears the session and bounces to login', async ({ authedAsAdmin: page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /sign out/i }).click();
    await expect(page).toHaveURL(/\/login/);
    // Session was actually cleared — no token in localStorage.
    const token = await page.evaluate(() => localStorage.getItem('civicos-admin-token'));
    expect(token).toBeNull();
  });
});
