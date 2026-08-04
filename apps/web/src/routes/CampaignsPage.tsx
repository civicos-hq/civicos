import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { HandCoins, MapPin, ShieldCheck } from 'lucide-react';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';
import {
  categoryKey,
  formatMoney,
  progressPercent,
  usePublicCampaigns,
  type CampaignCategory,
  type CampaignSort,
  type PublicCampaign,
} from '../hooks/useCampaigns';
import { useOptionalMe } from '../hooks/useMe';
import { useCommunities } from '../hooks/useCommunities';

// Public, unauthenticated. Uses the marketing chrome (TopNav/Footer) rather
// than the dashboard layout, because a citizen following a shared campaign
// link has no session and must not be bounced to /login.

// The browse sorts from the funding spec. NEAR_ME is conditional — see below.
const SORTS: CampaignSort[] = ['RECENT', 'ENDING_SOON', 'MOST_FUNDED', 'EMERGENCY', 'NEAR_ME'];

const CATEGORIES: Array<{ value: '' | CampaignCategory; key: string }> = [
  { value: '', key: 'all' },
  { value: 'EMERGENCY_RELIEF', key: 'emergencyRelief' },
  { value: 'COMMUNITY_DEVELOPMENT', key: 'communityDevelopment' },
  { value: 'EDUCATION', key: 'education' },
  { value: 'HEALTHCARE', key: 'healthcare' },
  { value: 'ENVIRONMENT', key: 'environment' },
  { value: 'AGRICULTURE', key: 'agriculture' },
];

function CampaignCard({ c }: { c: PublicCampaign }) {
  const { t, i18n } = useTranslation();
  const pct = progressPercent(c.raisedMinor, c.goalMinor);
  const place = [c.lga, c.state].filter(Boolean).join(', ');

  return (
    <Link to={`/campaigns/${c.slug}`} className="fund-card">
      {c.coverImageUrl && (
        <img className="fund-card-img" src={c.coverImageUrl} alt="" loading="lazy" />
      )}
      <div className="fund-card-body">
        <div className="fund-card-tags">
          {c.isEmergency && (
            <span className="fund-tag fund-tag--emergency">{t('campaigns.emergency')}</span>
          )}
          <span className="fund-tag">{t(`campaigns.categories.${categoryKey(c.category)}`)}</span>
        </div>

        <h3 className="fund-card-title">{c.title}</h3>
        <p className="fund-card-summary">{c.summary}</p>

        {c.organizationName && (
          <p className="fund-card-org">
            <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
            {c.organizationName}
          </p>
        )}
        {place && (
          <p className="fund-card-place">
            <MapPin className="h-3.5 w-3.5" aria-hidden="true" />
            {place}
          </p>
        )}

        {/* Progress is shown even at zero — a campaign that has raised
            nothing yet is exactly what a citizen should be able to see. */}
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
        <p className="fund-card-money">
          <strong>{formatMoney(c.raisedMinor, c.currency, i18n.language)}</strong>{' '}
          {t('campaigns.ofGoal', {
            goal: formatMoney(c.goalMinor, c.currency, i18n.language),
          })}
        </p>
      </div>
    </Link>
  );
}

export function CampaignsPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const category = (params.get('category') as CampaignCategory | null) ?? '';
  const emergency = params.get('emergency') === 'true';
  const verified = params.get('verified') === 'true';
  const sort = (params.get('sort') as CampaignSort | null) ?? 'RECENT';

  // "Near me" needs somewhere to measure from. The membership record carries
  // only a community id, so the place comes from the community list — already
  // cached for other pages, and no new endpoint.
  const { data: me } = useOptionalMe();
  const { data: communities } = useCommunities();
  const home = communities?.find((c) => c.id === me?.activeCommunityId);

  useSeo({
    title: t('campaigns.seoTitle'),
    description: t('campaigns.seoDescription'),
  });

  const query = usePublicCampaigns({
    category,
    emergency,
    verified,
    sort,
    nearState: home?.state,
    nearLga: home?.lga,
  });
  const items = query.data ?? [];

  function setCategory(next: '' | CampaignCategory) {
    const p = new URLSearchParams(params);
    if (next) p.set('category', next);
    else p.delete('category');
    setParams(p, { replace: true });
  }

  function toggleEmergency() {
    const p = new URLSearchParams(params);
    if (emergency) p.delete('emergency');
    else p.set('emergency', 'true');
    setParams(p, { replace: true });
  }

  function toggleVerified() {
    const p = new URLSearchParams(params);
    if (verified) p.delete('verified');
    else p.set('verified', 'true');
    setParams(p, { replace: true });
  }

  function setSort(next: CampaignSort) {
    const p = new URLSearchParams(params);
    if (next === 'RECENT') p.delete('sort');
    else p.set('sort', next);
    setParams(p, { replace: true });
  }

  return (
    <div className="home-shell">
      <TopNav />

      <section className="home-section fund-section">
        <div className="home-section-head">
          <h1 className="home-section-title">{t('campaigns.title')}</h1>
          <p className="fund-lede">{t('campaigns.lede')}</p>
        </div>

        <div className="fund-filters" role="group" aria-label={t('campaigns.filterLabel')}>
          {CATEGORIES.map((c) => (
            <button
              key={c.key}
              type="button"
              className={`fund-filter${category === c.value ? ' is-active' : ''}`}
              aria-pressed={category === c.value}
              onClick={() => setCategory(c.value)}
            >
              {t(`campaigns.categories.${c.key}`)}
            </button>
          ))}
          <button
            type="button"
            className={`fund-filter${emergency ? ' is-active' : ''}`}
            aria-pressed={emergency}
            onClick={toggleEmergency}
          >
            {t('campaigns.emergencyOnly')}
          </button>
          <button
            type="button"
            className={`fund-filter${verified ? ' is-active' : ''}`}
            aria-pressed={verified}
            onClick={toggleVerified}
          >
            {t('campaigns.verifiedOnly')}
          </button>
        </div>

        {/* NEAR_ME is offered only when there is a location to measure from.
            Showing it to a signed-out reader would be a control that silently
            does nothing, which is worse than not offering it. */}
        <div className="fund-sorts" role="group" aria-label={t('campaigns.sortLabel')}>
          <span className="fund-sorts-label">{t('campaigns.sortLabel')}</span>
          {SORTS.filter((s) => s !== 'NEAR_ME' || home).map((s) => (
            <button
              key={s}
              type="button"
              className={`fund-filter${sort === s ? ' is-active' : ''}`}
              aria-pressed={sort === s}
              onClick={() => setSort(s)}
            >
              {t(`campaigns.sorts.${s}`)}
            </button>
          ))}
        </div>

        {query.isLoading ? (
          <p className="fund-empty">{t('common.loading')}</p>
        ) : items.length === 0 ? (
          <div className="fund-empty">
            <HandCoins className="h-6 w-6" aria-hidden="true" />
            <p>{t('campaigns.empty')}</p>
          </div>
        ) : (
          <div className="fund-grid">
            {items.map((c) => (
              <CampaignCard key={c.id} c={c} />
            ))}
          </div>
        )}
      </section>

      <Footer />
    </div>
  );
}
