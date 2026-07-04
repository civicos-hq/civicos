import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

test.describe('users page', () => {
  test('search finds the seeded citizen', async ({ authedAsAdmin: page, citizen }) => {
    await page.goto('/users');
    await expect(page.locator('.admin-page-title')).toContainText(/user management/i);

    // Table renders (the admin user themselves is a row too).
    await expect(page.locator('.admin-table')).toBeVisible();

    // Filter to just our seeded citizen by email.
    await page.locator('.admin-table-search[placeholder*="Search email"]').fill(citizen.email);
    await expect(page.getByText(citizen.email).first()).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText(citizen.name).first()).toBeVisible();
  });

  test('role dropdown flip records to the audit log', async ({ authedAsAdmin: page, citizen }) => {
    await page.goto('/users');
    await page.locator('.admin-table-search[placeholder*="Search email"]').fill(citizen.email);

    // Wait for the row to render then use the row's Change role select.
    const row = page.locator('tr', { hasText: citizen.email });
    await expect(row).toBeVisible({ timeout: 5_000 });

    // Auto-confirm dialog (change-role prompts a confirm()).
    page.on('dialog', (d) => d.accept());
    await row.locator('select[aria-label*="Change role"]').selectOption('MODERATOR');

    // Refresh the search — the row's role chip should now read MODERATOR.
    await expect(row.locator('.admin-chip-role-MODERATOR')).toBeVisible({ timeout: 5_000 });

    // Audit trail has a user.role_changed entry from the admin.
    const auditCount = sql(
      `SELECT COUNT(*) FROM audit_logs WHERE target_id='${citizen.id}' AND action='user.role_changed';`,
    );
    expect(Number(auditCount)).toBe(1);
  });

  test('ban → unban cycle recorded twice in audit log', async ({
    authedAsAdmin: page,
    citizen,
  }) => {
    await page.goto('/users');
    await page.locator('.admin-table-search[placeholder*="Search email"]').fill(citizen.email);
    const row = page.locator('tr', { hasText: citizen.email });
    await expect(row).toBeVisible({ timeout: 5_000 });

    // Accept the prompt() for reason + confirm() for ban.
    let dialogCount = 0;
    page.on('dialog', (d) => {
      dialogCount += 1;
      if (d.type() === 'prompt') d.accept('e2e ban test');
      else d.accept();
    });
    await row.getByRole('button', { name: /^ban$/i }).click();

    // Row now shows the banned chip.
    await expect(row.locator('.admin-chip-banned')).toBeVisible({ timeout: 5_000 });

    // Unban.
    await row.getByRole('button', { name: /unban/i }).click();
    // Unban is idempotent — no prompts required by our UI, but the button
    // triggers the confirm() flow the same way. Wait for state to flip.
    await expect(row.locator('.admin-chip-banned')).toBeHidden({ timeout: 5_000 });

    // Both actions recorded.
    const bannedCount = sql(
      `SELECT COUNT(*) FROM audit_logs WHERE target_id='${citizen.id}' AND action='user.banned';`,
    );
    const unbannedCount = sql(
      `SELECT COUNT(*) FROM audit_logs WHERE target_id='${citizen.id}' AND action='user.unbanned';`,
    );
    expect(Number(bannedCount)).toBe(1);
    expect(Number(unbannedCount)).toBe(1);
    expect(dialogCount).toBeGreaterThan(0);
  });
});
