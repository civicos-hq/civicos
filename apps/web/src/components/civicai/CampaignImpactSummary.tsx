import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles, CircleAlert } from 'lucide-react';
import { Button } from '@civicos/ui';
import { summarizeCampaignImpact, type CampaignImpact } from '../../lib/civicai';
import { getApiError } from '../../lib/api';

/**
 * Reads back what a campaign has published — spend records, milestones,
 * progress updates — and summarises what a donor can actually tell from it.
 *
 * Aimed at the organization, not the public. It is a rehearsal: run it
 * before writing a completion report and you see your own campaign the way
 * someone who gave money sees it.
 *
 * `gaps` is why this is worth having. Anything can produce a flattering
 * paragraph from a spend table; the useful half is the list of things a
 * reader *cannot* tell, which is exactly what an organization closest to
 * the work is least able to notice. So gaps are rendered at least as
 * prominently as the summary, never folded away behind it.
 *
 * On demand rather than automatic: it costs a Gemini call and reads the
 * whole campaign, so it runs when someone asks.
 */
export function CampaignImpactSummary({ campaignId }: { campaignId: string }) {
  const { t } = useTranslation();
  const [impact, setImpact] = useState<CampaignImpact | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function run() {
    setPending(true);
    setError(null);
    try {
      setImpact(await summarizeCampaignImpact(campaignId));
    } catch (err) {
      setError(getApiError(err)?.message ?? t('campaignImpact.error'));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="mt-4 rounded-xl border border-slate-200 p-4 dark:border-slate-700">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="flex items-center gap-1.5 text-sm font-semibold text-slate-800 dark:text-slate-100">
            <Sparkles className="h-4 w-4 text-civic-500" aria-hidden="true" />
            {t('campaignImpact.heading')}
          </h3>
          <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
            {t('campaignImpact.lede')}
          </p>
        </div>
        <Button size="sm" variant="secondary" onClick={run} loading={pending}>
          {impact ? t('campaignImpact.rerun') : t('campaignImpact.run')}
        </Button>
      </div>

      {error && <p className="mt-3 text-sm text-red-600 dark:text-red-400">{error}</p>}

      {impact && (
        <div className="mt-4 space-y-4">
          <p className="text-sm leading-relaxed text-slate-700 dark:text-slate-200">
            {impact.summary}
          </p>

          {impact.highlights.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                {t('campaignImpact.highlights')}
              </h4>
              <ul className="mt-1.5 grid gap-1 text-sm text-slate-700 dark:text-slate-200">
                {impact.highlights.map((h, i) => (
                  <li key={i}>• {h}</li>
                ))}
              </ul>
            </div>
          )}

          {/* The half that does the work. Rendered in the same weight as an
              error, because a gap here is what a donor will notice. */}
          {impact.gaps.length > 0 && (
            <div className="rounded-lg bg-amber-50 p-3 dark:bg-amber-950/40">
              <h4 className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-amber-900 dark:text-amber-200">
                <CircleAlert className="h-3.5 w-3.5" aria-hidden="true" />
                {t('campaignImpact.gaps')}
              </h4>
              <ul className="mt-1.5 grid gap-1 text-sm text-amber-900 dark:text-amber-100">
                {impact.gaps.map((g, i) => (
                  <li key={i}>• {g}</li>
                ))}
              </ul>
            </div>
          )}

          <p className="text-[10px] uppercase tracking-wide text-slate-400 dark:text-slate-500">
            {t('campaignImpact.badge')}
          </p>
        </div>
      )}
    </div>
  );
}
