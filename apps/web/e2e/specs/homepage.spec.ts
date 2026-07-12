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

    // Register CTA links to /register.
    const registerCta = page.getByRole('link', { name: /join civicos/i }).first();
    await expect(registerCta).toBeVisible();
    await expect(registerCta).toHaveAttribute('href', '/register');

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
