import { test, expect } from '../fixtures/auth';

// Public marketing homepage — the un-authed landing surface. Should
// render every section, the Docket should be present with 4+ rows, and
// the top-nav anchor links should scroll (or navigate) into place.
test.describe('homepage', () => {
  test('renders every section and the Docket has live rows', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/');

    // Masthead + hero.
    // Note: `.hero-art` used to be a decorative SVG on the hero. It was
    // removed when the homepage adopted the spotlight-orb hero shell;
    // the surrounding structure (`.home-hero` container + the signature
    // stroke) is what we now assert on.
    await expect(page.locator('.home-hero-title')).toBeVisible();
    await expect(page.locator('.hero-signature')).toBeVisible();
    await expect(page.locator('.home-hero')).toBeVisible();

    // Docket — at least 4 initial seed rows, plus a live indicator.
    // The Docket component paints entries as `.docket-record` articles
    // (the earlier `.docket-row` class was superseded when the widget
    // was redesigned with the browser-chrome frame).
    const docketRows = page.locator('.docket-record');
    await expect(docketRows.first()).toBeVisible();
    expect(await docketRows.count()).toBeGreaterThanOrEqual(4);
    await expect(page.locator('.docket-live')).toBeVisible();

    // Every section marker present.
    const markers = [
      'parties.marker',
      'articles.marker',
      'principles.marker',
      'steps.marker',
      'faq.marker',
      'newsletter.marker',
    ];
    for (const key of markers) {
      // TypedMarker uses aria-label on the paragraph — much easier to grab.
      await expect(
        page.locator(`.home-section-marker[aria-label]`).nth(markers.indexOf(key)),
      ).toBeVisible();
    }

    // Register CTA links to /register. Copy changed from "Join CivicOS —
    // it's free" to "Join your community" when the landing page was
    // rewritten to lead with local government rather than democracy.
    const registerCta = page.getByRole('link', { name: /join your community/i }).first();
    await expect(registerCta).toBeVisible();
    await expect(registerCta).toHaveAttribute('href', '/register');

    // Docket rows name a real ward/LGA rather than a reference code.
    const places = page.locator('.docket-record-place');
    expect(await places.count()).toBeGreaterThanOrEqual(4);
    await expect(places.first()).toBeVisible();
    await expect(places.first()).toContainText(/LGA|Ward|Council|Municipal/i);

    await page.close();
  });

  // The onboarding section is a tablist, not a static card grid — selecting a
  // step swaps the detail panel. Keyboard parity matters here because the rail
  // is a single tab stop with roving tabindex.
  test('onboarding step-through responds to click and keyboard', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/');

    const tabs = page.locator('.home-stepper-tab');
    const panelTitle = page.locator('.home-stepper-panel-title');
    await expect(tabs).toHaveCount(4);

    // Step 01 selected on load, and it owns the only tab stop.
    await expect(tabs.nth(0)).toHaveAttribute('aria-selected', 'true');
    const firstTitle = await panelTitle.textContent();

    // Clicking a later step swaps the panel and moves selection.
    await tabs.nth(2).click();
    await expect(tabs.nth(2)).toHaveAttribute('aria-selected', 'true');
    await expect(tabs.nth(0)).toHaveAttribute('aria-selected', 'false');
    await expect(panelTitle).not.toHaveText(firstTitle ?? '');

    // The panel is wired to the selected tab both ways.
    const controls = await tabs.nth(2).getAttribute('aria-controls');
    await expect(page.locator('.home-stepper-panel')).toHaveAttribute('id', controls ?? '');

    // Arrow keys move selection; advancing past the last step wraps.
    await page.keyboard.press('ArrowDown');
    await expect(tabs.nth(3)).toHaveAttribute('aria-selected', 'true');
    await page.keyboard.press('ArrowDown');
    await expect(tabs.nth(0)).toHaveAttribute('aria-selected', 'true');
    await page.keyboard.press('End');
    await expect(tabs.nth(3)).toHaveAttribute('aria-selected', 'true');

    await page.close();
  });

  // Officials shouldn't land on the citizen form from the closing CTA.
  test('partner CTA deep-links to the representative signup', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/');

    await page.locator('.home-cta-buttons a[href="/register?type=REPRESENTATIVE"]').click();
    await expect(page).toHaveURL(/\/register\?type=REPRESENTATIVE$/);
    await expect(page.locator('input[name="accountType"][value="REPRESENTATIVE"]')).toBeChecked();

    await page.close();
  });

  test('top-nav anchor links jump to their sections', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/');

    // Click the "How it works" nav link — pathname stays / but hash flips.
    await page.getByRole('link', { name: /how it works/i }).click();
    await expect(page).toHaveURL(/\/#how$/);
    // The scroll effect targets #how.
    await expect(page.locator('#how')).toBeInViewport({ ratio: 0.05 });

    await page.close();
  });

  test('privacy link in footer navigates to /privacy', async ({ browser }) => {
    const page = await browser.newPage();
    await page.goto('/');
    await page.locator('.home-footer-col a[href="/privacy"]').click();
    await expect(page).toHaveURL(/\/privacy$/);
    await page.close();
  });
});
