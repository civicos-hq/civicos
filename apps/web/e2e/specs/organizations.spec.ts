import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Organizations end-to-end UI flow — this is the money-path spec for the
// service we just built. Admin registers an org, publishes an
// announcement, creates a project, and the flow renders each step.
test.describe('organizations', () => {
  test('citizen browses org list; register button is admin-gated', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/organizations');
    await expect(page.locator('.page-header-eyebrow').first()).toBeVisible();
    // Kind filter pills render.
    await expect(page.getByRole('button', { name: /^government$/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /^utility$/i })).toBeVisible();
    // Citizen doesn't see the create button.
    await expect(page.getByRole('button', { name: /register organization/i })).toHaveCount(0);
  });

  test('admin registers a new organization end to end', async ({ authedAsAdmin: page }) => {
    await page.goto('/organizations');
    // Open the New Organization modal.
    await page.getByRole('button', { name: /register organization/i }).click();
    const dialog = page
      .locator('[role="dialog"], .fixed')
      .filter({ hasText: /register|new organization/i })
      .first();
    await expect(dialog).toBeVisible();

    // Fill the form. First two inputs are Name then Slug (see NewOrganizationModal).
    const slug = `e2e-org-${Date.now()}`;
    const inputs = dialog.locator('input:not([type="file"])');
    await inputs.nth(0).fill(`E2E Org ${slug}`);
    await inputs.nth(1).fill(slug);

    // The two <select>s are Kind and Jurisdiction.
    await dialog.locator('select').nth(0).selectOption('UTILITY');
    await dialog.locator('select').nth(1).selectOption('LGA');

    // Optional description.
    await dialog.locator('textarea').fill('End-to-end test organization.');

    // Submit.
    await dialog.getByRole('button', { name: /^register$/i }).click();

    // Dialog closes on success.
    await expect(dialog).toBeHidden({ timeout: 5_000 });

    // Newly created org appears in the list.
    await expect(page.getByText(`E2E Org ${slug}`)).toBeVisible();

    // DB verification — the org and the auto-owner membership both landed.
    const memberCount = sql(
      `SELECT COUNT(*) FROM org_members WHERE organization_id=(SELECT id FROM organizations WHERE slug='${slug}');`,
    );
    expect(Number(memberCount)).toBe(1);
  });

  test('admin visits the org detail page and sees empty tabs', async ({ authedAsAdmin: page }) => {
    // Pre-create an org so we don't depend on the previous test's output.
    const admin = sql(`SELECT id FROM users WHERE email='gino.osahon@gmail.com';`);
    const slug = `e2e-detail-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Detail','${slug}','GOVERNMENT','LGA',false,1,0,0,0,'${admin}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin}','Gino Osahon','PLATFORM_ADMIN','OWNER',now());
    `);

    await page.goto(`/organizations/${orgId}`);
    // Header shows the org name.
    await expect(page.getByRole('heading', { name: 'E2E Detail' })).toBeVisible();
    // Three section headings — Announcements, Projects, Reports received.
    await expect(page.getByRole('heading', { name: /announcements/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /projects/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /reports received/i })).toBeVisible();
    // Empty states for all three (fresh org).
    const emptyStates = page.locator('.empty-state');
    expect(await emptyStates.count()).toBeGreaterThanOrEqual(3);
  });

  test('admin publishes an announcement from the org detail page', async ({
    authedAsAdmin: page,
  }) => {
    const admin = sql(`SELECT id FROM users WHERE email='gino.osahon@gmail.com';`);
    const slug = `e2e-ann-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Ann Org','${slug}','NGO','STATE',false,1,0,0,0,'${admin}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin}','Gino Osahon','PLATFORM_ADMIN','OWNER',now());
    `);

    await page.goto(`/organizations/${orgId}`);
    await page.getByRole('button', { name: /new announcement/i }).click();
    const dialog = page
      .locator('[role="dialog"], .fixed')
      .filter({ hasText: /new announcement/i })
      .first();
    await expect(dialog).toBeVisible();

    await dialog.locator('input').first().fill('End-to-end announcement');
    await dialog
      .locator('textarea')
      .fill('This announcement was published by the Playwright suite.');
    // "Publish immediately" checkbox is on by default.
    await dialog.getByRole('button', { name: /save/i }).click();

    await expect(dialog).toBeHidden({ timeout: 5_000 });

    // Announcement is visible in the org's section.
    await expect(page.getByText('End-to-end announcement')).toBeVisible();
    // Draft-mode empty state is gone.
    const emptyAnnouncements = page.locator('.empty-state').filter({ hasText: /announcement/i });
    expect(await emptyAnnouncements.count()).toBe(0);
  });
});
