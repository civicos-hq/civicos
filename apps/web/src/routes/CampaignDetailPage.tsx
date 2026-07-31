import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Circle, Clock, MapPin, ShieldCheck } from 'lucide-react';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';
import {
  accountedPercent,
  formatMoney,
  formatMoneyExact,
  progressPercent,
  useCampaignSpend,
  useCanManageCampaign,
  useCampaignUpdates,
  usePublicCampaign,
  usePublicDonations,
  type FundingUpdate,
  type PublicCampaignDetail,
  type PublicMilestone,
  type SpendRecord,
} from '../hooks/useCampaigns';
import { DonateForm } from '../components/DonateForm';
import { CampaignConsole } from '../components/CampaignConsole';

// Public campaign page. Shows the ask, the spend plan, and — since Phase 3 —
// the donate flow and public donor list. Phase 4 extends it with the full
// funds-flow dashboard (withdrawn / remaining, receipts, reports).
//
// Donations are only offered on campaigns that can actually take money.
// PAUSED is excluded: once funds settle straight to the organization,
// pausing is the only governance lever left, so the UI must respect it.

const MILESTONE_ICON = {
  COMPLETED: CheckCircle2,
  IN_PROGRESS: Clock,
  PLANNED: Circle,
} as const;

function MilestoneRow({
  m,
  currency,
  locale,
}: {
  m: PublicMilestone;
  currency: string;
  locale: string;
}) {
  const { t } = useTranslation();
  const Icon = MILESTONE_ICON[m.status] ?? Circle;
  return (
    <li className={`fund-milestone fund-milestone--${m.status.toLowerCase()}`}>
      <Icon className="h-4 w-4 fund-milestone-icon" aria-hidden="true" />
      <div className="fund-milestone-body">
        <p className="fund-milestone-title">{m.title}</p>
        {m.description && <p className="fund-milestone-desc">{m.description}</p>}
        <p className="fund-milestone-meta">
          <span className="fund-milestone-target">
            {formatMoney(m.targetMinor, currency, locale)}
          </span>
          <span className="fund-milestone-status">
            {t(`campaigns.milestoneStatus.${m.status.toLowerCase()}`)}
          </span>
        </p>
      </div>
    </li>
  );
}

/**
 * The accounting section — Phase 4's reason for existing.
 *
 * Two columns of figures that mean different things, and the page must not
 * blur them: `received` is ledger truth CivicOS observed, `reported` is a
 * claim the organization published. Because donations settle straight to the
 * organization's own account, CivicOS cannot verify the second column at
 * all. Every label here says "reported by" for that reason.
 */
function AccountingSection({
  campaign,
  spend,
  locale,
}: {
  campaign: PublicCampaignDetail;
  spend: SpendRecord[];
  locale: string;
}) {
  const { t } = useTranslation();
  const summary = campaign.spend;
  if (!summary) return null;

  const received = campaign.raisedMinor;
  const pct = accountedPercent(summary.reportedMinor, received);
  const byMilestone = new Map(campaign.milestones.map((m) => [m.id, m.title]));

  return (
    <>
      <h2 className="fund-plan-heading">{t('campaigns.accounting.heading')}</h2>
      <p className="fund-plan-lede">{t('campaigns.accounting.lede')}</p>

      <dl className="fund-account">
        <div className="fund-account-row">
          <dt>{t('campaigns.accounting.received')}</dt>
          <dd>{formatMoneyExact(received, campaign.currency, locale)}</dd>
        </div>
        <div className="fund-account-row">
          <dt>{t('campaigns.accounting.reported')}</dt>
          <dd>{formatMoneyExact(summary.reportedMinor, campaign.currency, locale)}</dd>
        </div>
        <div className="fund-account-row fund-account-row--rest">
          <dt>{t('campaigns.accounting.unreported')}</dt>
          <dd>{formatMoneyExact(summary.unreportedMinor, campaign.currency, locale)}</dd>
        </div>
      </dl>

      <div
        className="fund-progress fund-progress--account"
        role="progressbar"
        aria-valuenow={pct}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={t('campaigns.accounting.progressLabel', { percent: pct })}
      >
        <span className="fund-progress-fill" style={{ width: `${pct}%` }} />
      </div>
      <p className="fund-account-pct">{t('campaigns.accounting.percent', { percent: pct })}</p>

      {summary.exceedsReceived && (
        <p className="fund-account-note">{t('campaigns.accounting.exceeds')}</p>
      )}

      {spend.length === 0 ? (
        <p className="fund-empty-inline">{t('campaigns.accounting.none')}</p>
      ) : (
        <ul className="fund-spend-list">
          {spend.map((r) => (
            <li key={r.id} className="fund-spend">
              <div className="fund-spend-head">
                <span className="fund-spend-amount">
                  {formatMoneyExact(r.amountMinor, r.currency, locale)}
                </span>
                <span className="fund-spend-date">
                  {new Date(r.spentAt).toLocaleDateString(locale)}
                </span>
              </div>
              <p className="fund-spend-desc">{r.description}</p>
              <p className="fund-spend-meta">
                {byMilestone.get(r.milestoneId) ?? t('campaigns.accounting.unknownMilestone')}
                {' · '}
                {t('campaigns.accounting.publishedBy', { name: r.publishedBy })}
                {r.receiptUrl && (
                  <>
                    {' · '}
                    <a href={r.receiptUrl} target="_blank" rel="noreferrer noopener">
                      {t('campaigns.accounting.receipt')}
                    </a>
                  </>
                )}
              </p>
            </li>
          ))}
        </ul>
      )}

      {/* Whose claim this is, stated plainly rather than in a tooltip. */}
      <p className="fund-account-disclaimer">{t('campaigns.accounting.disclaimer')}</p>
    </>
  );
}

/** The evidence feed: what the organization says it has been doing. */
function UpdatesSection({ updates, locale }: { updates: FundingUpdate[]; locale: string }) {
  const { t } = useTranslation();
  if (updates.length === 0) return null;

  return (
    <>
      <h2 className="fund-plan-heading">{t('campaigns.updates.heading')}</h2>
      <ol className="fund-updates">
        {updates.map((u) => (
          <li key={u.id} className="fund-update">
            <p className="fund-update-meta">
              {new Date(u.createdAt).toLocaleDateString(locale)}
              {' · '}
              {u.authorName}
            </p>
            {u.title && <h3 className="fund-update-title">{u.title}</h3>}
            <p className="fund-update-body">{u.body}</p>
            {u.attachmentUrls.length > 0 && (
              <ul className="fund-update-files">
                {u.attachmentUrls.map((url) => (
                  <li key={url}>
                    <a href={url} target="_blank" rel="noreferrer noopener">
                      {t('campaigns.updates.attachment')}
                    </a>
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ol>
    </>
  );
}

export function CampaignDetailPage() {
  const { slug } = useParams();
  const { t, i18n } = useTranslation();
  const query = usePublicCampaign(slug);
  const c = query.data;
  const donationsQuery = usePublicDonations(c?.id);
  const spendQuery = useCampaignSpend(c?.id);
  const updatesQuery = useCampaignUpdates(c?.id);
  const canManage = useCanManageCampaign(c?.organizationId);

  useSeo({
    title: c ? `${c.title} — CivicOS` : t('campaigns.detailSeoFallback'),
    description: c?.summary ?? t('campaigns.seoDescription'),
  });

  if (query.isLoading) {
    return (
      <div className="home-shell">
        <TopNav />
        <section className="home-section fund-section">
          <p className="fund-empty">{t('common.loading')}</p>
        </section>
        <Footer />
      </div>
    );
  }

  // A campaign that is not publicly visible returns 404 from the API — the
  // same response as one that never existed, so this page must not imply
  // that a hidden campaign is merely unavailable.
  if (!c) {
    return (
      <div className="home-shell">
        <TopNav />
        <section className="home-section fund-section">
          <div className="fund-empty">
            <p>{t('campaigns.notFound')}</p>
            <Link to="/campaigns" className="home-btn home-btn-ghost">
              {t('campaigns.backToAll')}
            </Link>
          </div>
        </section>
        <Footer />
      </div>
    );
  }

  const pct = progressPercent(c.raisedMinor, c.goalMinor);
  const place = [c.lga, c.state].filter(Boolean).join(', ');
  const allocated = c.milestones.reduce((sum, m) => sum + m.targetMinor, 0);
  // Mirrors the server's rule in donations.CreateIntent. A campaign that is
  // completed, reported or paused must not show a donate form.
  const acceptsDonations = c.status === 'PUBLISHED' || c.status === 'FUNDED';
  const donations = donationsQuery.data ?? [];

  return (
    <div className="home-shell">
      <TopNav />

      <section className="home-section fund-section">
        <p className="fund-back">
          <Link to="/campaigns">{t('campaigns.backToAll')}</Link>
        </p>

        <div className="fund-detail">
          <div className="fund-detail-main">
            <div className="fund-card-tags">
              {c.isEmergency && (
                <span className="fund-tag fund-tag--emergency">{t('campaigns.emergency')}</span>
              )}
              <span className="fund-tag">
                {t(
                  `campaigns.categories.${c.category
                    .toLowerCase()
                    .replace(/_([a-z])/g, (_, ch: string) => ch.toUpperCase())}`,
                )}
              </span>
            </div>

            <h1 className="fund-detail-title">{c.title}</h1>
            <p className="fund-detail-summary">{c.summary}</p>

            <div className="fund-detail-meta">
              {c.organizationName && (
                <span>
                  <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                  {t('campaigns.runBy', { org: c.organizationName })}
                </span>
              )}
              {place && (
                <span>
                  <MapPin className="h-4 w-4" aria-hidden="true" />
                  {place}
                </span>
              )}
            </div>

            {c.coverImageUrl && (
              <img className="fund-detail-img" src={c.coverImageUrl} alt="" loading="lazy" />
            )}

            <div className="fund-detail-body">
              {c.description
                .split('\n')
                .map((para, i) => (para.trim() ? <p key={i}>{para}</p> : null))}
            </div>

            <h2 className="fund-plan-heading">{t('campaigns.spendPlan')}</h2>
            <p className="fund-plan-lede">
              {t('campaigns.spendPlanLede', {
                allocated: formatMoney(allocated, c.currency, i18n.language),
                goal: formatMoney(c.goalMinor, c.currency, i18n.language),
              })}
            </p>
            <ul className="fund-milestones">
              {c.milestones.map((m) => (
                <MilestoneRow key={m.id} m={m} currency={c.currency} locale={i18n.language} />
              ))}
            </ul>

            <AccountingSection campaign={c} spend={spendQuery.data ?? []} locale={i18n.language} />

            <UpdatesSection updates={updatesQuery.data ?? []} locale={i18n.language} />

            {/* Shown only to admins of the owning organization. Rendering
                only — every write is authorised again server-side. */}
            {canManage && (
              <CampaignConsole campaign={c} spend={spendQuery.data ?? []} locale={i18n.language} />
            )}

            {donations.length > 0 && (
              <>
                <h2 className="fund-plan-heading">{t('campaigns.donorsHeading')}</h2>
                <ul className="fund-donor-list">
                  {donations.map((d, i) => (
                    <li key={i} className="fund-donor">
                      <span className="fund-donor-name">{d.donorName}</span>
                      <span className="fund-donor-amount">
                        {formatMoney(d.amountMinor, c.currency, i18n.language)}
                      </span>
                      {d.message && <p className="fund-donor-message">{d.message}</p>}
                    </li>
                  ))}
                </ul>
              </>
            )}
          </div>

          <aside className="fund-detail-aside">
            <div className="fund-raised">
              {formatMoney(c.raisedMinor, c.currency, i18n.language)}
            </div>
            <p className="fund-raised-sub">
              {t('campaigns.ofGoal', { goal: formatMoney(c.goalMinor, c.currency, i18n.language) })}
            </p>

            <div
              className="fund-progress"
              role="progressbar"
              aria-valuenow={pct}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={t('campaigns.progressLabel', { percent: pct })}
            >
              <span className="fund-progress-fill" style={{ width: `${pct}%` }} />
            </div>

            <p className="fund-donors">{t('campaigns.donorCount', { count: c.donorCount })}</p>

            {acceptsDonations ? (
              <DonateForm campaign={c} />
            ) : (
              <p className="fund-soon">{t('campaigns.notAccepting')}</p>
            )}

            <p className="fund-trust">
              <ShieldCheck className="h-4 w-4" aria-hidden="true" />
              {t('campaigns.trustNote')}
            </p>
          </aside>
        </div>
      </section>

      <Footer />
    </div>
  );
}
