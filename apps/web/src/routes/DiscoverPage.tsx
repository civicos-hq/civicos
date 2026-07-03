import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { AlertCircle, FileText, MapPin } from 'lucide-react';
import { Button } from '@civicos/ui';
import type { ApiResponse, Issue, Petition } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useEnumLabels } from '../hooks/useEnumLabels';
import { useRelativeTime } from '../hooks/useRelativeTime';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { CommunityGate, CommunityGateLink } from '../components/CommunityGate';

type Tier = 'COMMUNITY' | 'LGA' | 'STATE' | 'COUNTRY';

interface CommunitySummary {
  id: string;
  name: string;
  state: string;
  lga: string;
}

interface FeedItem {
  kind: 'issue' | 'petition';
  tier: Tier;
  createdAt: string;
  communityId: string;
  community?: CommunitySummary;
  issue?: Issue;
  petition?: Petition;
}

interface FeedResponse {
  items: FeedItem[];
  nextOffset?: number | null;
}

type TierFilter = Tier | 'ALL';
type KindFilter = 'all' | 'issue' | 'petition';

const PAGE_SIZE = 20;

const KIND_KEYS: KindFilter[] = ['all', 'issue', 'petition'];

const TIER_TONE: Record<Tier, string> = {
  COMMUNITY: 'bg-civic-100 text-civic-700',
  LGA: 'bg-emerald-100 text-emerald-700',
  STATE: 'bg-sky-100 text-sky-700',
  COUNTRY: 'bg-slate-100 text-slate-700',
};

const TIER_ORDER: Tier[] = ['COMMUNITY', 'LGA', 'STATE', 'COUNTRY'];

// Grouped (all-tiers) view — single fetch, curated.
function useGroupedFeed(communityId: string | undefined, kind: KindFilter) {
  return useQuery({
    queryKey: ['discover-feed', 'grouped', communityId ?? 'anon', kind],
    queryFn: async () => {
      const res = await api.get<ApiResponse<FeedResponse>>('/api/v1/discover/feed', {
        params: {
          ...(communityId ? { communityId } : {}),
          ...(kind !== 'all' ? { kind } : {}),
        },
      });
      return res.data.data.items;
    },
    staleTime: 30_000,
  });
}

// Single-tier flat view — paginated via offset cursor.
function useTierFeed(tier: Tier, communityId: string | undefined, kind: KindFilter) {
  return useInfiniteQuery({
    queryKey: ['discover-feed', 'tier', tier, communityId ?? 'anon', kind],
    queryFn: async ({ pageParam = 0 }) => {
      const res = await api.get<ApiResponse<FeedResponse>>('/api/v1/discover/feed', {
        params: {
          tier,
          limit: PAGE_SIZE,
          offset: pageParam,
          ...(communityId ? { communityId } : {}),
          ...(kind !== 'all' ? { kind } : {}),
        },
      });
      return res.data.data;
    },
    initialPageParam: 0 as number,
    getNextPageParam: (last) => (typeof last.nextOffset === 'number' ? last.nextOffset : undefined),
    staleTime: 30_000,
  });
}

export function DiscoverPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const meQuery = useMe();
  const communityId = meQuery.data?.communityId;
  const [tier, setTier] = useState<TierFilter>('ALL');
  const [kind, setKind] = useState<KindFilter>('all');

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('discoverPage.eyebrow')}
        title={t('discoverPage.title')}
        subtitle={t('discoverPage.subtitle')}
        meta={meta}
      >
        {!meQuery.isLoading && !communityId && (
          <CommunityGate>
            {t('discoverPage.noCommunityBefore')}{' '}
            <CommunityGateLink>{t('discoverPage.noCommunityLink')}</CommunityGateLink>{' '}
            {t('discoverPage.noCommunityAfter')}
          </CommunityGate>
        )}

        <div className="mt-4 flex flex-wrap gap-2">
          <TierPill
            active={tier === 'ALL'}
            onClick={() => setTier('ALL')}
            label={t('discoverPage.tiers.ALL')}
          />
          {TIER_ORDER.map((tk) => (
            <TierPill
              key={tk}
              active={tier === tk}
              onClick={() => setTier(tk)}
              label={t(`discoverPage.tiers.${tk}`)}
              tone={TIER_TONE[tk]}
            />
          ))}
        </div>
        <div className="mt-2 flex flex-wrap gap-2">
          {KIND_KEYS.map((k) => (
            <TierPill
              key={k}
              active={kind === k}
              onClick={() => setKind(k)}
              label={t(`discoverPage.kinds.${k}`)}
            />
          ))}
        </div>
      </PageHeader>

      {tier === 'ALL' ? (
        <GroupedView communityId={communityId} kind={kind} />
      ) : (
        <TierView tier={tier} communityId={communityId} kind={kind} />
      )}
    </section>
  );
}

function GroupedView({ communityId, kind }: { communityId?: string; kind: KindFilter }) {
  const { t } = useTranslation();
  const feedQuery = useGroupedFeed(communityId, kind);
  const items = feedQuery.data ?? [];

  const grouped: Partial<Record<Tier, FeedItem[]>> = {};
  for (const item of items) {
    (grouped[item.tier] ||= []).push(item);
  }

  if (feedQuery.isLoading) {
    return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  }
  if (items.length === 0) {
    return <EmptyState title={t('discoverPage.empty.grouped')} />;
  }

  return (
    <>
      {TIER_ORDER.map((tk) => {
        const tierItems = grouped[tk];
        if (!tierItems || tierItems.length === 0) return null;
        return (
          <section key={tk} className="space-y-3">
            <div className="flex items-center gap-2">
              <span
                className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${TIER_TONE[tk]}`}
              >
                {t(`discoverPage.tiers.${tk}`)}
              </span>
              <span className="text-xs text-slate-600">
                {t('discoverPage.itemCount', { count: tierItems.length })}
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              {tierItems.map((item) => (
                <FeedCard key={`${item.kind}-${item.issue?.id ?? item.petition?.id}`} item={item} />
              ))}
            </div>
          </section>
        );
      })}
    </>
  );
}

function TierView({
  tier,
  communityId,
  kind,
}: {
  tier: Tier;
  communityId?: string;
  kind: KindFilter;
}) {
  const { t } = useTranslation();
  const feed = useTierFeed(tier, communityId, kind);
  const pages = feed.data?.pages ?? [];
  const items = pages.flatMap((p) => p.items);

  if (feed.isLoading) {
    return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  }
  if (items.length === 0) {
    return <EmptyState title={t('discoverPage.empty.tier')} />;
  }

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2">
        <span
          className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${TIER_TONE[tier]}`}
        >
          {t(`discoverPage.tiers.${tier}`)}
        </span>
        <span className="text-xs text-slate-600">
          {t('discoverPage.itemsLoaded', { count: items.length })}
        </span>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        {items.map((item) => (
          <FeedCard key={`${item.kind}-${item.issue?.id ?? item.petition?.id}`} item={item} />
        ))}
      </div>

      {feed.hasNextPage && (
        <div className="flex justify-center pt-2">
          <Button
            variant="secondary"
            size="sm"
            loading={feed.isFetchingNextPage}
            onClick={() => feed.fetchNextPage()}
          >
            {t('discoverPage.loadMore')}
          </Button>
        </div>
      )}
    </section>
  );
}

function TierPill({
  active,
  onClick,
  label,
  tone,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
  tone?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={
        active
          ? `rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wide shadow-sm ${tone ?? 'bg-civic-700 text-white'}`
          : 'rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-semibold uppercase tracking-wide text-slate-600 hover:border-civic-300 hover:text-civic-700'
      }
    >
      {label}
    </button>
  );
}

function FeedCard({ item }: { item: FeedItem }) {
  if (item.kind === 'issue' && item.issue) {
    return <IssueCard issue={item.issue} community={item.community} createdAt={item.createdAt} />;
  }
  if (item.kind === 'petition' && item.petition) {
    return (
      <PetitionCard
        petition={item.petition}
        community={item.community}
        createdAt={item.createdAt}
      />
    );
  }
  return null;
}

function IssueCard({
  issue,
  community,
  createdAt,
}: {
  issue: Issue;
  community?: CommunitySummary;
  createdAt: string;
}) {
  const { t } = useTranslation();
  const enums = useEnumLabels();
  return (
    <Link
      to={`/issues/${issue.id}`}
      className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
    >
      <div className="flex items-start gap-2">
        <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-rose-500" />
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-wide text-rose-600">
            {t('discoverPage.labels.issue')}
          </p>
          <h3 className="mt-0.5 line-clamp-2 font-semibold text-slate-900">{issue.title}</h3>
          <p className="mt-1 line-clamp-2 text-sm text-slate-600">{issue.description}</p>
          <CardMeta community={community} createdAt={createdAt}>
            <span>{t('discoverPage.meta.upvotes', { count: issue.upvoteCount })}</span>
            <span>·</span>
            <span>{enums.issueStatus(issue.status)}</span>
          </CardMeta>
        </div>
      </div>
    </Link>
  );
}

function PetitionCard({
  petition,
  community,
  createdAt,
}: {
  petition: Petition;
  community?: CommunitySummary;
  createdAt: string;
}) {
  const { t } = useTranslation();
  const progress = Math.min(100, Math.round((petition.signatureCount / petition.goal) * 100));
  return (
    <Link
      to={`/petitions/${petition.id}`}
      className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
    >
      <div className="flex items-start gap-2">
        <FileText className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-600" />
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-wide text-civic-700">
            {t('discoverPage.labels.petition')}
          </p>
          <h3 className="mt-0.5 line-clamp-2 font-semibold text-slate-900">{petition.title}</h3>
          <p className="mt-1 line-clamp-2 text-sm text-slate-600">{petition.description}</p>
          <div className="mt-3 h-1.5 rounded-full bg-slate-100">
            <div
              className="h-full rounded-full bg-gradient-to-r from-civic-700 to-civic-500"
              style={{ width: `${progress}%` }}
            />
          </div>
          <CardMeta community={community} createdAt={createdAt}>
            <span>
              {t('discoverPage.meta.signaturesOf', {
                signatures: petition.signatureCount.toLocaleString(),
                goal: petition.goal.toLocaleString(),
              })}
            </span>
            <span>·</span>
            <span>{progress}%</span>
          </CardMeta>
        </div>
      </div>
    </Link>
  );
}

function CardMeta({
  community,
  createdAt,
  children,
}: {
  community?: CommunitySummary;
  createdAt: string;
  children?: React.ReactNode;
}) {
  const relative = useRelativeTime();
  return (
    <div className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-600">
      {community && (
        <span className="inline-flex items-center gap-1">
          <MapPin className="h-3 w-3" />
          {community.name}
        </span>
      )}
      <span>·</span>
      <span>{relative(createdAt)}</span>
      {children && <span className="ml-auto inline-flex items-center gap-1">{children}</span>}
    </div>
  );
}
