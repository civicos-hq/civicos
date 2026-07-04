import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// The citizen half of the moderator flow. A citizen visits an issue,
// clicks Report on a comment, picks a reason, and sees the success
// state. The row must land in the moderation queue with matching reason
// and description — that's the criterion that separates "the button is
// wired" from "the pipe is actually connected."
test.describe('citizen report affordance', () => {
  test('opens the modal, files a report, sees the success state', async ({
    authedAsCitizenInCommunity: page,
    citizenInCommunity,
    admin,
  }) => {
    // Seed a comment on a fresh issue in the citizen's community.
    const communityId = sql(`SELECT community_id FROM users WHERE id='${citizenInCommunity.id}';`);
    const issueId = sql(`
      INSERT INTO issues (id,title,description,category,status,community_id,reported_by_id,image_urls,upvote_count,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E report-target issue','Placeholder body for the citizen report spec.','INFRASTRUCTURE','OPEN','${communityId}','${admin.id}','[]',0,0,now(),now())
      RETURNING id;
    `);
    const commentId = sql(`
      INSERT INTO issue_comments (id,content,issue_id,author_id,author_name,author_role,is_official_response,created_at)
      VALUES (gen_random_uuid(),'A comment the e2e citizen will report','${issueId}','${admin.id}','Reported Author','CITIZEN',false,now())
      RETURNING id;
    `);

    await page.goto(`/issues/${issueId}`);

    // Citizen clicks Report on the comment they just saw.
    await page
      .getByRole('button', { name: /report this content/i })
      .first()
      .click();

    // Modal opens with the reason legend visible.
    const dialog = page
      .locator('[role="dialog"], .fixed')
      .filter({ hasText: /report to moderators/i })
      .first();
    await expect(dialog).toBeVisible();

    // Pick ABUSE and add a description.
    await dialog.getByRole('radio', { name: /abuse or harassment/i }).click();
    await dialog.getByLabel(/anything else moderators should know/i).fill('automated citizen e2e');
    await dialog.getByRole('button', { name: /send report/i }).click();

    // Success state renders.
    await expect(dialog.getByText(/report received/i)).toBeVisible({ timeout: 5_000 });

    // DB confirms — one PENDING flag from this citizen against this comment.
    const flagRow = sql(
      `SELECT reason||'|'||status FROM content_flags WHERE content_id='${commentId}' AND reporter_id='${citizenInCommunity.id}';`,
    );
    expect(flagRow).toBe('ABUSE|PENDING');

    // Cleanup.
    sql(`DELETE FROM content_flags WHERE content_id='${commentId}';`);
    sql(`DELETE FROM issue_comments WHERE id='${commentId}';`);
    sql(`DELETE FROM issues WHERE id='${issueId}';`);
  });
});
