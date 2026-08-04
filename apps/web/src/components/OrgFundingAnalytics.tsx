import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { HandCoins, Info } from 'lucide-react';
import {
  primaryMoney,
  useOrgFundingAnalytics,
  type AnalyticsTrendPoint,
} from '../hooks/useFundingAnalytics';
import { formatMoney, progressPercent } from '../hooks/useCampaigns';
import { EmptyState } from './EmptyState';

/**
 * An organization's own funding analytics.
 *
 * Deliberately not a mirror of the admin page. An organization already knows
 * how many campaigns it ran; what it cannot easily see is the shape of its
 * giving over time, whether donors come back, and how much of its finished
 * work it has actually accounted for.
 *
 * The two rates carry the most weight and are the easiest to misread, so both
 * state their denominator on screen rather than in a tooltip. `reportingRate`
 * is the one an organization should watch: it is the share of its completed
 * campaigns that came with an account of the money, and it is the number a
 * donor deciding whether to give again is effectively asking about.
 */
export function OrgFundingAnalytics({ orgId }: { orgId: string }) {
  const { t, i18n } = useTranslation();
  const [weeks, setWeeks] = useState(12);
  const query = useOrgFundingAnalytics(orgId, weeks);
  const a = query.data;

  if (query.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }
  if (query.isError || !a) {
    return (
      <p className="text-sm text-slate-600 dark:text-slate-300">{t('orgAnalytics.loadError')}</p>
    );
  }

  const raised = primaryMoney(a.fundsRaised);
  const avg = primaryMoney(a.donors.averageDonation);
  const money = (minor: number, cur = raised.currency) => formatMoney(minor, cur, i18n.language);

  // Nothing has been raised and nothing published: an empty state reads better
  // than a wall of zeros, which looks like failure rather than "not yet".
  if (a.campaigns.everPublished === 0 && raised.donationCount === 0) {
    return (
      <EmptyState
        icon={<HandCoins className="h-6 w-6" aria-hidden="true" />}
        title={t('orgAnalytics.emptyTitle')}
        body={t('orgAnalytics.emptyBody')}
      />
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600 dark:text-slate-300">{t('orgAnalytics.lede')}</p>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label={t('orgAnalytics.raised')} value={money(raised.amountMinor)} />
        <Stat label={t('orgAnalytics.donations')} value={raised.donationCount.toLocaleString()} />
        <Stat label={t('orgAnalytics.average')} value={money(avg.amountMinor)} />
        <Stat
          label={t('orgAnalytics.repeatDonors')}
          value={a.donors.repeatDonors.toLocaleString()}
        />
      </div>

      {/* Shown whenever the gap exists. An organization reading "3 unique
          donors" without knowing that anonymous giving is invisible will
          conclude it has three supporters. */}
      {a.donors.totalDonations > a.donors.attributableDonations && (
        <p className="flex gap-2 rounded-lg bg-slate-100 dark:bg-slate-800 p-3 text-xs leading-relaxed text-slate-600 dark:text-slate-300">
          <Info className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" aria-hidden="true" />
          {t('orgAnalytics.donorFloor', {
            attributable: a.donors.attributableDonations,
            total: a.donors.totalDonations,
          })}
        </p>
      )}

      <section className="rounded-2xl border border-slate-200 dark:border-slate-700 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {t('orgAnalytics.trend')}
          </h3>
          <select
            aria-label={t('orgAnalytics.window')}
            value={weeks}
            onChange={(e) => setWeeks(Number(e.target.value))}
            className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1 text-xs text-slate-900 dark:text-slate-100"
          >
            <option value={12}>{t('orgAnalytics.weeks', { count: 12 })}</option>
            <option value={26}>{t('orgAnalytics.weeks', { count: 26 })}</option>
            <option value={52}>{t('orgAnalytics.weeks', { count: 52 })}</option>
          </select>
        </div>
        <TrendBars points={a.trend} format={money} />
      </section>

      <div className="grid gap-3 sm:grid-cols-2">
        <Rate
          label={t('orgAnalytics.completionRate')}
          value={a.campaigns.completionRate}
          hint={t('orgAnalytics.completionHint', {
            completed: a.campaigns.completed,
            published: a.campaigns.everPublished,
          })}
        />
        {/* Amber below 60%: not a failure, but the thing to fix next. */}
        <Rate
          label={t('orgAnalytics.reportingRate')}
          value={a.campaigns.reportingRate}
          warn={a.campaigns.completed > 0 && a.campaigns.reportingRate < 60}
          hint={t('orgAnalytics.reportingHint', {
            reported: a.campaigns.reported,
            completed: a.campaigns.completed,
          })}
        />
      </div>

      {a.topCampaigns.length > 0 && (
        <section className="rounded-2xl border border-slate-200 dark:border-slate-700 p-4">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {t('orgAnalytics.byCampaign')}
          </h3>
          <ul className="mt-3 space-y-3">
            {a.topCampaigns.map((c) => (
              <li key={c.id}>
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <Link
                    to={`/campaigns/${c.slug}`}
                    className="text-sm font-semibold text-slate-900 hover:underline dark:text-slate-100"
                  >
                    {c.title}
                  </Link>
                  <span className="text-sm text-slate-700 dark:text-slate-200">
                    {money(c.raisedMinor, c.currency)}{' '}
                    <span className="text-slate-500 dark:text-slate-400">
                      {t('campaigns.ofGoal', { goal: money(c.goalMinor, c.currency) })}
                    </span>
                  </span>
                </div>
                <div
                  className="fund-progress fund-progress--feed"
                  role="progressbar"
                  aria-valuenow={progressPercent(c.raisedMinor, c.goalMinor)}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-label={t('campaigns.progressLabel', {
                    percent: progressPercent(c.raisedMinor, c.goalMinor),
                  })}
                >
                  <span
                    className="fund-progress-fill"
                    style={{ width: `${progressPercent(c.raisedMinor, c.goalMinor)}%` }}
                  />
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* From the API, so the caveats travel wherever these numbers are read
          — not only on this screen. */}
      <section className="rounded-2xl bg-slate-50 dark:bg-slate-800/60 p-4">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          {t('orgAnalytics.notesHeading')}
        </h3>
        <ul className="mt-2 list-disc space-y-1.5 pl-5 text-xs leading-relaxed text-slate-600 dark:text-slate-300">
          {a.notes.map((n, i) => (
            <li key={i}>{n}</li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-200 dark:border-slate-700 p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {label}
      </p>
      <p className="mt-1 text-xl font-bold text-slate-900 dark:text-slate-100">{value}</p>
    </div>
  );
}

function Rate({
  label,
  value,
  hint,
  warn = false,
}: {
  label: string;
  value: number;
  hint: string;
  warn?: boolean;
}) {
  return (
    <div
      className={`rounded-2xl border p-4 ${
        warn
          ? 'border-amber-300 dark:border-amber-500/50 bg-amber-50 dark:bg-amber-500/10'
          : 'border-slate-200 dark:border-slate-700'
      }`}
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        {label}
      </p>
      <p className="mt-1 text-xl font-bold text-slate-900 dark:text-slate-100">{value}%</p>
      {/* The denominator, on screen. Both rates are meaningless without it. */}
      <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">{hint}</p>
    </div>
  );
}

/**
 * Bars, not a line. Empty weeks come back as zeros, and a line joining the
 * weeks either side of a silence draws giving that never happened.
 */
function TrendBars({
  points,
  format,
}: {
  points: AnalyticsTrendPoint[];
  format: (minor: number) => string;
}) {
  const { t } = useTranslation();
  const peak = Math.max(1, ...points.map((p) => p.amountMinor));
  return (
    <>
      <div className="mt-3 flex h-24 items-end gap-[3px]">
        {points.map((p) => (
          <div
            key={p.periodStart}
            className="flex h-full flex-1 items-end rounded-sm bg-slate-100 dark:bg-slate-700/60"
            title={`${new Date(p.periodStart).toLocaleDateString()} — ${format(p.amountMinor)}`}
          >
            <span
              className="block w-full rounded-sm bg-civic-600 dark:bg-civic-400"
              style={{ height: `${Math.max((p.amountMinor / peak) * 100, 2)}%` }}
            />
          </div>
        ))}
      </div>
      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
        {t('orgAnalytics.trendCaption', { peak: format(peak) })}
      </p>
    </>
  );
}
