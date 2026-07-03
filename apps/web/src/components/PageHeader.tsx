import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * The one masthead every dashboard page hangs off. Consciously echoes the
 * homepage's "public record" typography:
 *
 *   - A short accent rule on top (brand → accent gradient)
 *   - `§` marker + all-caps eyebrow — reads as a section register
 *   - Optional meta line on the right ("Fri · 3 Jul 2026 · Public") —
 *     masthead-style dateline that gives the app its civic-register feel
 *   - Fraunces headline in the middle row
 *   - Subtitle in Space Grotesk
 *   - Actions slot on the right of the title row
 *   - Children slot underneath for banners / notices that belong to
 *     THIS page's header (unverified nag, "join a community first", etc.)
 *
 * If a page just needs eyebrow + title + subtitle, that's the entire prop
 * surface — everything else is optional. Pages that want the standard
 * dateline can call {@link useTodayMeta} without wiring up their own
 * locale-aware date formatter.
 */
export interface PageHeaderProps {
  /** Small-caps section label; a § marker is added automatically. */
  eyebrow?: string;
  /** The main headline — set in Fraunces at the component level. */
  title: ReactNode;
  /** Optional one-sentence context under the headline. */
  subtitle?: ReactNode;
  /** Right side of the title row — usually a primary Button. */
  actions?: ReactNode;
  /**
   * Masthead-style meta line on the top-right (e.g. today's date + "Public").
   * Pass a plain string, or use {@link useTodayMeta} for the standard value.
   */
  meta?: ReactNode;
  /** Optional heading level override. Default h1. Detail pages might want h2. */
  titleAs?: 'h1' | 'h2';
  /** Notice slot that renders inside the header card, below the title row. */
  children?: ReactNode;
}

export function PageHeader({
  eyebrow,
  title,
  subtitle,
  actions,
  meta,
  titleAs = 'h1',
  children,
}: PageHeaderProps) {
  const TitleTag = titleAs;

  return (
    <header className="page-header">
      {(eyebrow || meta) && (
        <div className="page-header-top">
          {eyebrow ? (
            <p className="page-header-eyebrow">
              <span className="page-header-marker" aria-hidden="true">
                §
              </span>
              {eyebrow}
            </p>
          ) : (
            <span aria-hidden="true" />
          )}
          {meta && <p className="page-header-meta">{meta}</p>}
        </div>
      )}

      <div className="page-header-title-row">
        <div className="page-header-titles">
          <TitleTag className="page-header-title">{title}</TitleTag>
          {subtitle && <p className="page-header-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className="page-header-actions">{actions}</div>}
      </div>

      {children && <div className="page-header-notice">{children}</div>}
    </header>
  );
}

/**
 * Locale-aware "public record" dateline. Returns something like
 * "Fri · 3 Jul 2026 · Public" so it can be dropped straight into
 * <PageHeader meta={useTodayMeta()} />.
 *
 * The literal "Public" suffix comes from the pageHeader.publicRecord key
 * so translators can adjust it (Yorùbá / Igbo / Hausa / Pidgin already
 * have this string). Leaving the marker off is a fine choice too — just
 * omit the meta prop entirely.
 */
export function useTodayMeta(): string {
  const { i18n, t } = useTranslation();
  const parts = new Intl.DateTimeFormat(i18n.language, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date());
  return `${parts} · ${t('pageHeader.publicRecord')}`;
}
