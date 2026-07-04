import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// The UserDetailPage proves two things at once:
//   1. Clicking the email in the Users list navigates to the detail
//      route with a working data fetch.
//   2. The audit trail + reports-filed sections both render live data
//      from the two backend queries (identity-service).
test.describe('user detail page', () => {
  test('clicking a user email opens the detail page with three data cards', async ({
    authedAsAdmin: page,
    citizen,
    admin,
  }) => {
    // Seed one audit row so the "actions taken" section has content.
    sql(`
      INSERT INTO audit_logs (id,actor_id,actor_name,actor_role,action,target_type,target_id,metadata,created_at)
      VALUES (gen_random_uuid(),'${admin.id}','${admin.name}','PLATFORM_ADMIN','user.role_changed','USER','${citizen.id}','{"previousRole":"CITIZEN","newRole":"MODERATOR"}',now())
    `);
    // Seed one flag they filed so the "reports filed" section has content.
    sql(`
      INSERT INTO content_flags (id,content_type,content_id,reporter_id,reporter_name,reason,description,status,created_at,updated_at)
      VALUES (gen_random_uuid(),'ISSUE_COMMENT',gen_random_uuid(),'${citizen.id}','${citizen.name}','SPAM','automated e2e','PENDING',now(),now())
    `);

    // Filter Users list to just this citizen, then click their email.
    await page.goto('/users');
    await page.locator('.admin-table-search[placeholder*="Search email"]').fill(citizen.email);
    const row = page.locator('tr', { hasText: citizen.email });
    await expect(row).toBeVisible({ timeout: 5_000 });
    await row.getByRole('link', { name: citizen.email }).click();

    // Detail page renders with the correct URL and title.
    await expect(page).toHaveURL(new RegExp(`/users/${citizen.id}$`));
    await expect(page.locator('.admin-page-title')).toContainText(citizen.name);
    await expect(page.locator('.admin-page-sub')).toContainText(citizen.email);

    // Four identity cards (Role, Verified, Status, Joined).
    await expect(page.locator('.admin-stat-card')).toHaveCount(4);
    await expect(page.getByText(/^Role$/i)).toBeVisible();
    await expect(page.getByText(/Account status/i)).toBeVisible();

    // Audit trail section shows the seeded user.role_changed row.
    await expect(page.getByRole('heading', { name: /actions taken/i })).toBeVisible();
    await expect(page.getByText('user.role_changed')).toBeVisible();

    // Reports-filed section shows the SPAM flag.
    await expect(page.getByRole('heading', { name: /reports filed/i })).toBeVisible();
    await expect(page.getByText(/^SPAM$/)).toBeVisible();

    // Back link routes to /users.
    await page.getByRole('link', { name: /back to users/i }).click();
    await expect(page).toHaveURL(/\/users$/);

    // Cleanup — user, audit rows, flags are all cascaded by the global teardown
    // via the smoke email prefix, so nothing to do here.
  });

  test('banned user cannot write — POST to community returns 403 ACCOUNT_BANNED', async ({
    citizen,
  }) => {
    // Ban the citizen directly, then attempt a write with their still-valid JWT.
    sql(`UPDATE users SET banned_at=now() WHERE id='${citizen.id}';`);

    const res = await fetch('http://localhost:3000/api/v1/issues', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${citizen.accessToken}`,
      },
      body: JSON.stringify({
        title: 'This should never land',
        description: 'A banned user is trying to post — the middleware should refuse.',
        category: 'INFRASTRUCTURE',
        communityId: '00000000-0000-0000-0000-000000000000',
      }),
    });
    expect(res.status).toBe(403);
    const body = (await res.json()) as { code?: string };
    expect(body.code).toBe('ACCOUNT_BANNED');

    // Un-ban so global teardown can delete the user without RESTRICT issues.
    sql(`UPDATE users SET banned_at=NULL WHERE id='${citizen.id}';`);
  });

  test('banned user can still read — GET on public endpoint returns 200', async ({ citizen }) => {
    sql(`UPDATE users SET banned_at=now() WHERE id='${citizen.id}';`);

    // /me is authenticated but a safe method, so ban check skips it.
    const res = await fetch('http://localhost:3000/api/v1/auth/me', {
      headers: { Authorization: `Bearer ${citizen.accessToken}` },
    });
    expect(res.status).toBe(200);

    sql(`UPDATE users SET banned_at=NULL WHERE id='${citizen.id}';`);
  });
});
