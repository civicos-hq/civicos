import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Sparkles, RefreshCw, MessageSquare } from 'lucide-react';
import { Button } from '@civicos/ui';
import {
  summarizeDiscussion,
  type DiscussionSummary,
  type SummarizableResource,
} from '../../lib/civicai';
import { getApiError } from '../../lib/api';

// DiscussionSummaryPanel renders a "Summarize discussion" affordance for
// petition and issue detail pages. Only staff-role users should see this;
// the caller is expected to gate the mount point on role membership.
//
// The panel is intentionally collapsible: a fresh Gemini call costs real
// money, so we don't auto-fire on page load. The user clicks, we call, we
// render. Every visible AI output carries a "review before publishing"
// affordance so admins remember to check the summary before acting.

interface Props {
  resource: SummarizableResource;
  resourceId: string;
  // commentCount lets the panel show a helpful "not enough discussion yet"
  // hint without a Gemini call when the thread is basically empty.
  commentCount: number;
}

const MIN_COMMENTS_FOR_SUMMARY = 2;

export function DiscussionSummaryPanel({ resource, resourceId, commentCount }: Props) {
  const { t } = useTranslation();
  const [summary, setSummary] = useState<DiscussionSummary | null>(null);

  const mutation = useMutation({
    mutationFn: () => summarizeDiscussion(resource, resourceId),
    onSuccess: (data) => setSummary(data),
  });

  const notEnoughComments = commentCount < MIN_COMMENTS_FOR_SUMMARY;
  const isPending = mutation.isPending;
  const errorMsg = mutation.isError
    ? (getApiError(mutation.error)?.message ?? t('civicai.summary.genericError'))
    : null;

  return (
    <section className="rounded-2xl border border-civic-200/70 dark:border-civic-500/30 bg-gradient-to-br from-civic-50/60 to-white dark:from-civic-500/10 dark:to-slate-900 p-4 md:p-5 shadow-sm">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-8 w-8 items-center justify-center rounded-full bg-civic-100 dark:bg-civic-500/25 text-civic-700 dark:text-civic-200">
            <Sparkles className="h-4 w-4" aria-hidden="true" />
          </span>
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {t('civicai.summary.title')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('civicai.summary.subtitle')}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {summary && (
            <span className="rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400 ring-1 ring-slate-200 dark:ring-slate-700">
              {summary.cached ? t('civicai.summary.cachedBadge') : t('civicai.summary.freshBadge')}
            </span>
          )}
          <Button
            type="button"
            variant={summary ? 'secondary' : 'primary'}
            onClick={() => mutation.mutate()}
            loading={isPending}
            disabled={notEnoughComments || isPending}
            title={
              notEnoughComments
                ? t('civicai.summary.needMoreComments', { min: MIN_COMMENTS_FOR_SUMMARY })
                : undefined
            }
          >
            {summary ? (
              <span className="flex items-center gap-1.5">
                <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
                {t('civicai.summary.regenerate')}
              </span>
            ) : (
              t('civicai.summary.generate')
            )}
          </Button>
        </div>
      </header>

      {notEnoughComments && !summary && (
        <p className="mt-3 flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
          <MessageSquare className="h-3.5 w-3.5" aria-hidden="true" />
          {t('civicai.summary.needMoreComments', { min: MIN_COMMENTS_FOR_SUMMARY })}
        </p>
      )}

      {errorMsg && (
        <p className="mt-3 rounded-lg bg-rose-50 dark:bg-rose-500/15 px-3 py-2 text-xs text-rose-700 dark:text-rose-300">
          {errorMsg}
        </p>
      )}

      {summary && <SummaryBody summary={summary} />}
    </section>
  );
}

function SummaryBody({ summary }: { summary: DiscussionSummary }) {
  const { t } = useTranslation();
  return (
    <div className="mt-4 space-y-4">
      <p className="text-sm leading-relaxed text-slate-800 dark:text-slate-200">{summary.tldr}</p>

      <SentimentBar sentiment={summary.sentiment} />

      <div className="grid gap-4 md:grid-cols-2">
        <SummaryList heading={t('civicai.summary.themes')} items={summary.themes} tone="civic" />
        <SummaryList heading={t('civicai.summary.topAsks')} items={summary.topAsks} tone="slate" />
      </div>

      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          {t('civicai.summary.recommendedActions')}
        </h4>
        <ol className="list-decimal space-y-1.5 pl-5 text-sm text-slate-800 dark:text-slate-200">
          {summary.recommendedActions.map((action, i) => (
            <li key={i}>{action}</li>
          ))}
        </ol>
      </div>

      <footer className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-200/70 dark:border-slate-700/70 pt-3 text-[11px] text-slate-500 dark:text-slate-400">
        <span>
          {t('civicai.summary.meta', {
            count: summary.commentsAnalyzed,
            model: summary.model,
          })}
        </span>
        <span className="rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 font-medium uppercase tracking-wide ring-1 ring-slate-200 dark:ring-slate-700">
          {t('civicai.summary.reviewBadge')}
        </span>
      </footer>
    </div>
  );
}

function SentimentBar({ sentiment }: { sentiment: DiscussionSummary['sentiment'] }) {
  const { t } = useTranslation();
  // Percentages for the visual bar. Round for display but drive layout
  // from the raw fractions so tiny slices still get a visible sliver.
  const pos = Math.max(sentiment.positive, 0);
  const neu = Math.max(sentiment.neutral, 0);
  const neg = Math.max(sentiment.negative, 0);
  const total = pos + neu + neg || 1;
  const posPct = (pos / total) * 100;
  const neuPct = (neu / total) * 100;
  const negPct = (neg / total) * 100;

  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-[11px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        <span>{t('civicai.summary.sentiment')}</span>
        <span className="tabular-nums">
          {Math.round(posPct)}% · {Math.round(neuPct)}% · {Math.round(negPct)}%
        </span>
      </div>
      <div
        className="flex h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
        role="img"
        aria-label={t('civicai.summary.sentimentAria', {
          positive: Math.round(posPct),
          neutral: Math.round(neuPct),
          negative: Math.round(negPct),
        })}
      >
        <span className="bg-emerald-500" style={{ width: `${posPct}%` }} />
        <span className="bg-slate-400 dark:bg-slate-500" style={{ width: `${neuPct}%` }} />
        <span className="bg-rose-500" style={{ width: `${negPct}%` }} />
      </div>
    </div>
  );
}

function SummaryList({
  heading,
  items,
  tone,
}: {
  heading: string;
  items: string[];
  tone: 'civic' | 'slate';
}) {
  if (!items?.length) return null;
  const bulletClass =
    tone === 'civic' ? 'bg-civic-500/70 dark:bg-civic-400/70' : 'bg-slate-400 dark:bg-slate-500';
  return (
    <div>
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {heading}
      </h4>
      <ul className="space-y-1.5 text-sm text-slate-800 dark:text-slate-200">
        {items.map((item, i) => (
          <li key={i} className="flex items-start gap-2">
            <span className={`mt-1.5 h-1.5 w-1.5 flex-none rounded-full ${bulletClass}`} />
            <span>{item}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
