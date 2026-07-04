import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Complements admin/e2e/specs/hide-enforcement.spec.ts. That spec proves
// the hide DECISION removes the comment from the count. This spec
// proves the citizen surface shows the "[Removed by moderator]"
// placeholder instead of silently dropping the row — the UX side of
// the same feature.
test.describe('hide placeholder — citizen surface preserves conversation flow', () => {
  test('hidden comment renders as [Removed] with muted styling', async ({
    authedAsCitizenInCommunity: page,
    citizenInCommunity,
    admin,
  }) => {
    const communityId = sql(`SELECT community_id FROM users WHERE id='${citizenInCommunity.id}';`);
    const issueId = sql(`
      INSERT INTO issues (id,title,description,category,status,community_id,reported_by_id,image_urls,upvote_count,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E placeholder-target issue','Description for the hide-placeholder spec.','INFRASTRUCTURE','OPEN','${communityId}','${admin.id}','[]',0,0,now(),now())
      RETURNING id;
    `);
    // One normal comment (should render normally) + one that will be hidden.
    const okId = sql(`
      INSERT INTO issue_comments (id,content,issue_id,author_id,author_name,author_role,is_official_response,created_at)
      VALUES (gen_random_uuid(),'This comment should render normally.','${issueId}','${citizenInCommunity.id}','${citizenInCommunity.name}','CITIZEN',false,now())
      RETURNING id;
    `);
    const hiddenId = sql(`
      INSERT INTO issue_comments (id,content,issue_id,author_id,author_name,author_role,is_official_response,created_at)
      VALUES (gen_random_uuid(),'THIS SHOULD BE HIDDEN — never rendered to the citizen.','${issueId}','${admin.id}','Author Whose Name Should Also Vanish','CITIZEN',false,now())
      RETURNING id;
    `);
    // Seed a HIDDEN flag against the second comment directly.
    sql(`
      INSERT INTO content_flags (id,content_type,content_id,reporter_id,reporter_name,reason,status,resolved_by_id,resolved_by_name,resolved_at,created_at,updated_at)
      VALUES (gen_random_uuid(),'ISSUE_COMMENT','${hiddenId}','${admin.id}','${admin.name}','ABUSE','HIDDEN','${admin.id}','${admin.name}',now(),now(),now());
    `);

    await page.goto(`/issues/${issueId}`);

    // Normal comment renders unchanged.
    await expect(page.getByText('This comment should render normally.')).toBeVisible();

    // Hidden comment renders as the placeholder — actual content and
    // original author name must NOT be present anywhere on the page.
    await expect(page.getByText(/this comment was removed by a moderator/i)).toBeVisible();
    await expect(page.getByText(/THIS SHOULD BE HIDDEN/)).toHaveCount(0);
    await expect(page.getByText(/Author Whose Name Should Also Vanish/)).toHaveCount(0);

    // The placeholder row has the muted-slate treatment (dashed border).
    const placeholder = page.locator('[data-hidden="true"]');
    await expect(placeholder).toBeVisible();

    // Cleanup.
    sql(`DELETE FROM content_flags WHERE content_id='${hiddenId}';`);
    sql(`DELETE FROM issue_comments WHERE issue_id='${issueId}';`);
    sql(`DELETE FROM issues WHERE id='${issueId}';`);
    // ok reference — silence lint if any
    void okId;
  });
});
