import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Circle, Clock, MapPin, ShieldCheck } from 'lucide-react';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';
import {
  formatMoney,
  progressPercent,
  usePublicCampaign,
  type PublicMilestone,
} from '../hooks/useCampaigns';

// Public campaign page. Phase 2 shows the ask and the spend plan; Phase 4
// extends this same page with the funds-flow dashboard (received /
// withdrawn / remaining, receipts, reports).
//
// There is deliberately no donate button yet — no payment rail exists, and
// a dead CTA on a fundraising page is worse than none.

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

export function CampaignDetailPage() {
  const { slug } = useParams();
  const { t, i18n } = useTranslation();
  const query = usePublicCampaign(slug);
  const c = query.data;

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

            {/* Honest placeholder rather than a dead button. Donations land
                in Phase 3; until a payment rail exists, saying so is more
                trustworthy than a CTA that goes nowhere. */}
            <p className="fund-soon">{t('campaigns.donationsSoon')}</p>

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
