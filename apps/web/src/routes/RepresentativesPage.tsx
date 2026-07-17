import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { type ApiResponse, type Representative } from '@civicos/types';
import { api, uploadUrl } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useFollowedReps } from '../hooks/useFollowedReps';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { CommunityGate, CommunityGateLink } from '../components/CommunityGate';
import { Users } from 'lucide-react';

// Strips a leading title from a name so `${title} ${name}` never doubles up.
// Handles "Mr. Mr Kola", "Mr Kola" against title "Mr." or "Mr". Used both at
// display time (fixes existing bad records) and at save time (prevents new
// ones).
export function stripTitleFromName(title: string | undefined | null, name: string): string {
  const cleanName = name.trim();
  const cleanTitle = (title ?? '').trim();
  if (!cleanTitle) return cleanName;
  const base = cleanTitle.replace(/\.$/, '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return cleanName.replace(new RegExp(`^${base}\\.?\\s+`, 'i'), '').trim();
}

function useRepresentatives(communityId?: string) {
  return useQuery({
    queryKey: ['representatives', communityId ?? 'all'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ representatives: Representative[] }>>(
        '/api/v1/representatives',
        { params: communityId ? { communityId } : undefined },
      );
      return res.data.data.representatives;
    },
  });
}

export function RepresentativesPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const meQuery = useMe();
  const communityId = meQuery.data?.activeCommunityId;
  const repsQuery = useRepresentatives(communityId);
  const followsQuery = useFollowedReps();

  const reps = repsQuery.data ?? [];
  const followedSet = followsQuery.data ?? new Set<string>();
  const hasCommunity = Boolean(communityId);

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('representativesPage.eyebrow')}
        title={t('representativesPage.title')}
        subtitle={t('representativesPage.subtitle')}
        meta={meta}
      >
        {!meQuery.isLoading && !hasCommunity && (
          <CommunityGate>
            {t('representativesPage.noCommunityBefore')}{' '}
            <CommunityGateLink>{t('representativesPage.noCommunityLink')}</CommunityGateLink>{' '}
            {t('representativesPage.noCommunityAfter')}
          </CommunityGate>
        )}
      </PageHeader>

      {repsQuery.isLoading ? (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      ) : reps.length === 0 ? (
        <EmptyState
          icon={<Users className="h-5 w-5" />}
          illustration="/designs/08_representatives_engage.png?v=6"
          title={t('representativesPage.empty')}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {reps.map((rep) => (
            <Link
              key={rep.id}
              to={`/representatives/${rep.id}`}
              className="flex gap-4 rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-5 shadow-sm transition hover:border-civic-300 dark:hover:border-civic-500"
            >
              <Avatar name={rep.name} src={rep.avatarUrl} />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <h2 className="truncate text-lg font-semibold text-slate-900 dark:text-slate-100">
                    {rep.title} {stripTitleFromName(rep.title, rep.name)}
                  </h2>
                  <FollowButton representativeId={rep.id} isFollowing={followedSet.has(rep.id)} />
                </div>
                <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                  {rep.position} · {rep.constituency}
                </p>
                {rep.party && (
                  <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700 dark:text-civic-200">
                    {rep.party}
                  </p>
                )}
                <div className="mt-3 flex flex-wrap gap-3 text-xs text-slate-600 dark:text-slate-300">
                  <span>
                    {t('representativesPage.card.responseRate', { rate: rep.responseRate })}
                  </span>
                  <span>·</span>
                  <span>
                    {t('representativesPage.card.followerCount', { count: rep.followerCount })}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Rep profiles are only minted by approving a
          RepresentativeApplication in identity-service — the signup
          form's REPRESENTATIVE track is the entry point. */}
      <p className="text-xs text-slate-500 dark:text-slate-300">
        {t('representativesPage.applyPrompt')}{' '}
        <Link
          to="/register"
          className="font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('representativesPage.applyCta')}
        </Link>
      </p>
    </section>
  );
}

// ─── Follow button ────────────────────────────────────────────────────────────

export function FollowButton({
  representativeId,
  isFollowing,
}: {
  representativeId: string;
  isFollowing: boolean;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const toggle = useMutation({
    mutationFn: async () => {
      if (isFollowing) {
        await api.delete(`/api/v1/representatives/${representativeId}/follow`);
      } else {
        await api.post(`/api/v1/representatives/${representativeId}/follow`);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['followedReps'] });
      queryClient.invalidateQueries({ queryKey: ['representatives'] });
      queryClient.invalidateQueries({ queryKey: ['representative', representativeId] });
    },
  });

  return (
    <Button
      variant={isFollowing ? 'secondary' : 'primary'}
      size="sm"
      loading={toggle.isPending}
      aria-pressed={isFollowing}
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        toggle.mutate();
      }}
    >
      {isFollowing
        ? t('representativesPage.follow.following')
        : t('representativesPage.follow.follow')}
    </Button>
  );
}

// ─── Avatar ───────────────────────────────────────────────────────────────────

export function Avatar({ name, src, size = 64 }: { name: string; src?: string; size?: number }) {
  if (src) {
    return (
      <img
        src={uploadUrl(src)}
        alt={name}
        className="flex-shrink-0 rounded-full object-cover ring-2 ring-slate-100 dark:ring-slate-800"
        style={{ width: size, height: size }}
      />
    );
  }
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('');
  return (
    <div
      className="flex flex-shrink-0 items-center justify-center rounded-full bg-civic-100 dark:bg-civic-500/15 font-semibold text-civic-700 dark:text-civic-200 ring-2 ring-slate-100 dark:ring-slate-800"
      style={{ width: size, height: size, fontSize: Math.round(size / 2.8) }}
    >
      {initials || '?'}
    </div>
  );
}
