import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// Seed a citizen who files a pending flag against a fabricated UUID.
// The moderation queue should list it, and hiding it should record an
// audit-log row with the resolver's identity and note.
async function seedPendingFlag(reporterId: string): Promise<{ id: string; targetId: string }> {
  // The content_id is a bare UUID reference — no FK to specific content
  // tables, so we can use any UUID.
  const targetId = sql(`SELECT gen_random_uuid();`);
  const id = sql(`
    INSERT INTO content_flags (id,content_type,content_id,reporter_id,reporter_name,reason,description,status,created_at,updated_at)
    VALUES (gen_random_uuid(),'ISSUE_COMMENT','${targetId}','${reporterId}','E2E Reporter','SPAM','automated e2e','PENDING',now(),now())
    RETURNING id;
  `);
  return { id, targetId };
}

test.describe('moderation queue', () => {
  test('pending flag appears in the queue', async ({ authedAsAdmin: page, citizen }) => {
    const flag = await seedPendingFlag(citizen.id);
    await page.goto('/flags');
    await expect(page.locator('.admin-page-title')).toContainText(/moderation queue/i);

    // Row identified by the reason + content type chip.
    const row = page.locator('tr', { hasText: 'SPAM' }).filter({ hasText: 'ISSUE_COMMENT' });
    await expect(row).toBeVisible({ timeout: 5_000 });
    await expect(row.locator('.admin-chip-status-PENDING')).toBeVisible();
    // Reporter name renders.
    await expect(row).toContainText(/E2E Reporter/);
    // Cleanup after use — global teardown catches it too but keep DB tidy.
    sql(`DELETE FROM content_flags WHERE id='${flag.id}';`);
  });

  test('hide action resolves the flag and writes an audit entry', async ({
    authedAsAdmin: page,
    citizen,
  }) => {
    const flag = await seedPendingFlag(citizen.id);
    await page.goto('/flags');

    const row = page.locator('tr', { hasText: 'SPAM' }).first();
    await expect(row).toBeVisible({ timeout: 5_000 });

    // The flags UI is two-step: PENDING rows expose a "Review" button which
    // expands an inline resolution panel with the note textarea + Hide /
    // Dismiss buttons in a second <tr>. Click Review, fill the note in the
    // panel (no more browser prompt), then click Hide.
    await row.getByRole('button', { name: /^review$/i }).click();
    const panel = page.locator('tr').filter({ hasText: /Resolve flag for/i });
    await expect(panel).toBeVisible();
    await panel.locator('textarea').fill('e2e — hidden by test');
    // Any remaining prompt/confirm listener is a no-op safety net for older
    // UI states that might still surface a dialog.
    page.on('dialog', (d) => {
      if (d.type() === 'prompt') d.accept('e2e — hidden by test');
      else d.accept();
    });
    await panel.getByRole('button', { name: /^hide$/i }).click();

    // Flag DB state reflects the resolution.
    await expect
      .poll(() => sql(`SELECT status FROM content_flags WHERE id='${flag.id}';`), {
        timeout: 5_000,
      })
      .toBe('HIDDEN');

    // Audit trail has a flag.resolved entry from the admin.
    const auditCount = sql(
      `SELECT COUNT(*) FROM audit_logs WHERE target_id='${flag.id}' AND action='flag.resolved';`,
    );
    expect(Number(auditCount)).toBe(1);

    // Cleanup.
    sql(`DELETE FROM audit_logs WHERE target_id='${flag.id}';`);
    sql(`DELETE FROM content_flags WHERE id='${flag.id}';`);
  });
});
