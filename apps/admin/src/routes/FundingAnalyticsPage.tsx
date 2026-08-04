import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Info } from 'lucide-react';
import { apiGet } from '../lib/api';

/**
 * Platform-wide funding analytics.
 *
 * The design problem here is not charting, it is honesty. These figures get
 * screenshotted into board packs and funding applications, where they arrive
 * without whatever caveat was on screen. So the caveats are rendered as part
 * of the page rather than as a footnote, and the API returns them in the
 * payload for the same reason.
 *
 * Two things the spec asks for are deliberately absent, and the page says so
 * rather than leaving a reader to wonder:
 *
 *  - **People helped.** Nothing in the record measures it.
 *  - **Funds withdrawn / remaining balance.** CivicOS never holds the money.
 */

interface Money {
  currency: string;
  amountMinor: number;
  donationCount: number;
}
interface TrendPoint {
  periodStart: string;
  amountMinor: number;
  count: number;
}
interface Analytics {
  totalCampaigns: number;
  fundsRaised: Money[];
  donors: {
    uniqueDonors: number;
    repeatDonors: number;
    attributableDonations: number;
    totalDonations: number;
    averageDonation: Money[];
  };
  campaigns: {
    total: number;
    byStatus: Record<string, number>;
    everPublished: number;
    completed: number;
    reported: number;
    completionRate: number;
    reportingRate: number;
  };
  organizations: {
    total: number;
    verified: number;
    fundingEligible: number;
    withPublishedCampaigns: number;
  };
  countries: { country: string; organizations: number }[];
  categories: { category: string; campaigns: number; raisedMinor: number; currency: string }[];
  emergency: {
    campaigns: number;
    fundsRaised: Money[];
    medianHoursToFirstDonation: number | null;
    fundedWithin7Days: number;
  };
  review: {
    reviewed: number;
    averageHours: number | null;
    medianHours: number | null;
    awaitingReview: number;
    oldestWaitingHours: number | null;
  };
  trend: TrendPoint[];
  generatedAt: string;
  notes: string[];
}

function money(minor: number, currency = 'NGN') {
  return new Intl.NumberFormat('en-NG', {
    style: 'currency',
    currency,
    currencyDisplay: 'narrowSymbol',
    maximumFractionDigits: 0,
  }).format(minor / 100);
}

function primary(list: Money[]): Money {
  return list?.[0] ?? { currency: 'NGN', amountMinor: 0, donationCount: 0 };
}

function hours(h: number | null) {
  if (h === null || h === undefined) return '—';
  if (h < 1) return `${Math.round(h * 60)} min`;
  if (h < 48) return `${h.toFixed(1)} hrs`;
  return `${(h / 24).toFixed(1)} days`;
}

export function FundingAnalyticsPage() {
  const [weeks, setWeeks] = useState(12);
  const query = useQuery({
    queryKey: ['funding-analytics', weeks],
    queryFn: () =>
      apiGet<{ analytics: Analytics }>(`/api/v1/admin/funding-analytics?weeks=${weeks}`),
  });

  const a = query.data?.analytics;
  const raised = primary(a?.fundsRaised ?? []);
  const avg = primary(a?.donors?.averageDonation ?? []);
  const peakTrend = Math.max(1, ...(a?.trend ?? []).map((p) => p.amountMinor));

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Money</p>
        <h1 className="admin-page-title">Funding analytics</h1>
        <p className="admin-page-sub">
          Campaigns and the donation ledger across the whole platform. Every money figure is what
          settled <em>through CivicOS</em> — donations go straight to each organization's own bank
          account, so none of this is what they hold, have spent, or have left.
        </p>
      </header>

      {query.isLoading && <p className="admin-empty">Loading…</p>}
      {query.isError && <p className="admin-error">Could not load analytics.</p>}

      {a && (
        <>
          <div className="grid gap-3 md:grid-cols-4 mb-4">
            <Metric label="Total raised" value={money(raised.amountMinor, raised.currency)} />
            <Metric label="Donations" value={raised.donationCount.toLocaleString()} />
            <Metric label="Average donation" value={money(avg.amountMinor, avg.currency)} />
            <Metric label="Campaigns" value={a.totalCampaigns.toLocaleString()} />
          </div>

          <Panel title="Campaign performance">
            <div className="grid gap-3 md:grid-cols-4">
              <Metric label="Ever published" value={a.campaigns.everPublished.toLocaleString()} />
              <Metric label="Completed" value={a.campaigns.completed.toLocaleString()} />
              <Metric
                label="Completion rate"
                value={`${a.campaigns.completionRate}%`}
                hint="Completed or reported, over campaigns that ever published."
              />
              {/* The most telling number on the page: the share of finished
                  work that came with an account of the money. */}
              <Metric
                label="Filed a final report"
                value={`${a.campaigns.reportingRate}%`}
                tone={a.campaigns.reportingRate < 60 ? 'pending' : 'success'}
                hint="Of completed campaigns. This is the share of finished work that came with an account of the money."
              />
            </div>
          </Panel>

          <Panel title="Donation trend">
            <div className="admin-table-toolbar" style={{ borderTop: 'none' }}>
              <select
                className="admin-table-search"
                style={{ flex: '0 0 160px' }}
                value={weeks}
                onChange={(e) => setWeeks(Number(e.target.value))}
              >
                <option value={12}>Last 12 weeks</option>
                <option value={26}>Last 26 weeks</option>
                <option value={52}>Last 52 weeks</option>
              </select>
            </div>
            {/* Bars rather than a line: empty weeks are returned as zeros and
                a line drawn through them would imply giving that did not
                happen. */}
            <div className="admin-trend">
              {a.trend.map((p) => (
                <div
                  key={p.periodStart}
                  className="admin-trend-bar"
                  title={`Week of ${new Date(p.periodStart).toLocaleDateString()} — ${money(p.amountMinor, raised.currency)} from ${p.count}`}
                >
                  <span style={{ height: `${(p.amountMinor / peakTrend) * 100}%` }} />
                </div>
              ))}
            </div>
            <p className="admin-trend-caption">
              Weekly, oldest first. Peak week {money(peakTrend, raised.currency)}.
            </p>
          </Panel>

          <div className="grid gap-4 md:grid-cols-2">
            <Panel title="Donors">
              <dl className="admin-kv">
                <Row k="Unique donors" v={a.donors.uniqueDonors.toLocaleString()} />
                <Row k="Repeat donors" v={a.donors.repeatDonors.toLocaleString()} />
                <Row k="Total donations" v={a.donors.totalDonations.toLocaleString()} />
              </dl>
              {/* Rendered whenever it bites, not tucked into a tooltip: the
                  two counts above are a floor, and a reader who does not know
                  that will treat them as a total. */}
              {a.donors.totalDonations > a.donors.attributableDonations && (
                <p className="admin-caveat">
                  <Info className="mr-1 inline h-3.5 w-3.5" aria-hidden="true" />
                  {a.donors.attributableDonations} of {a.donors.totalDonations} donations can be
                  tied to a person. The rest were given while signed out, so donor counts are a
                  floor, not a total.
                </p>
              )}
            </Panel>

            <Panel title="Organizations">
              <dl className="admin-kv">
                <Row k="Registered" v={a.organizations.total.toLocaleString()} />
                <Row k="Verified" v={a.organizations.verified.toLocaleString()} />
                <Row
                  k="Able to take money"
                  v={a.organizations.fundingEligible.toLocaleString()}
                  hint="Verified AND with a connected payout account."
                />
                <Row
                  k="Have published a campaign"
                  v={a.organizations.withPublishedCampaigns.toLocaleString()}
                />
              </dl>
            </Panel>

            <Panel title="Review queue">
              <dl className="admin-kv">
                <Row k="Campaigns reviewed" v={a.review.reviewed.toLocaleString()} />
                <Row k="Median wait" v={hours(a.review.medianHours)} />
                <Row k="Average wait" v={hours(a.review.averageHours)} />
                <Row k="Waiting now" v={a.review.awaitingReview.toLocaleString()} />
                {/* An average stays comfortable while one campaign sits for a
                    fortnight. This is the number that catches that. */}
                <Row k="Longest currently waiting" v={hours(a.review.oldestWaitingHours)} />
              </dl>
            </Panel>

            <Panel title="Emergency appeals">
              <dl className="admin-kv">
                <Row k="Published" v={a.emergency.campaigns.toLocaleString()} />
                <Row
                  k="Raised"
                  v={money(primary(a.emergency.fundsRaised).amountMinor, raised.currency)}
                />
                <Row
                  k="Median time to first donation"
                  v={hours(a.emergency.medianHoursToFirstDonation)}
                />
                <Row
                  k="Fully funded within 7 days"
                  v={a.emergency.fundedWithin7Days.toLocaleString()}
                />
              </dl>
            </Panel>
          </div>

          <Panel title="Categories">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Category</th>
                  <th>Campaigns</th>
                  <th>Raised</th>
                </tr>
              </thead>
              <tbody>
                {a.categories.map((c) => (
                  <tr key={c.category}>
                    <td>{c.category.replace(/_/g, ' ').toLowerCase()}</td>
                    <td>{c.campaigns}</td>
                    <td>{money(c.raisedMinor, c.currency)}</td>
                  </tr>
                ))}
                {a.categories.length === 0 && (
                  <tr>
                    <td colSpan={3}>No published campaigns yet.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </Panel>

          <Panel title="Countries">
            <dl className="admin-kv">
              {a.countries.map((c) => (
                <Row key={c.country} k={c.country} v={c.organizations.toLocaleString()} />
              ))}
            </dl>
          </Panel>

          {/* The notes come from the API so they travel with the numbers
              wherever they are consumed, not only here. */}
          <section className="admin-table-shell">
            <div className="admin-table-toolbar">
              <strong className="text-sm">How to read these</strong>
            </div>
            <ul className="admin-notes">
              {a.notes.map((n, i) => (
                <li key={i}>{n}</li>
              ))}
              <li>
                "Funds withdrawn" and "remaining balance" are not shown. CivicOS never holds the
                money, so it has no way to know either.
              </li>
            </ul>
            <p className="admin-trend-caption">
              Generated {new Date(a.generatedAt).toLocaleString()}
            </p>
          </section>
        </>
      )}
    </>
  );
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="admin-table-shell" style={{ marginBottom: 16 }}>
      <div className="admin-table-toolbar">
        <strong className="text-sm">{title}</strong>
      </div>
      <div className="p-4">{children}</div>
    </section>
  );
}

function Metric({
  label,
  value,
  tone = 'neutral',
  hint,
}: {
  label: string;
  value: string;
  tone?: 'neutral' | 'pending' | 'success';
  hint?: string;
}) {
  return (
    <div className={`admin-metric-card admin-metric-card-${tone}`}>
      <div className="admin-metric-label">{label}</div>
      <div className="admin-metric-value">{value}</div>
      {hint && <p className="admin-metric-hint">{hint}</p>}
    </div>
  );
}

function Row({ k, v, hint }: { k: string; v: string; hint?: string }) {
  return (
    <>
      <dt>
        {k}
        {hint && <span className="admin-kv-hint">{hint}</span>}
      </dt>
      <dd>{v}</dd>
    </>
  );
}
