import { type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, Sparkles } from 'lucide-react';
import { Button } from '@civicos/ui';
import type { AIProvenance } from '../../lib/civicai';

/**
 * The shared shell for every campaign AI surface.
 *
 * All of these produce text about money other people gave, from claims
 * CivicOS cannot verify. Three things follow, and putting them in one
 * component is the only way they stay true of all six:
 *
 *  - The output is always labelled as AI-written, with the model named.
 *  - Nothing is applied by generating it. Accepting a draft is a separate,
 *    explicit click, so a distracted org admin cannot publish words they
 *    never read.
 *  - Warnings render as prominently as the draft itself. The warnings are
 *    the most useful part — they are what a reviewer will ask for, heard
 *    two days earlier.
 */

export function AIBadge({ provenance }: { provenance?: AIProvenance }) {
  const { t } = useTranslation();
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-civic-100 dark:bg-civic-500/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-civic-700 dark:text-civic-200">
      <Sparkles className="h-3 w-3" aria-hidden="true" />
      {t('campaignAi.reviewBadge')}
      {provenance?.model && <span className="font-normal normal-case">· {provenance.model}</span>}
    </span>
  );
}

/**
 * Things the organization should fix before this goes out.
 *
 * Styled as a caution rather than an error: none of them blocks anything,
 * and most campaigns will have at least one. Rendered above the draft body
 * on purpose — an admin who reads only the first thing on screen should read
 * this.
 */
export function AIWarnings({ warnings }: { warnings: string[] }) {
  const { t } = useTranslation();
  if (!warnings || warnings.length === 0) return null;
  return (
    <div className="rounded-lg border border-amber-300 dark:border-amber-500/50 bg-amber-50 dark:bg-amber-500/10 p-3">
      <p className="flex items-center gap-1.5 text-xs font-semibold text-amber-900 dark:text-amber-100">
        <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
        {t('campaignAi.warningsHeading')}
      </p>
      <ul className="mt-1.5 list-disc space-y-1 pl-5 text-sm text-amber-900 dark:text-amber-100">
        {warnings.map((w, i) => (
          <li key={i}>{w}</li>
        ))}
      </ul>
    </div>
  );
}

interface Props {
  /** Shown above the brief input. */
  title: string;
  /** One line on what this will produce. */
  hint: string;
  brief: string;
  onBriefChange: (v: string) => void;
  placeholder: string;
  minBrief: number;
  onGenerate: () => void;
  isPending: boolean;
  error?: string;
  /** The rendered draft, if there is one. */
  children?: ReactNode;
  /**
   * Provenance of the rendered draft. Required whenever `children` is set:
   * principle 2 of the CivicAI plan is that every AI output is
   * provenance-tagged, and a badge that does not name the model only tells
   * the reader that *something* generated this.
   */
  provenance?: AIProvenance;
  /** Applying is always separate from generating. */
  onApply?: () => void;
  applyLabel?: string;
}

export function CampaignAIPanel({
  title,
  hint,
  brief,
  onBriefChange,
  placeholder,
  minBrief,
  onGenerate,
  isPending,
  error,
  children,
  provenance,
  onApply,
  applyLabel,
}: Props) {
  const { t } = useTranslation();
  const tooShort = brief.trim().length < minBrief;

  return (
    <section className="rounded-xl border border-civic-200 dark:border-civic-500/30 bg-civic-50/60 dark:bg-civic-500/10 p-4 space-y-3">
      <div>
        <h3 className="flex items-center gap-1.5 text-sm font-semibold text-slate-900 dark:text-slate-100">
          <Sparkles className="h-4 w-4 text-civic-700 dark:text-civic-300" aria-hidden="true" />
          {title}
        </h3>
        <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-300">{hint}</p>
      </div>

      <textarea
        name="aiBrief"
        rows={3}
        value={brief}
        onChange={(e) => onBriefChange(e.target.value)}
        placeholder={placeholder}
        className="w-full rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
      />

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          onClick={onGenerate}
          loading={isPending}
          disabled={isPending || tooShort}
        >
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
          {t('campaignAi.generate')}
        </Button>
        {tooShort && (
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {t('campaignAi.briefTooShort', { count: minBrief })}
          </span>
        )}
      </div>

      {/* Fail open: the whole point is that the person can still do the work
          by hand, so an error says so rather than looking like a dead end. */}
      {error && (
        <p className="rounded-lg border border-red-300 dark:border-red-500/50 bg-red-50 dark:bg-red-500/10 p-2.5 text-sm text-red-900 dark:text-red-100">
          {error}
        </p>
      )}

      {children && (
        <div className="space-y-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/70 p-3">
          <AIBadge provenance={provenance} />
          {children}
          {onApply && (
            <div className="flex justify-end">
              <Button type="button" size="sm" variant="secondary" onClick={onApply}>
                {applyLabel ?? t('campaignAi.apply')}
              </Button>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
