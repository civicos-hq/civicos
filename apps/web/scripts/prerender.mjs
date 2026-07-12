// Marketing-surface prerender pass. Runs after `vite build` and, for
// each public marketing route, spawns a headless Chromium via
// Playwright, waits for network idle + a beat for useEffect-driven
// meta updates (title + description via `useSeo`), then serialises the
// full DOM into a per-route `index.html` under `dist/`.
//
// Why: `/`, `/privacy`, and `/terms` are the only pages a crawler
// (Google, WhatsApp, LinkedIn, etc.) or a fresh visitor lands on
// without an auth round-trip. Serving prerendered HTML gives them
// full content on frame 1 — measurable LCP + SEO win over the vanilla
// SPA shell that just says `<div id="root"></div>`.
//
// Dashboard routes stay client-rendered — they need auth + per-user
// data and aren't citizen-facing SEO targets. Robots.txt already
// Disallows them.
//
// Regenerate manually: `node apps/web/scripts/prerender.mjs`.
// Automatic: wired into the `build` npm script so a plain `pnpm build`
// produces prerendered HTML with zero extra steps.

import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const APP_DIR = resolve(__dirname, '..');
const DIST = resolve(APP_DIR, 'dist');
const ORIGIN = 'https://civicos.ng';
const PORT = 4173;
// Every route to prerender. Adding one is a single-line append.
const ROUTES = ['/', '/privacy', '/terms'];

async function waitForServer(url, timeoutMs = 30_000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
    } catch {
      // still coming up
    }
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(`vite preview never came up at ${url}`);
}

function outFileFor(route) {
  return route === '/' ? resolve(DIST, 'index.html') : resolve(DIST, route.slice(1), 'index.html');
}

// Rewrites the canonical URL, og:url, and twitter meta pair to match
// the current route. Without this every prerendered page would carry
// the homepage's canonical + og:url, which tells Google "these are
// duplicates of /" — the exact opposite of what we want.
function rewriteRouteMeta(html, route) {
  const absoluteUrl = `${ORIGIN}${route === '/' ? '/' : route}`;
  return html
    .replace(
      /<link rel="canonical" href="[^"]*"[^>]*>/,
      `<link rel="canonical" href="${absoluteUrl}" />`,
    )
    .replace(
      /<meta property="og:url" content="[^"]*"[^>]*>/,
      `<meta property="og:url" content="${absoluteUrl}" />`,
    );
}

async function main() {
  if (!existsSync(DIST)) {
    throw new Error(`dist not found at ${DIST} — run "vite build" first`);
  }

  console.log('[prerender] booting vite preview…');
  const preview = spawn('npx', ['vite', 'preview', '--port', String(PORT), '--strictPort'], {
    cwd: APP_DIR,
    stdio: ['ignore', 'inherit', 'inherit'],
  });

  try {
    await waitForServer(`http://localhost:${PORT}/`);

    const browser = await chromium.launch();
    const ctx = await browser.newContext();
    const page = await ctx.newPage();

    for (const route of ROUTES) {
      const url = `http://localhost:${PORT}${route}`;
      console.log(`[prerender] rendering ${route}`);
      await page.goto(url, { waitUntil: 'networkidle' });
      // React 18's concurrent commit + our `useSeo` effect land the
      // per-route title/description in a microtask AFTER load. Give
      // them a beat before serialising so the snapshot captures the
      // updated tags rather than the index.html defaults.
      await page.waitForTimeout(250);

      let html = await page.content();
      // page.content() sometimes emits without a top-level doctype
      // depending on quirks; force it so every output is valid HTML5.
      if (!/^<!doctype /i.test(html)) {
        html = '<!doctype html>\n' + html;
      }
      html = rewriteRouteMeta(html, route);

      const outFile = outFileFor(route);
      await mkdir(dirname(outFile), { recursive: true });
      await writeFile(outFile, html, 'utf8');
      const rel = outFile.replace(DIST + '/', '');
      console.log(`[prerender] → ${rel} (${html.length.toLocaleString()} bytes)`);
    }

    await browser.close();
  } finally {
    preview.kill('SIGTERM');
    // Small grace so the preview server exits cleanly.
    await new Promise((r) => setTimeout(r, 300));
  }
}

main().catch((err) => {
  console.error('[prerender] failed:', err);
  process.exit(1);
});
