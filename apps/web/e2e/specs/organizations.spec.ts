import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Organizations end-to-end UI flow. Org rows are only minted by
// approving an OrganizationApplication (signup flow) — no in-app
// direct-create button for admins or citizens. Later tests seed
// directly via SQL so the org-owner UI still gets coverage.
test.describe('organizations', () => {
  test('no in-app "register organization" button — admins or citizens', async ({
    authedAsAdmin,
    authedAsCitizenInCommunity,
  }) => {
    await authedAsCitizenInCommunity.goto('/organizations');
    await expect(authedAsCitizenInCommunity.locator('.page-header-eyebrow').first()).toBeVisible();
    // Kind filter pills render.
    await expect(
      authedAsCitizenInCommunity.getByRole('button', { name: /^government$/i }),
    ).toBeVisible();
    await expect(
      authedAsCitizenInCommunity.getByRole('button', { name: /^utility$/i }),
    ).toBeVisible();
    await expect(
      authedAsCitizenInCommunity.getByRole('button', { name: /register organization/i }),
    ).toHaveCount(0);

    await authedAsAdmin.goto('/organizations');
    await expect(authedAsAdmin.getByRole('button', { name: /register organization/i })).toHaveCount(
      0,
    );
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
    // Empty states for all three (fresh org). Assignments is conditional on
    // `isMember` — which depends on the `/organizations/:id/assignments`
    // query resolving successfully. Use `toHaveCount` so Playwright waits
    // for the query to settle instead of snapshotting the DOM the moment
    // the ProjectList paint completes.
    await expect(page.locator('.empty-state')).toHaveCount(3, { timeout: 10_000 });
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
