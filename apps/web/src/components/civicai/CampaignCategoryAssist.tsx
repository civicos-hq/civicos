import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles, TriangleAlert } from 'lucide-react';
import { classifyCampaign, type CampaignClassification } from '../../lib/civicai';
import { categoryKey, type CampaignCategory } from '../../hooks/useCampaigns';

/**
 * Suggests a category — and whether this reads as an emergency — while
 * someone writes a campaign.
 *
 * Mirrors the chip `IssuesPage` shows for `classifyIssue`: debounced,
 * cancellable, silent on failure. CivicAI is additive here, and an author
 * who ignores it must be able to submit exactly as before.
 *
 * The emergency flag is the part with teeth. It changes how prominently a
 * campaign is surfaced, so this only ever *offers* it — nothing is applied
 * without a click, and the reasoning is attached to the control so an
 * author can see why before accepting.
 */
export function CampaignCategoryAssist({
  title,
  description,
  category,
  categoryTouched,
  isEmergency,
  onApplyCategory,
  onApplyEmergency,
}: {
  title: string;
  description: string;
  category: CampaignCategory;
  categoryTouched: boolean;
  isEmergency: boolean;
  onApplyCategory: (next: CampaignCategory) => void;
  onApplyEmergency: (next: boolean) => void;
}) {
  const { t } = useTranslation();
  const [suggestion, setSuggestion] = useState<CampaignClassification | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    // Below this there is nothing for the model to reason about, and
    // calling anyway burns a Gemini request per keystroke-pause.
    if (title.trim().length < 5 || description.trim().length < 20) {
      setSuggestion(null);
      setPending(false);
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      setPending(true);
      try {
        setSuggestion(await classifyCampaign({ title, description }, controller.signal));
      } catch {
        // Fail quiet — the author picks a category themselves.
      } finally {
        setPending(false);
      }
    }, 700);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [title, description]);

  if (pending && !suggestion) {
    return (
      <p className="mt-1.5 flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
        <Sparkles className="h-3.5 w-3.5 animate-pulse text-civic-500" aria-hidden="true" />
        {t('campaignAssist.pending')}
      </p>
    );
  }
  if (!suggestion) return null;

  const suggested = suggestion.category as CampaignCategory;
  const categoryMatches = suggested === category;
  // Only worth raising if it disagrees with where the form currently is.
  const emergencyDiffers = suggestion.isEmergency !== isEmergency;

  // Author already chose, and CivicAI agrees on both counts — say nothing.
  if (categoryMatches && categoryTouched && !emergencyDiffers) return null;

  return (
    <div className="mt-1.5 space-y-1.5">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <Sparkles className="h-3.5 w-3.5 text-civic-500" aria-hidden="true" />
        <span className="font-medium text-slate-600 dark:text-slate-300">
          {t('campaignAssist.suggests')}
        </span>
        {categoryMatches ? (
          <span className="rounded-full bg-emerald-50 px-2 py-0.5 font-semibold text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">
            {t(`campaigns.categories.${categoryKey(suggested)}`, suggested)}
          </span>
        ) : (
          <button
            type="button"
            onClick={() => onApplyCategory(suggested)}
            className="rounded-full border border-civic-300 bg-civic-50 px-2 py-0.5 font-semibold text-civic-700 hover:bg-civic-100 dark:border-civic-500/60 dark:bg-civic-500/15 dark:text-civic-200 dark:hover:bg-civic-500/25"
            title={suggestion.reasoning}
          >
            {t(`campaigns.categories.${categoryKey(suggested)}`, suggested)} ·{' '}
            {t('campaignAssist.apply')}
          </button>
        )}
        <span className="ml-auto text-[10px] uppercase tracking-wide text-slate-400 dark:text-slate-500">
          {t('campaignAssist.badge')}
        </span>
      </div>

      {emergencyDiffers && (
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <TriangleAlert className="h-3.5 w-3.5 text-amber-500" aria-hidden="true" />
          <span className="text-slate-600 dark:text-slate-300">
            {suggestion.isEmergency
              ? t('campaignAssist.looksUrgent')
              : t('campaignAssist.looksNotUrgent')}
          </span>
          <button
            type="button"
            onClick={() => onApplyEmergency(suggestion.isEmergency)}
            className="rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 font-semibold text-amber-800 hover:bg-amber-100 dark:border-amber-500/60 dark:bg-amber-500/15 dark:text-amber-200 dark:hover:bg-amber-500/25"
            title={suggestion.reasoning}
          >
            {suggestion.isEmergency
              ? t('campaignAssist.markUrgent')
              : t('campaignAssist.unmarkUrgent')}
          </button>
        </div>
      )}
    </div>
  );
}
