import { useTranslation } from 'react-i18next';
import { Waves, ExternalLink } from 'lucide-react';
import type { CommunityFloodAlert, FloodSeverity } from '@civicos/types';
import { useFloodAlerts } from '../../hooks/useFloodAlerts';

/**
 * Shows the flood forecast currently attached to a community.
 *
 * Three rules this component exists to hold:
 *
 * 1. **It is never CivicOS's forecast.** Google Research runs the models;
 *    CivicOS matched their output to this community. The attribution sits
 *    in the banner itself, not a page footer, so nobody can screenshot a
 *    warning without the source attached.
 * 2. **There is no reassuring state.** With nothing forecast the component
 *    renders nothing at all. A green "no flooding expected" panel would be
 *    a safety claim CivicOS cannot stand behind — the absence of a warning
 *    is not the presence of safety.
 * 3. **It does not give advice.** CivicOS says what the forecast says and
 *    points at NEMA and NiMet, who are the ones authorised to tell people
 *    what to do.
 */
const TONE: Record<FloodSeverity, string> = {
  EXTREME:
    'border-red-300 bg-red-50 text-red-950 dark:border-red-500/50 dark:bg-red-950/40 dark:text-red-100',
  SEVERE:
    'border-orange-300 bg-orange-50 text-orange-950 dark:border-orange-500/50 dark:bg-orange-950/40 dark:text-orange-100',
  ABOVE_NORMAL:
    'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-500/50 dark:bg-amber-950/40 dark:text-amber-100',
};

export function FloodAlertBanner({ communityId }: { communityId: string | undefined }) {
  const { t, i18n } = useTranslation();
  const { data } = useFloodAlerts(communityId);

  const alerts = data?.alerts ?? [];
  // Nothing forecast, feature off, community unlocated, or the request
  // failed — all render nothing. See rule 2 above.
  if (alerts.length === 0) return null;

  // Already ordered worst-first by the API.
  const worst = alerts[0];
  const others = alerts.length - 1;

  return (
    <aside className={`rounded-2xl border p-4 ${TONE[worst.severity]}`} role="status">
      <div className="flex items-start gap-3">
        <Waves className="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold">{t(`floodAlert.title.${worst.severity}`)}</h2>

          <p className="mt-1 text-sm">
            {t('floodAlert.body', {
              river: worst.river ?? t('floodAlert.aRiverNearby'),
              distance: Math.round(worst.distanceKm),
            })}
            {worst.trend === 'RISE' && ` ${t('floodAlert.rising')}`}
          </p>

          {worst.forecastEndAt && (
            <p className="mt-1 text-sm opacity-90">
              {t('floodAlert.window', {
                date: new Date(worst.forecastEndAt).toLocaleDateString(i18n.language, {
                  day: 'numeric',
                  month: 'short',
                }),
              })}
            </p>
          )}

          {others > 0 && (
            <p className="mt-1 text-xs opacity-80">{t('floodAlert.more', { count: others })}</p>
          )}

          {/* Attribution and the official channel, in the banner rather
              than a footer — see rule 1. */}
          <p className="mt-3 text-xs opacity-90">
            {t('floodAlert.attribution')}{' '}
            <a
              className="inline-flex items-center gap-0.5 font-semibold underline underline-offset-2"
              href="https://sites.research.google/floods/"
              target="_blank"
              rel="noreferrer noopener"
            >
              {t('floodAlert.viewFloodHub')}
              <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          </p>
          <p className="mt-1 text-xs font-medium opacity-90">{t('floodAlert.official')}</p>
        </div>
      </div>
    </aside>
  );
}

/** Exported for tests and for any other surface that needs the same ordering. */
export function worstSeverity(alerts: CommunityFloodAlert[]): FloodSeverity | null {
  const rank: Record<FloodSeverity, number> = { EXTREME: 3, SEVERE: 2, ABOVE_NORMAL: 1 };
  return alerts.reduce<FloodSeverity | null>(
    (acc, a) => (acc === null || rank[a.severity] > rank[acc] ? a.severity : acc),
    null,
  );
}
