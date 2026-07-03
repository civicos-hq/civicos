import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// The end-to-end payoff: a progress update posted by an organization
// against an assigned issue MUST render inside the Official Progress
// section on the citizen-facing issue detail page. This exercises the
// useIssueProgressUpdates hook + OfficialProgressSection component +
// useQueries batch org-name fetch we wired last week.
test.describe('cross-service — org progress on issue detail', () => {
  test('citizen viewing their issue sees the Official Progress section', async ({
    authedAsCitizenInCommunity: page,
    citizenInCommunity,
    admin,
  }) => {
    const communityId = sql(`SELECT community_id FROM users WHERE id='${citizenInCommunity.id}';`);
    const issueId = sql(`
      INSERT INTO issues (id,title,description,category,status,community_id,reported_by_id,image_urls,upvote_count,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E issue for progress test','A short description of the community issue used in a cross-service Playwright spec.','INFRASTRUCTURE','OPEN','${communityId}','${citizenInCommunity.id}','[]',0,0,now(),now())
      RETURNING id;
    `);

    const orgSlug = `e2e-progress-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Response Board','${orgSlug}','GOVERNMENT','LGA',true,1,0,0,1,'${admin.id}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin.id}','${admin.name}','PLATFORM_ADMIN','OWNER',now());
    `);
    sql(`
      INSERT INTO issue_assignments (id,organization_id,issue_id,status,assigned_by_id,assigned_by_name,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','${issueId}','IN_PROGRESS','${admin.id}','${admin.name}',now(),now());
    `);
    sql(`
      INSERT INTO progress_updates (id,organization_id,issue_id,body,is_public,author_id,author_name,created_at)
      VALUES (gen_random_uuid(),'${orgId}','${issueId}','Team dispatched. Repair scheduled for next week — automated Playwright test.',true,'${admin.id}','${admin.name}',now());
    `);

    // Citizen opens their issue detail page.
    await page.goto(`/issues/${issueId}`);

    // Official Progress section renders, headed by the Megaphone icon.
    await expect(page.getByRole('heading', { name: /official progress/i })).toBeVisible();

    // The org name is a link to /organizations/:orgId.
    const orgLink = page.getByRole('link', { name: 'E2E Response Board' });
    await expect(orgLink).toBeVisible();
    await expect(orgLink).toHaveAttribute('href', `/organizations/${orgId}`);

    // The body text of the update renders.
    await expect(page.getByText(/team dispatched.*repair scheduled/i)).toBeVisible();

    // Author attribution renders.
    await expect(page.getByText(new RegExp(`posted by ${admin.name}`, 'i'))).toBeVisible();
  });

  test('assignments endpoint is member-only — citizen 403 gate holds', async ({
    admin,
    citizenInCommunity,
  }) => {
    const orgSlug = `e2e-gate-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Gate Test','${orgSlug}','NGO','LGA',false,1,0,0,0,'${admin.id}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin.id}','${admin.name}','PLATFORM_ADMIN','OWNER',now());
    `);

    // Citizen tries to read the assignments list — should 403.
    const res = await fetch(`http://localhost:3000/api/v1/organizations/${orgId}/assignments`, {
      headers: { Authorization: `Bearer ${citizenInCommunity.accessToken}` },
    });
    expect(res.status).toBe(403);
  });
});
