import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Search covers seven buckets. This spec seeds one row per bucket with
// a shared unique needle in the searchable field, hits GET /api/v1/search
// through the citizen page, and asserts every bucket returns exactly the
// seeded row. Catches regression on any of the 7 shared-DB SELECTs.
test.describe('search — all seven buckets', () => {
  test('returns each seeded row in its correct bucket', async ({ admin, request }) => {
    // Needle is timestamp-based so parallel runs and prior data don't
    // collide with each other's seeded rows.
    const needle = `zzsrch${Date.now()}`;

    // Community for issue + petition + rep foreign keys. Reuse the shared
    // e2e community if it exists (created by other specs), else create.
    let communityId = sql(`SELECT id FROM communities WHERE slug='e2e-shared';`);
    if (!communityId) {
      communityId = sql(`
        INSERT INTO communities (id,name,slug,state,lga,country,created_by_id,created_at,updated_at)
        VALUES (gen_random_uuid(),'E2E Shared','e2e-shared','Lagos','Ikeja','Nigeria','${admin.id}',now(),now())
        RETURNING id;
      `);
    }

    // Org to anchor announcement/project/consultation.
    const orgSlug = `e2e-srch-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,description,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'Org ${needle}','${orgSlug}','NGO','STATE',false,1,0,0,0,'A search-test org','${admin.id}',now(),now())
      RETURNING id;
    `);

    // Seed one row per remaining bucket.
    sql(`
      INSERT INTO issues (id,title,description,category,status,community_id,reported_by_id,image_urls,upvote_count,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'Issue ${needle}','desc','OTHER','OPEN','${communityId}','${admin.id}','[]',0,0,now(),now());
    `);
    sql(`
      INSERT INTO petitions (id,title,description,status,goal,signature_count,community_id,created_by_id,image_urls,comment_count,created_at,updated_at)
      VALUES (gen_random_uuid(),'Petition ${needle}','desc','ACTIVE',100,0,'${communityId}','${admin.id}','[]',0,now(),now());
    `);
    sql(`
      INSERT INTO representatives (id,name,title,position,constituency,community_id,response_rate,follower_count,comment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'Rep ${needle}','Hon.','Councillor','Ward 1','${communityId}',0,0,0,'${admin.id}',now(),now());
    `);
    sql(`
      INSERT INTO announcements (id,organization_id,title,body,status,published_at,author_id,author_name,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Ann ${needle}','body','PUBLISHED',now(),'${admin.id}','${admin.name}',now(),now());
    `);
    sql(`
      INSERT INTO projects (id,organization_id,title,description,status,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Project ${needle}','desc','ACTIVE','${admin.id}',now(),now());
    `);
    sql(`
      INSERT INTO consultations (id,organization_id,title,summary,description,status,author_id,author_name,published_at,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Consult ${needle}','summary','desc','PUBLISHED','${admin.id}','${admin.name}',now(),now(),now());
    `);

    // Hit the gateway directly. Search is a public endpoint (no auth).
    const res = await request.get(`http://localhost:3000/api/v1/search?q=${needle}`);
    expect(res.ok(), `search request failed: ${res.status()}`).toBe(true);
    const body = (await res.json()) as {
      data: {
        issues: Array<{ title: string }>;
        petitions: Array<{ title: string }>;
        representatives: Array<{ name: string }>;
        organizations: Array<{ name: string }>;
        consultations: Array<{ title: string }>;
        announcements: Array<{ title: string }>;
        projects: Array<{ title: string }>;
      };
    };

    expect(body.data.issues.some((i) => i.title.includes(needle))).toBe(true);
    expect(body.data.petitions.some((p) => p.title.includes(needle))).toBe(true);
    expect(body.data.representatives.some((r) => r.name.includes(needle))).toBe(true);
    expect(body.data.organizations.some((o) => o.name.includes(needle))).toBe(true);
    expect(body.data.consultations.some((c) => c.title.includes(needle))).toBe(true);
    expect(body.data.announcements.some((a) => a.title.includes(needle))).toBe(true);
    expect(body.data.projects.some((p) => p.title.includes(needle))).toBe(true);
  });

  test('DRAFT consultation and non-PUBLISHED announcement are hidden from search', async ({
    admin,
    request,
  }) => {
    const needle = `zzhide${Date.now()}`;
    const orgSlug = `e2e-srchhide-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'Hide Org','${orgSlug}','NGO','STATE',false,1,0,0,0,'${admin.id}',now(),now())
      RETURNING id;
    `);
    // Draft consultation — must not appear.
    sql(`
      INSERT INTO consultations (id,organization_id,title,summary,description,status,author_id,author_name,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Consult ${needle}','summary','desc','DRAFT','${admin.id}','${admin.name}',now(),now());
    `);
    // Draft announcement — must not appear.
    sql(`
      INSERT INTO announcements (id,organization_id,title,body,status,author_id,author_name,created_at,updated_at)
      VALUES (gen_random_uuid(),'${orgId}','Ann ${needle}','body','DRAFT','${admin.id}','${admin.name}',now(),now());
    `);

    const res = await request.get(`http://localhost:3000/api/v1/search?q=${needle}`);
    const body = (await res.json()) as {
      data: {
        consultations: Array<{ title: string }>;
        announcements: Array<{ title: string }>;
      };
    };
    expect(body.data.consultations.some((c) => c.title.includes(needle))).toBe(false);
    expect(body.data.announcements.some((a) => a.title.includes(needle))).toBe(false);
  });
});
