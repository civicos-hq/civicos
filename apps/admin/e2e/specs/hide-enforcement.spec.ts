import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// The payoff test: a citizen posts a comment on an issue. A flag is
// filed. The admin hides it via the moderation queue UI. The citizen-
// facing GET /issues/:id/comments must drop that comment on next fetch.
// This is what turns "record the moderator's decision" (already tested)
// into "actually stop showing bad content."
test.describe('hide enforcement — moderator hide removes content from citizen surface', () => {
  test('hiding an issue comment via the queue drops it from the citizen list', async ({
    authedAsAdmin: page,
    citizen,
    admin,
  }) => {
    // Seed a real Nigerian community + issue, pin the citizen to it, post
    // a comment that we'll flag and hide. All directly through the DB so
    // the test doesn't burn rate-limit budget on setup.
    const communityId =
      sql(`SELECT id FROM communities WHERE slug='e2e-shared';`) ||
      sql(`
        INSERT INTO communities (id,name,slug,state,lga,country,created_by_id,created_at,updated_at)
        VALUES (gen_random_uuid(),'E2E Shared','e2e-shared','Lagos','Ikeja','Nigeria','${admin.id}',now(),now())
        RETURNING id;
      `);
    sql(`UPDATE users SET community_id='${communityId}' WHERE id='${citizen.id}';`);

    const issueId = sql(`
      INSERT INTO issues (id,title,description,category,status,community_id,reported_by_id,image_urls,upvote_count,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E hide-target issue','Placeholder description for the hide-enforcement spec.','INFRASTRUCTURE','OPEN','${communityId}','${citizen.id}','[]',0,0,now(),now())
      RETURNING id;
    `);

    const commentId = sql(`
      INSERT INTO issue_comments (id,content,issue_id,author_id,author_name,author_role,is_official_response,created_at)
      VALUES (gen_random_uuid(),'This comment will be hidden by the moderator via the admin UI.','${issueId}','${citizen.id}','${citizen.name}','CITIZEN',false,now())
      RETURNING id;
    `);

    // Sanity: the citizen-facing list currently returns this comment.
    const before = await fetch(`http://localhost:3000/api/v1/issues/${issueId}/comments`).then(
      (r) => r.json() as Promise<{ data: { comments: unknown[] } }>,
    );
    expect(before.data.comments.length).toBe(1);

    // Seed the PENDING flag directly (the flag intake API is Strict-rate-
    // limited and this suite runs alongside the other spec files that
    // also file flags, so bypass the endpoint).
    const flagId = sql(`
      INSERT INTO content_flags (id,content_type,content_id,reporter_id,reporter_name,reason,description,status,created_at,updated_at)
      VALUES (gen_random_uuid(),'ISSUE_COMMENT','${commentId}','${citizen.id}','${citizen.name}','ABUSE','automated hide-enforcement spec','PENDING',now(),now())
      RETURNING id;
    `);

    // Admin navigates to the moderation queue and clicks Hide on the row
    // that shows the ABUSE reason for our specific flag.
    await page.goto('/flags');
    const row = page.locator('tr', { hasText: 'ABUSE' }).filter({
      hasText: 'automated hide-enforcement spec',
    });
    await expect(row).toBeVisible({ timeout: 5_000 });

    // Same two-step workflow as flags.spec.ts: Review opens an inline
    // resolution panel that carries the Hide button and the note textarea.
    await row.getByRole('button', { name: /^review$/i }).click();
    const panel = page.locator('tr').filter({ hasText: /Resolve flag for/i });
    await expect(panel).toBeVisible();
    await panel.locator('textarea').fill('hidden by e2e');
    page.on('dialog', (d) => {
      if (d.type() === 'prompt') d.accept('hidden by e2e');
      else d.accept();
    });
    await panel.getByRole('button', { name: /^hide$/i }).click();

    // Flag flips to HIDDEN in the DB.
    await expect
      .poll(() => sql(`SELECT status FROM content_flags WHERE id='${flagId}';`), {
        timeout: 5_000,
      })
      .toBe('HIDDEN');

    // Citizen-facing list still returns the comment as a placeholder row
    // (isHidden=true, content + authorName replaced). The original
    // content and author name must NOT appear anywhere in the response
    // — that's the actual privacy/moderation guarantee.
    const after = await fetch(`http://localhost:3000/api/v1/issues/${issueId}/comments`).then(
      (r) =>
        r.json() as Promise<{
          data: {
            comments: Array<{
              id: string;
              content: string;
              authorName: string;
              isHidden?: boolean;
            }>;
          };
        }>,
    );
    expect(after.data.comments.length).toBe(1);
    const hiddenRow = after.data.comments[0];
    expect(hiddenRow.isHidden).toBe(true);
    expect(hiddenRow.content).toBe('[Removed by moderator]');
    expect(hiddenRow.authorName).toBe('[Removed]');
    expect(hiddenRow.content).not.toContain('This comment will be hidden');

    // Cleanup — reverse dependency order.
    sql(`DELETE FROM audit_logs WHERE target_id='${flagId}';`);
    sql(`DELETE FROM content_flags WHERE id='${flagId}';`);
    sql(`DELETE FROM issue_comments WHERE id='${commentId}';`);
    sql(`DELETE FROM issues WHERE id='${issueId}';`);
  });
});
