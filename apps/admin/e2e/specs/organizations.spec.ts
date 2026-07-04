import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Seed a fresh unverified org before the test, verify it via the UI,
// confirm the DB reflects the change. Uses the admin-e2e- slug prefix
// so global teardown cleans it up.
async function seedOrg(admin: { id: string }): Promise<{ id: string; name: string; slug: string }> {
  const slug = `admin-e2e-${Date.now()}${Math.floor(Math.random() * 1000)}`;
  const name = `E2E Admin Org ${slug.slice(-6)}`;
  const id = sql(`
    INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
    VALUES (gen_random_uuid(),'${name}','${slug}','UTILITY','LGA',false,1,0,0,0,'${admin.id}',now(),now())
    RETURNING id;
  `);
  sql(`
    INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
    VALUES (gen_random_uuid(),'${id}','${admin.id}','Gino Osahon','PLATFORM_ADMIN','OWNER',now());
  `);
  return { id, name, slug };
}

test.describe('organizations page', () => {
  test('table renders with kind + jurisdiction chips', async ({ authedAsAdmin: page, admin }) => {
    const org = await seedOrg(admin);
    await page.goto('/organizations');
    await expect(page.locator('.admin-page-title')).toContainText(/organization management/i);
    // Newly seeded org shows up.
    const row = page.locator('tr', { hasText: org.name });
    await expect(row).toBeVisible({ timeout: 5_000 });
    await expect(row).toContainText('UTILITY');
    await expect(row).toContainText('LGA');
  });

  test('verify toggle flips DB state', async ({ authedAsAdmin: page, admin }) => {
    const org = await seedOrg(admin);
    await page.goto('/organizations');
    const row = page.locator('tr', { hasText: org.name });
    await expect(row).toBeVisible({ timeout: 5_000 });

    // Auto-confirm the "Grant verified badge to ..." dialog.
    page.on('dialog', (d) => d.accept());
    await row.getByRole('button', { name: /^verify$/i }).click();

    // Row now shows the "Revoke verify" secondary button.
    await expect(row.getByRole('button', { name: /revoke verify/i })).toBeVisible({
      timeout: 5_000,
    });

    // DB state confirms.
    const verified = sql(`SELECT verified FROM organizations WHERE id='${org.id}';`);
    expect(verified).toBe('t');
  });
});
