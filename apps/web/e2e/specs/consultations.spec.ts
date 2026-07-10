import type { Browser, Page } from '@playwright/test';
import { test, expect } from '../fixtures/auth';
import { sql } from '../fixtures/db';

// End-to-end lifecycle for the consultations feature — the highest-stakes
// flow in the org surface. This spec walks the full path:
//   seed org + owner → open org dashboard → create draft → try to publish
//   with no questions (NO_QUESTIONS) → add three question types →
//   publish → citizen submits response → admin closes → admin publishes
//   outcome → assert the responder gets a CONSULTATION_UPDATE notification
//   pointing at `#outcome`.
//
// Each step targets an invariant we want to catch breakage on:
//   - Publish rejected without at least one question
//   - Question builder writes to the DB
//   - Response submission requires PUBLISHED (state gate)
//   - Outcome publishing requires CLOSED
//   - Outcome fan-out reaches responders via the shared notifications table
test.describe('consultations', () => {
  test('org owner runs draft → publish → response → close → outcome flow', async ({
    browser,
    admin,
    citizen,
  }) => {
    // ── Seed a fresh org with the admin as OWNER ──────────────────
    // Fresh org per run keeps the spec independent from other specs and
    // from prior runs — no cross-test state, no timing races.
    const slug = `e2e-consult-${Date.now()}`;
    const orgId = sql(`
      INSERT INTO organizations (id,name,slug,kind,jurisdiction,verified,member_count,announcement_count,project_count,assignment_count,created_by_id,created_at,updated_at)
      VALUES (gen_random_uuid(),'E2E Consult Org','${slug}','NGO','STATE',false,1,0,0,0,'${admin.id}',now(),now())
      RETURNING id;
    `);
    sql(`
      INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES (gen_random_uuid(),'${orgId}','${admin.id}','${admin.name}','${admin.role}','OWNER',now());
    `);

    // Two authed page contexts — admin drives the org side, citizen
    // submits the response. Cannot reuse the `authedAsAdmin` fixture
    // because we need both authed pages open at once.
    const adminPage = await openAuthed(browser, admin.accessToken);
    // window.confirm auto-accept for the Close / Delete buttons.
    adminPage.on('dialog', (d) => d.accept());
    const citizenPage = await openAuthed(browser, citizen.accessToken);

    try {
      // ── Admin: open org dashboard, click New consultation ───────
      await adminPage.goto(`/org/${orgId}`);
      await adminPage.getByRole('button', { name: /new consultation/i }).click();
      await expect(adminPage).toHaveURL(new RegExp(`/org/${orgId}/consultations/new`));

      // ── Fill the draft create form. Scope every selector to the
      // form — the layout Topbar has a global SearchBar `<input>` that
      // would otherwise steal `.nth(0)`. ─────────────────────────────
      const title = `E2E consultation ${slug}`;
      const createForm = adminPage.locator('form');
      await createForm.locator('input').nth(0).fill(title);
      await createForm
        .locator('input')
        .nth(1)
        .fill('Playwright end-to-end summary — do not review.');
      await createForm
        .locator('textarea')
        .fill(
          'This consultation was created by the E2E suite. It exercises the full lifecycle and asserts each transition.',
        );
      await createForm.getByRole('button', { name: /save draft/i }).click();

      // Land on the org-side detail page — path carries the new id.
      await expect(adminPage).toHaveURL(new RegExp(`/org/${orgId}/consultations/[0-9a-f-]{36}$`), {
        timeout: 10_000,
      });
      const consultationId = adminPage.url().split('/').at(-1)!;

      // ── Publish before adding questions — must be rejected with
      // NO_QUESTIONS and the state must stay DRAFT. ────────────────
      await adminPage.getByRole('button', { name: /^publish$/i }).click();
      await expect(adminPage.getByText(/at least one question/i)).toBeVisible({
        timeout: 5_000,
      });
      expect(sql(`SELECT status FROM consultations WHERE id='${consultationId}';`)).toBe('DRAFT');

      // ── Add three question types covering both answer families:
      //   YES_NO + SINGLE_CHOICE → Selections[]
      //   SHORT_TEXT             → TextValue
      // ────────────────────────────────────────────────────────────
      await addQuestion(adminPage, {
        prompt: 'Do you support the proposed budget?',
        type: 'YES_NO',
      });
      await addQuestion(adminPage, {
        prompt: 'What is the top priority for your ward?',
        type: 'SHORT_TEXT',
        required: true,
      });
      await addQuestion(adminPage, {
        prompt: 'Which service needs the most work?',
        type: 'SINGLE_CHOICE',
        options: ['Water', 'Roads', 'Security'],
      });

      // Question rows landed in the DB.
      expect(
        Number(
          sql(
            `SELECT COUNT(*) FROM consultation_questions WHERE consultation_id='${consultationId}';`,
          ),
        ),
      ).toBe(3);

      // ── Publish ─────────────────────────────────────────────────
      await adminPage.getByRole('button', { name: /^publish$/i }).click();
      await expect(adminPage.getByRole('button', { name: /close consultation/i })).toBeVisible({
        timeout: 10_000,
      });
      expect(sql(`SELECT status FROM consultations WHERE id='${consultationId}';`)).toBe(
        'PUBLISHED',
      );

      // ── Citizen: fill and submit the response form ──────────────
      await citizenPage.goto(`/consultations/${consultationId}`);
      await expect(citizenPage.getByRole('heading', { name: title })).toBeVisible();

      // YES/NO — check Yes
      await citizenPage.getByRole('radio', { name: /^yes$/i }).check();
      // SHORT_TEXT — the required text input
      await citizenPage.locator('input[type="text"]').first().fill('Reliable electricity');
      // SINGLE_CHOICE — pick Roads
      await citizenPage.getByRole('radio', { name: /^roads$/i }).check();

      await citizenPage.getByRole('button', { name: /submit response/i }).click();
      // The mutation's onSuccess invalidates `my-consultation-responses`,
      // which flips `alreadyResponded` true and unmounts the form (and
      // the transient "Response recorded" toast) before it can be
      // observed. Instead wait for the read-only "already submitted"
      // banner — that's the state the page settles in after submit.
      await expect(citizenPage.getByText(/already submitted a response/i)).toBeVisible({
        timeout: 10_000,
      });
      expect(
        Number(
          sql(
            `SELECT COUNT(*) FROM consultation_responses WHERE consultation_id='${consultationId}';`,
          ),
        ),
      ).toBe(1);

      // ── Admin: close ────────────────────────────────────────────
      await adminPage.reload();
      await adminPage.getByRole('button', { name: /close consultation/i }).click();
      // The OutcomeForm's heading is our signal that CLOSED transitioned
      // and the outcome form rendered.
      await expect(adminPage.getByRole('heading', { name: /publish the outcome/i })).toBeVisible({
        timeout: 10_000,
      });
      expect(sql(`SELECT status FROM consultations WHERE id='${consultationId}';`)).toBe('CLOSED');

      // ── Admin: publish outcome ──────────────────────────────────
      const textareas = adminPage.locator('textarea');
      await textareas
        .nth(0)
        .fill('Majority of respondents flagged infrastructure as the top concern.');
      await textareas.nth(1).fill('We will prioritize road repairs in the next quarterly cycle.');
      await textareas
        .nth(2)
        .fill('Publish the tender schedule within 30 days and revisit here next quarter.');
      await adminPage.getByRole('button', { name: /^publish outcome$/i }).click();

      // The read-only outcome section renders once the POST returns —
      // its heading is "Outcome" (distinct from "Publish the outcome").
      await expect(adminPage.getByRole('heading', { name: /^outcome$/i })).toBeVisible({
        timeout: 10_000,
      });
      expect(
        Number(
          sql(
            `SELECT COUNT(*) FROM consultation_outcomes WHERE consultation_id='${consultationId}';`,
          ),
        ),
      ).toBe(1);

      // ── The responder was notified — CONSULTATION_UPDATE row with
      // link pointing at the outcome anchor. This is the "close the
      // loop" moment; if this regresses we ship silent outcomes. ──
      const notifCount = sql(`
        SELECT COUNT(*) FROM notifications
        WHERE user_id='${citizen.id}'
          AND type='CONSULTATION_UPDATE'
          AND link_url='/consultations/${consultationId}#outcome';
      `);
      expect(Number(notifCount)).toBeGreaterThanOrEqual(1);
    } finally {
      await adminPage.context().close();
      await citizenPage.context().close();
    }
  });
});

// Opens a new context with the given token pre-seeded in localStorage —
// same pattern as the auth fixture's `injectToken`, exposed here because
// this spec needs two authed pages at once (org owner + responder).
async function openAuthed(browser: Browser, token: string): Promise<Page> {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await page.addInitScript((t) => {
    localStorage.setItem('accessToken', t);
  }, token);
  return page;
}

// addQuestion opens the QuestionAdder, fills the form, saves, and waits
// for the form to collapse. Assumes the consultation is in DRAFT.
async function addQuestion(
  page: Page,
  q: {
    prompt: string;
    type: 'SHORT_TEXT' | 'LONG_TEXT' | 'SINGLE_CHOICE' | 'MULTI_CHOICE' | 'YES_NO';
    options?: string[];
    required?: boolean;
  },
) {
  await page.getByRole('button', { name: /^add question$/i }).click();
  // Only one form is open at a time in the builder — use `.last()` so
  // we grab it regardless of which existing question rows are on-page.
  const form = page.locator('form').last();
  await form.locator('input').first().fill(q.prompt);
  await form.locator('select').first().selectOption(q.type);
  if (q.options && q.options.length) {
    await form.locator('textarea').first().fill(q.options.join('\n'));
  }
  if (q.required) {
    await form.locator('input[type="checkbox"]').check();
  }
  await form.getByRole('button', { name: /^save$/i }).click();
  // Form collapses after save → "Add question" button reappears.
  await expect(page.getByRole('button', { name: /^add question$/i })).toBeVisible({
    timeout: 10_000,
  });
}
