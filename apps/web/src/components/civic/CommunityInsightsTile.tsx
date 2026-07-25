import { useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Sparkles, RefreshCw, Users, FileText, MessageCircle } from 'lucide-react';
import { Button } from '@civicos/ui';
import { getCommunityInsights, type CommunityInsights } from '../../lib/civicai';
import { getApiError } from '../../lib/api';
import { useState } from 'react';

// CommunityInsightsTile renders a staff-only, on-demand digest of what's
// happening across a whole community — themes, sentiment, top asks,
// recommended actions. Backed by /v1/ai/community-insights, which fans
// out reads to community-service and asks Gemini for an aggregate view.
//
// On-demand (not auto-loaded): fresh calls are expensive and this is a
// "when I need it" surface. Redis-cached 1h on the server side, so a
// second click is essentially free.

interface Props {
  communityId: string | undefined;
  communityLabel?: string;
}

export function CommunityInsightsTile({ communityId, communityLabel }: Props) {
  const { t } = useTranslation();
  const [insights, setInsights] = useState<CommunityInsights | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      if (!communityId) throw new Error('missing community');
      return getCommunityInsights(communityId);
    },
    onSuccess: (data) => setInsights(data),
  });

  const errorMsg = mutation.isError
    ? (getApiError(mutation.error)?.message ?? t('civicai.insights.genericError'))
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
              {t('civicai.insights.title')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {communityLabel
                ? t('civicai.insights.subtitleWithCommunity', { community: communityLabel })
                : t('civicai.insights.subtitle')}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {insights && (
            <span className="rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400 ring-1 ring-slate-200 dark:ring-slate-700">
              {insights.cached
                ? t('civicai.insights.cachedBadge')
                : t('civicai.insights.freshBadge')}
            </span>
          )}
          <Button
            type="button"
            variant={insights ? 'secondary' : 'primary'}
            onClick={() => mutation.mutate()}
            loading={mutation.isPending}
            disabled={!communityId || mutation.isPending}
            title={!communityId ? t('civicai.insights.needCommunity') : undefined}
          >
            {insights ? (
              <span className="flex items-center gap-1.5">
                <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
                {t('civicai.insights.regenerate')}
              </span>
            ) : (
              t('civicai.insights.generate')
            )}
          </Button>
        </div>
      </header>

      {!communityId && (
        <p className="mt-3 rounded-lg bg-amber-50 dark:bg-amber-500/15 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
          {t('civicai.insights.needCommunity')}
        </p>
      )}

      {errorMsg && (
        <p className="mt-3 rounded-lg bg-rose-50 dark:bg-rose-500/15 px-3 py-2 text-xs text-rose-700 dark:text-rose-300">
          {errorMsg}
        </p>
      )}

      {insights && <InsightsBody insights={insights} />}
    </section>
  );
}

function InsightsBody({ insights }: { insights: CommunityInsights }) {
  const { t } = useTranslation();
  return (
    <div className="mt-4 space-y-4">
      <ActivityStrip activity={insights.activity} />

      <p className="text-sm leading-relaxed text-slate-800 dark:text-slate-200">{insights.tldr}</p>

      <SentimentBar mix={insights.sentimentMix} />

      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          {t('civicai.insights.themes')}
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {insights.themes.map((theme, i) => (
            <span
              key={i}
              className="rounded-full bg-civic-100 dark:bg-civic-500/20 px-2.5 py-1 text-xs font-medium text-civic-700 dark:text-civic-200"
            >
              {theme}
            </span>
          ))}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            {t('civicai.insights.topAsks')}
          </h4>
          <ul className="space-y-1.5 text-sm text-slate-800 dark:text-slate-200">
            {insights.topAsks.map((ask, i) => (
              <li key={i} className="flex items-start gap-2">
                <span className="mt-1.5 h-1.5 w-1.5 flex-none rounded-full bg-slate-400 dark:bg-slate-500" />
                <span>{ask}</span>
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            {t('civicai.insights.recommendedActions')}
          </h4>
          <ol className="list-decimal space-y-1.5 pl-5 text-sm text-slate-800 dark:text-slate-200">
            {insights.recommendedActions.map((action, i) => (
              <li key={i}>{action}</li>
            ))}
          </ol>
        </div>
      </div>

      <footer className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-200/70 dark:border-slate-700/70 pt-3 text-[11px] text-slate-500 dark:text-slate-400">
        <span>{t('civicai.insights.meta', { model: insights.model })}</span>
        <span className="rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 font-medium uppercase tracking-wide ring-1 ring-slate-200 dark:ring-slate-700">
          {t('civicai.insights.reviewBadge')}
        </span>
      </footer>
    </div>
  );
}

function ActivityStrip({ activity }: { activity: CommunityInsights['activity'] }) {
  const { t } = useTranslation();
  return (
    <div className="grid grid-cols-3 gap-2 rounded-xl bg-white/60 dark:bg-slate-900/60 p-2 ring-1 ring-slate-200/70 dark:ring-slate-700/70">
      <ActivityStat
        icon={<FileText className="h-3.5 w-3.5" />}
        label={t('civicai.insights.stats.issues')}
        value={activity.issueCount}
      />
      <ActivityStat
        icon={<Users className="h-3.5 w-3.5" />}
        label={t('civicai.insights.stats.petitions')}
        value={activity.petitionCount}
      />
      <ActivityStat
        icon={<MessageCircle className="h-3.5 w-3.5" />}
        label={t('civicai.insights.stats.comments')}
        value={activity.commentCount}
      />
    </div>
  );
}

function ActivityStat({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
}) {
  return (
    <div className="flex flex-col items-center gap-0.5 rounded-lg px-2 py-1.5">
      <span className="flex items-center gap-1 text-[10px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {icon}
        {label}
      </span>
      <span className="text-lg font-semibold tabular-nums text-slate-900 dark:text-slate-100">
        {value}
      </span>
    </div>
  );
}

function SentimentBar({ mix }: { mix: CommunityInsights['sentimentMix'] }) {
  const { t } = useTranslation();
  const pos = Math.max(mix.positive, 0);
  const neu = Math.max(mix.neutral, 0);
  const neg = Math.max(mix.negative, 0);
  const total = pos + neu + neg || 1;
  const posPct = (pos / total) * 100;
  const neuPct = (neu / total) * 100;
  const negPct = (neg / total) * 100;

  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-[11px] uppercase tracking-wide text-slate-500 dark:text-slate-400">
        <span>{t('civicai.insights.sentiment')}</span>
        <span className="tabular-nums">
          {Math.round(posPct)}% · {Math.round(neuPct)}% · {Math.round(negPct)}%
        </span>
      </div>
      <div
        className="flex h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800"
        role="img"
        aria-label={t('civicai.insights.sentimentAria', {
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
