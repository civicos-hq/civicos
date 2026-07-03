import { test, expect } from '../fixtures/auth';

// Representatives page — public list, follow toggle, admin can create.
test.describe('representatives', () => {
  test('citizen sees the reps page with follow toggles', async ({
    authedAsCitizenInCommunity: page,
  }) => {
    await page.goto('/representatives');
    await expect(page.locator('.page-header-eyebrow').first()).toBeVisible();
    // If there are any reps rendered, they must have follow buttons.
    const followButtons = page.getByRole('button', { name: /follow$|following/i });
    // Not strict — if no reps in this community, the empty state renders.
    // Either the empty-state icon OR at least one follow button is present.
    const hasEmpty = await page.locator('.empty-state').count();
    const hasReps = await followButtons.count();
    expect(
      hasEmpty + hasReps,
      'either empty state or at least one representative should render',
    ).toBeGreaterThan(0);
  });

  test('admin sees the "+ New representative" button; citizen does not', async ({
    authedAsAdmin,
    authedAsCitizenInCommunity,
  }) => {
    await authedAsAdmin.goto('/representatives');
    await expect(authedAsAdmin.getByRole('button', { name: /new representative/i })).toBeVisible();

    await authedAsCitizenInCommunity.goto('/representatives');
    await expect(
      authedAsCitizenInCommunity.getByRole('button', { name: /new representative/i }),
    ).toHaveCount(0);
  });
});
