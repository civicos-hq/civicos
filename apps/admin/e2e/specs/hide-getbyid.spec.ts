import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Deep-link protection: hitting an announcement or progress-update URL
// directly when a moderator has flagged it HIDDEN must 404 like any
// other missing row. This is what closes the "I have the link, I can
// still read it" bypass.
test.describe('hide-enforcement — direct URL to hidden content returns 404', () => {
  test('hidden announcement GET /announcements/:id → 404', async ({ admin }) => {
    const slug = `admin-e2e-hide-getbyid-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Hide GetByID Org','${slug}','NGO','LGA',false,1,0,0,0,'${admin.id}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin.id}','${admin.name}','PLATFORM_ADMIN','OWNER',now());
    `);
    const annId = sql(`
      INSERT INTO announcements (id,organization_id,title,body,status,published_at,author_id,author_name,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Direct-link test','This announcement will be hidden before we fetch it by id.','PUBLISHED',now(),'${admin.id}','${admin.name}',now(),now())
      RETURNING id;
    `);

    // Sanity: without a HIDDEN flag, GET-by-id returns 200.
    const before = await fetch(`http://localhost:3000/api/v1/announcements/${annId}`);
    expect(before.status).toBe(200);

    // Flag as HIDDEN.
    sql(`
      INSERT INTO content_flags (id,content_type,content_id,reporter_id,reporter_name,reason,status,resolved_by_id,resolved_by_name,resolved_at,created_at,updated_at)
      VALUES (gen_random_uuid(),'ANNOUNCEMENT','${annId}','${admin.id}','${admin.name}','ABUSE','HIDDEN','${admin.id}','${admin.name}',now(),now(),now());
    `);

    // Direct URL now returns 404 with ANNOUNCEMENT_NOT_FOUND.
    const after = await fetch(`http://localhost:3000/api/v1/announcements/${annId}`);
    expect(after.status).toBe(404);
    const body = (await after.json()) as { code?: string };
    expect(body.code).toBe('ANNOUNCEMENT_NOT_FOUND');

    // Cleanup (CASCADE handles the org's members + announcement).
    sql(`DELETE FROM content_flags WHERE content_id='${annId}';`);
    sql(`DELETE FROM organizations WHERE id='${orgId}';`);
  });
});
