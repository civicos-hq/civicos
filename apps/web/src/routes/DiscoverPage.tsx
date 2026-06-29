import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { AlertCircle, FileText, MapPin } from 'lucide-react';
import { Button } from '@civicos/ui';
import type { ApiResponse, Issue, Petition } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';

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

const PAGE_SIZE = 20;

const TIER_LABEL: Record<Tier, string> = {
  COMMUNITY: 'In your community',
  LGA: 'Near you · same LGA',
  STATE: 'Across the state',
  COUNTRY: 'Elsewhere',
};

const TIER_TONE: Record<Tier, string> = {
  COMMUNITY: 'bg-civic-100 text-civic-700',
  LGA: 'bg-emerald-100 text-emerald-700',
  STATE: 'bg-sky-100 text-sky-700',
  COUNTRY: 'bg-slate-100 text-slate-700',
};

const TIER_ORDER: Tier[] = ['COMMUNITY', 'LGA', 'STATE', 'COUNTRY'];

// Grouped (all-tiers) view — single fetch, curated.
function useGroupedFeed(communityId?: string) {
  return useQuery({
    queryKey: ['discover-feed', 'grouped', communityId ?? 'anon'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<FeedResponse>>('/api/v1/discover/feed', {
        params: communityId ? { communityId } : undefined,
      });
      return res.data.data.items;
    },
    staleTime: 30_000,
  });
}

// Single-tier flat view — paginated via offset cursor.
function useTierFeed(tier: Tier, communityId?: string) {
  return useInfiniteQuery({
    queryKey: ['discover-feed', 'tier', tier, communityId ?? 'anon'],
    queryFn: async ({ pageParam = 0 }) => {
      const res = await api.get<ApiResponse<FeedResponse>>('/api/v1/discover/feed', {
        params: {
          tier,
          limit: PAGE_SIZE,
          offset: pageParam,
          ...(communityId ? { communityId } : {}),
        },
      });
      return res.data.data;
    },
    initialPageParam: 0 as number,
    getNextPageParam: (last) => (typeof last.nextOffset === 'number' ? last.nextOffset : undefined),
    staleTime: 30_000,
  });
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diff / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

export function DiscoverPage() {
  const meQuery = useMe();
  const communityId = meQuery.data?.communityId;
  const [tier, setTier] = useState<TierFilter>('ALL');

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">Discover</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-900">What's happening near you</h1>
        <p className="mt-2 max-w-2xl text-sm text-slate-600">
          A live feed of issues and petitions, ranked by how close they are to your community.
        </p>
        {!meQuery.isLoading && !communityId && (
          <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
            You haven't joined a community yet —{' '}
            <Link to="/community" className="font-semibold underline">
              pick one
            </Link>{' '}
            and this feed will start sorting by proximity.
          </p>
        )}

        <div className="mt-4 flex flex-wrap gap-2">
          <TierPill active={tier === 'ALL'} onClick={() => setTier('ALL')} label="All" />
          {TIER_ORDER.map((t) => (
            <TierPill
              key={t}
              active={tier === t}
              onClick={() => setTier(t)}
              label={TIER_LABEL[t]}
              tone={TIER_TONE[t]}
            />
          ))}
        </div>
      </header>

      {tier === 'ALL' ? (
        <GroupedView communityId={communityId} />
      ) : (
        <TierView tier={tier} communityId={communityId} />
      )}
    </section>
  );
}

function GroupedView({ communityId }: { communityId?: string }) {
  const feedQuery = useGroupedFeed(communityId);
  const items = feedQuery.data ?? [];

  const grouped: Partial<Record<Tier, FeedItem[]>> = {};
  for (const item of items) {
    (grouped[item.tier] ||= []).push(item);
  }

  if (feedQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (items.length === 0) {
    return (
      <article className="rounded-2xl border border-dashed border-slate-300 bg-white/60 p-8 text-center text-sm text-slate-500">
        Nothing to discover yet. Once issues and petitions start appearing, they'll show up here.
      </article>
    );
  }

  return (
    <>
      {TIER_ORDER.map((tier) => {
        const tierItems = grouped[tier];
        if (!tierItems || tierItems.length === 0) return null;
        return (
          <section key={tier} className="space-y-3">
            <div className="flex items-center gap-2">
              <span
                className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${TIER_TONE[tier]}`}
              >
                {TIER_LABEL[tier]}
              </span>
              <span className="text-xs text-slate-500">
                {tierItems.length} {tierItems.length === 1 ? 'item' : 'items'}
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

function TierView({ tier, communityId }: { tier: Tier; communityId?: string }) {
  const feed = useTierFeed(tier, communityId);
  const pages = feed.data?.pages ?? [];
  const items = pages.flatMap((p) => p.items);

  if (feed.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (items.length === 0) {
    return (
      <article className="rounded-2xl border border-dashed border-slate-300 bg-white/60 p-8 text-center text-sm text-slate-500">
        No items in this tier yet.
      </article>
    );
  }

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2">
        <span
          className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${TIER_TONE[tier]}`}
        >
          {TIER_LABEL[tier]}
        </span>
        <span className="text-xs text-slate-500">
          {items.length} {items.length === 1 ? 'item' : 'items'} loaded
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
            Load more
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
      className={
        active
          ? `rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wide shadow-sm ${tone ?? 'bg-civic-700 text-white'}`
          : 'rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-semibold uppercase tracking-wide text-slate-500 hover:border-civic-300 hover:text-civic-700'
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
  return (
    <Link
      to={`/issues/${issue.id}`}
      className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
    >
      <div className="flex items-start gap-2">
        <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-rose-500" />
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-wide text-rose-600">Issue</p>
          <h3 className="mt-0.5 line-clamp-2 font-semibold text-slate-900">{issue.title}</h3>
          <p className="mt-1 line-clamp-2 text-sm text-slate-600">{issue.description}</p>
          <CardMeta community={community} createdAt={createdAt}>
            <span>{issue.upvoteCount} upvotes</span>
            <span>·</span>
            <span>{issue.status}</span>
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
  const progress = Math.min(100, Math.round((petition.signatureCount / petition.goal) * 100));
  return (
    <Link
      to={`/petitions/${petition.id}`}
      className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
    >
      <div className="flex items-start gap-2">
        <FileText className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-600" />
        <div className="min-w-0 flex-1">
          <p className="text-[10px] font-bold uppercase tracking-wide text-civic-700">Petition</p>
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
              {petition.signatureCount.toLocaleString()}/{petition.goal.toLocaleString()} signatures
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
  return (
    <div className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-500">
      {community && (
        <span className="inline-flex items-center gap-1">
          <MapPin className="h-3 w-3" />
          {community.name}
        </span>
      )}
      <span>·</span>
      <span>{timeAgo(createdAt)}</span>
      {children && <span className="ml-auto inline-flex items-center gap-1">{children}</span>}
    </div>
  );
}
