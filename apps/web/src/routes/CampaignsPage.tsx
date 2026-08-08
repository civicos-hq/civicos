import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { HandCoins } from 'lucide-react';
import { TopNav, Footer } from './HomePage';
import { PageHeader } from '../components/PageHeader';
import { useSeo } from '../hooks/useSeo';
import {
  usePublicCampaigns,
  type CampaignCategory,
  type CampaignSort,
} from '../hooks/useCampaigns';
import { CampaignCard } from '../components/CampaignCard';
import { useOptionalMe } from '../hooks/useMe';
import { useCommunity } from '../hooks/useCommunities';

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

/**
 * The body of the campaigns browse page, free of chrome. The route
 * layer in `App.tsx` decides which chrome to wrap this with — see
 * `CampaignRoute` and `CampaignsPage`.
 *
 * The `variant` prop picks the header treatment:
 *
 *   - `dashboard` — shared `PageHeader` (left-aligned, eyebrow + title,
 *     masthead dateline). Reads as a normal list page inside the
 *     dashboard chrome, not a marketing hero.
 *
 *   - `marketing` — the centered hero with a Fraunces headline. Kept
 *     because the standalone `/campaigns` route is the surface a
 *     citizen following a shared link from Twitter/WhatsApp lands on,
 *     and that page earns its keep by reading like a public record.
 */
export function CampaignsContent({ variant }: { variant: 'dashboard' | 'marketing' }) {
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
  const { data: home } = useCommunity(me?.activeCommunityId);

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

  // Two layouts:
  //   - Inside the dashboard (`DashboardLayout` parent): use the shared
  //     PageHeader so the page reads as a normal dashboard list — left-
  //     aligned title, eyebrow, no marketing hero.
  //   - Standalone on the public marketing page (`home-shell` parent):
  //     keep the centered hero, since that's the surface a citizen
  //     following a shared link lands on.
  if (variant === 'dashboard') {
    return (
      <section className="space-y-6">
        <PageHeader
          eyebrow={t('campaigns.eyebrow')}
          title={t('campaigns.title')}
          subtitle={t('campaigns.lede')}
        />
        <CampaignFilters
          category={category}
          emergency={emergency}
          verified={verified}
          sort={sort}
          home={home}
          onCategory={setCategory}
          onEmergency={toggleEmergency}
          onVerified={toggleVerified}
          onSort={setSort}
        />
        <CampaignGrid items={items} isLoading={query.isLoading} />
      </section>
    );
  }

  return (
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
  );
}

/**
 * The filter + sort row, used in both layouts. Same DOM in both so the
 * public and dashboard views share the same UX.
 */
function CampaignFilters({
  category,
  emergency,
  verified,
  sort,
  home,
  onCategory,
  onEmergency,
  onVerified,
  onSort,
}: {
  category: '' | CampaignCategory;
  emergency: boolean;
  verified: boolean;
  sort: CampaignSort;
  home: { state?: string; lga?: string } | null | undefined;
  onCategory: (next: '' | CampaignCategory) => void;
  onEmergency: () => void;
  onVerified: () => void;
  onSort: (next: CampaignSort) => void;
}) {
  const { t } = useTranslation();
  return (
    <>
      <div className="fund-filters" role="group" aria-label={t('campaigns.filterLabel')}>
        {CATEGORIES.map((c) => (
          <button
            key={c.key}
            type="button"
            className={`fund-filter${category === c.value ? ' is-active' : ''}`}
            aria-pressed={category === c.value}
            onClick={() => onCategory(c.value)}
          >
            {t(`campaigns.categories.${c.key}`)}
          </button>
        ))}
        <button
          type="button"
          className={`fund-filter${emergency ? ' is-active' : ''}`}
          aria-pressed={emergency}
          onClick={onEmergency}
        >
          {t('campaigns.emergencyOnly')}
        </button>
        <button
          type="button"
          className={`fund-filter${verified ? ' is-active' : ''}`}
          aria-pressed={verified}
          onClick={onVerified}
        >
          {t('campaigns.verifiedOnly')}
        </button>
      </div>
      <div className="fund-sorts" role="group" aria-label={t('campaigns.sortLabel')}>
        <span className="fund-sorts-label">{t('campaigns.sortLabel')}</span>
        {SORTS.filter((s) => s !== 'NEAR_ME' || home).map((s) => (
          <button
            key={s}
            type="button"
            className={`fund-filter${sort === s ? ' is-active' : ''}`}
            aria-pressed={sort === s}
            onClick={() => onSort(s)}
          >
            {t(`campaigns.sorts.${s}`)}
          </button>
        ))}
      </div>
    </>
  );
}

function CampaignGrid({
  items,
  isLoading,
}: {
  items: ReturnType<typeof usePublicCampaigns>['data'];
  isLoading: boolean;
}) {
  const { t } = useTranslation();
  if (isLoading) return <p className="fund-empty">{t('common.loading')}</p>;
  if (!items || items.length === 0) {
    return (
      <div className="fund-empty">
        <HandCoins className="h-6 w-6" aria-hidden="true" />
        <p>{t('campaigns.empty')}</p>
      </div>
    );
  }
  return (
    <div className="fund-grid">
      {items.map((c) => (
        <CampaignCard key={c.id} c={c} />
      ))}
    </div>
  );
}

/**
 * Public-facing wrapper used by the `/campaigns` route mounted outside
 * `RequireAuth`. Adds the marketing `TopNav` + `Footer` so a citizen
 * following a shared campaign link lands on a complete page even
 * without an account.
 *
 * Signed-in users reach the same campaigns content through the
 * dashboard route, which renders `<CampaignsContent variant="dashboard" />`
 * directly under `DashboardLayout`. They never hit this wrapper — see
 * `App.tsx` `CampaignRoute`.
 */
export function CampaignsPage() {
  return (
    <div className="home-shell">
      <TopNav />
      <CampaignsContent variant="marketing" />
      <Footer />
    </div>
  );
}
