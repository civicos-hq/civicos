import { useEffect } from 'react';

interface SeoInput {
  /** Full <title> for the current route. Keep it under ~60 chars so it doesn't truncate in SERPs. */
  title: string;
  /** Meta description for the current route. Keep it under ~155 chars. */
  description?: string;
}

/**
 * useSeo syncs the browser's `<title>` and `<meta name="description">`
 * with per-route values. Called once per top-level route component on
 * mount; the tag values persist until the next route overrides them,
 * which matches how a user actually experiences browser tabs.
 *
 * Deliberately does NOT restore previous values on unmount — restore
 * causes a one-frame flicker of the old title during route transitions
 * under React 18 concurrent rendering. Instead, every top-level route
 * calls useSeo with its own values so there's always a fresh set.
 *
 * Not a replacement for prerendered `<head>` tags — crawlers running
 * without JS still see the defaults from `index.html`. This is the
 * browser-tab + share-URL correctness layer.
 */
export function useSeo({ title, description }: SeoInput): void {
  useEffect(() => {
    document.title = title;
    if (description) {
      const meta = document.querySelector<HTMLMetaElement>('meta[name="description"]');
      if (meta) meta.content = description;
    }
  }, [title, description]);
}
