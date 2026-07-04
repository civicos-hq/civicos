import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

test.describe('audit log page', () => {
  test('renders recent entries with actor, action and metadata columns', async ({
    authedAsAdmin: page,
    admin,
  }) => {
    // Seed one audit entry so we're not depending on prior test state.
    const targetId = sql(`SELECT gen_random_uuid();`);
    const entryId = sql(`
      INSERT INTO audit_logs (id,actor_id,actor_name,actor_role,action,target_type,target_id,metadata,ip_address,user_agent,created_at)
      VALUES (gen_random_uuid(),'${admin.id}','${admin.name}','PLATFORM_ADMIN','audit.smoke','SMOKE_TARGET','${targetId}','{"note":"seeded by e2e"}','127.0.0.1','e2e-agent',now())
      RETURNING id;
    `);

    await page.goto('/audit');
    await expect(page.locator('.admin-page-title')).toContainText(/audit log/i);
    // Column headers present.
    await expect(page.locator('.admin-table thead')).toContainText(/actor/i);
    await expect(page.locator('.admin-table thead')).toContainText(/action/i);
    await expect(page.locator('.admin-table thead')).toContainText(/target/i);
    // Our seeded row is visible.
    await expect(page.getByText('audit.smoke').first()).toBeVisible({ timeout: 5_000 });

    // Cleanup.
    sql(`DELETE FROM audit_logs WHERE id='${entryId}';`);
  });

  test('filtering by action prefix narrows the list', async ({ authedAsAdmin: page, admin }) => {
    // Seed 2 rows with distinct action prefixes.
    const t1 = sql(`SELECT gen_random_uuid();`);
    const t2 = sql(`SELECT gen_random_uuid();`);
    const e1 = sql(`
      INSERT INTO audit_logs (id,actor_id,actor_name,actor_role,action,target_type,target_id,metadata,created_at)
      VALUES (gen_random_uuid(),'${admin.id}','${admin.name}','PLATFORM_ADMIN','filter_test.alpha','X','${t1}','{}',now())
      RETURNING id;
    `);
    const e2 = sql(`
      INSERT INTO audit_logs (id,actor_id,actor_name,actor_role,action,target_type,target_id,metadata,created_at)
      VALUES (gen_random_uuid(),'${admin.id}','${admin.name}','PLATFORM_ADMIN','filter_other.beta','Y','${t2}','{}',now())
      RETURNING id;
    `);

    await page.goto('/audit');
    // Filter by prefix "filter_test."
    await page.locator('input[placeholder*="Filter by action"]').fill('filter_test.');
    // Only the alpha row shows.
    await expect(page.getByText('filter_test.alpha')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText('filter_other.beta')).toHaveCount(0);

    sql(`DELETE FROM audit_logs WHERE id='${e1}';`);
    sql(`DELETE FROM audit_logs WHERE id='${e2}';`);
  });
});
