// Generates the Open Graph share image at apps/web/public/og-image.png.
// Renders an HTML template with Playwright's headless Chromium so the
// output matches the site's paper-record aesthetic pixel-for-pixel
// (same Fraunces + JetBrains Mono fonts, same warm-ink palette).
//
// Regenerate any time the branding shifts:
//   node apps/web/scripts/generate-og-image.mjs
//
// The committed output at /public/og-image.png is what social crawlers
// (WhatsApp, LinkedIn, X, Facebook, Slack) fetch. 1200×630 is the
// spec that satisfies every major platform without cropping.

import { chromium } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { readFileSync } from 'node:fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT = resolve(__dirname, '../public/og-image.png');
const MARK = resolve(__dirname, '../public/civicos-mark.png');
// Inline the brand mark as base64 so the render is self-contained and
// doesn't race a file:// fetch during headless load.
const markDataUri = `data:image/png;base64,${readFileSync(MARK).toString('base64')}`;

const html = `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,700&family=JetBrains+Mono:wght@500&display=swap" rel="stylesheet" />
<style>
  :root {
    --ink: #1a1a1a;
    --ink-soft: #4a4a4a;
    --paper: #f7f4ef;
    --rule: rgba(0, 0, 0, 0.08);
    --stamp: #3b82f6;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  html, body { width: 1200px; height: 630px; overflow: hidden; }
  body {
    font-family: 'Fraunces', Georgia, serif;
    background: var(--paper);
    color: var(--ink);
    position: relative;
    display: flex;
    flex-direction: column;
    padding: 72px;
    /* Repeating hairline ruling — echoes the site's document-ledger feel. */
    background-image:
      repeating-linear-gradient(0deg, transparent 0 47px, var(--rule) 47px 48px);
  }
  .marker {
    font-family: 'JetBrains Mono', monospace;
    font-size: 15px;
    letter-spacing: 0.24em;
    text-transform: uppercase;
    color: var(--ink-soft);
    margin-bottom: 24px;
  }
  .stamp { color: var(--stamp); font-weight: 500; }
  h1 {
    font-size: 88px;
    line-height: 1.02;
    letter-spacing: -0.02em;
    font-weight: 700;
    max-width: 900px;
    margin-bottom: 28px;
  }
  h1 em {
    font-style: italic;
    color: var(--stamp);
    font-weight: 600;
  }
  .sub {
    font-family: 'Fraunces', Georgia, serif;
    font-size: 26px;
    line-height: 1.4;
    color: var(--ink-soft);
    max-width: 820px;
    margin-bottom: auto;
  }
  .foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-family: 'JetBrains Mono', monospace;
    font-size: 18px;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--ink-soft);
    padding-top: 24px;
    border-top: 1px solid var(--rule);
  }
  .brand {
    display: inline-flex;
    align-items: center;
    gap: 14px;
  }
  .brand img { width: 44px; height: 44px; border-radius: 10px; }
  .brand-name {
    font-family: 'Fraunces', Georgia, serif;
    font-weight: 700;
    font-size: 26px;
    letter-spacing: -0.01em;
    color: var(--ink);
    text-transform: none;
  }
</style>
</head>
<body>
  <div class="marker"><span class="stamp">§ 00</span> — CIVICOS.NG</div>
  <h1>An operating system for <em>civic participation</em>.</h1>
  <p class="sub">
    Open infrastructure that lets citizens, governments, universities,
    NGOs, and communities build trusted civic experiences — organized
    around the places people actually live.
  </p>
  <div class="foot">
    <div class="brand">
      <img src="${markDataUri}" alt="" />
      <span class="brand-name">CivicOS</span>
    </div>
    <span>OPEN CIVIC INFRASTRUCTURE</span>
  </div>
</body>
</html>`;

async function main() {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1200, height: 630 },
    deviceScaleFactor: 2, // retina-quality output; social crawlers render at 2x on modern devices
  });
  const page = await context.newPage();
  await page.setContent(html, { waitUntil: 'networkidle' });
  // Give the Google Fonts one extra beat to fully rasterize — waitUntil
  // networkidle catches the CSS request but the font-face swap can lag
  // a frame behind the last request.
  await page.waitForTimeout(500);
  await page.screenshot({ path: OUT, type: 'png', omitBackground: false });
  await browser.close();
  console.log(`wrote ${OUT}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
