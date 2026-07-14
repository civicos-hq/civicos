import { useState } from 'react';
import axios from 'axios';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { type Community } from '@civicos/types';
import { api } from '../lib/api';
import { useCommunities } from '../hooks/useCommunities';
import { useMe } from '../hooks/useMe';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { Home, Crown } from 'lucide-react';

// Communities are civic geography — the canonical list is managed by
// platform operators from the admin console, not from the citizen web.
// No in-app create affordance here for any role.

export function CommunityPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const meQuery = useMe();
  const communitiesQuery = useCommunities();
  const queryClient = useQueryClient();

  const joinMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.post('/api/v1/auth/me/community', { communityId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries();
    },
  });

  const switchMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.patch('/api/v1/auth/me/active-community', { communityId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries();
    },
  });

  const [primaryError, setPrimaryError] = useState<string | null>(null);

  const primaryMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.patch('/api/v1/auth/me/primary-community', { communityId });
    },
    onSuccess: () => {
      setPrimaryError(null);
      queryClient.invalidateQueries();
    },
    onError: (err) => {
      if (axios.isAxiosError(err) && err.response?.status === 429) {
        const data = err.response.data as {
          code?: string;
          nextEligibleAt?: string;
        };
        if (data.code === 'PRIMARY_COMMUNITY_COOLDOWN' && data.nextEligibleAt) {
          const when = new Date(data.nextEligibleAt).toLocaleDateString();
          setPrimaryError(t('communityPage.primary.cooldownError', { date: when }));
          return;
        }
      }
      setPrimaryError(t('communityPage.primary.genericError'));
    },
  });

  const me = meQuery.data;
  const communities = communitiesQuery.data ?? [];
  const memberships = me?.memberships ?? [];
  const joinedCommunityIDs = new Set(memberships.map((membership) => membership.communityId));
  const activeCommunity = communities.find((c) => c.id === me?.activeCommunityId);
  const joinedCommunities = communities.filter((c) => joinedCommunityIDs.has(c.id));
  const availableCommunities = communities.filter((c) => !joinedCommunityIDs.has(c.id));
  const primaryCommunityId = me?.primaryCommunityId;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('communityPage.eyebrow')}
        title={activeCommunity ? activeCommunity.name : t('communityPage.findYours')}
        subtitle={
          activeCommunity
            ? t('communityPage.memberSub', {
                name: activeCommunity.name,
                lga: activeCommunity.lga,
                state: activeCommunity.state,
              })
            : t('communityPage.joinPrompt')
        }
        meta={meta}
      />

      {meQuery.isLoading || communitiesQuery.isLoading ? (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      ) : activeCommunity ? (
        <>
          <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
              {t('communityPage.activeCommunity')}
            </h2>
            <div className="mt-3 grid gap-4 md:grid-cols-3">
              <Stat label={t('communityPage.stats.state')} value={activeCommunity.state} />
              <Stat label={t('communityPage.stats.lga')} value={activeCommunity.lga} />
              <Stat label={t('communityPage.stats.country')} value={activeCommunity.country} />
            </div>
            {activeCommunity.description && (
              <p className="mt-4 text-sm text-slate-600 dark:text-slate-300">
                {activeCommunity.description}
              </p>
            )}
          </article>

          <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
              {t('communityPage.joinedCommunities')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              {t('communityPage.joinedSub')}
            </p>
            <p className="mt-2 text-xs text-slate-500 dark:text-slate-300">
              {t('communityPage.primary.explainer')}
            </p>
            {primaryError && (
              <p className="mt-2 text-sm text-red-600 dark:text-red-400">{primaryError}</p>
            )}
            <div className="mt-4 grid gap-3">
              {joinedCommunities.map((c) => (
                <JoinedCommunityRow
                  key={c.id}
                  community={c}
                  isActive={c.id === me?.activeCommunityId}
                  isPrimary={c.id === primaryCommunityId}
                  switchLabel={
                    c.id === me?.activeCommunityId
                      ? t('communityPage.actions.active')
                      : t('communityPage.actions.switch')
                  }
                  primaryLabel={
                    c.id === primaryCommunityId
                      ? t('communityPage.actions.primary')
                      : t('communityPage.actions.makePrimary')
                  }
                  primaryBadge={t('communityPage.primary.badge')}
                  switching={switchMutation.isPending && switchMutation.variables === c.id}
                  changingPrimary={primaryMutation.isPending && primaryMutation.variables === c.id}
                  onSwitch={() => switchMutation.mutate(c.id)}
                  onMakePrimary={() => {
                    setPrimaryError(null);
                    primaryMutation.mutate(c.id);
                  }}
                />
              ))}
            </div>
          </section>

          <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
              {t('communityPage.availableCommunities')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              {t('communityPage.availableSub')}
            </p>
            {availableCommunities.length === 0 ? (
              <div className="mt-4">
                <EmptyState
                  icon={<Home className="h-5 w-5" />}
                  title={t('communityPage.emptyJoinedAll')}
                />
              </div>
            ) : (
              <div className="mt-4 grid gap-3">
                {availableCommunities.map((c) => (
                  <AvailableCommunityRow
                    key={c.id}
                    community={c}
                    actionLabel={t('communityPage.actions.join')}
                    loading={joinMutation.isPending && joinMutation.variables === c.id}
                    onAction={() => joinMutation.mutate(c.id)}
                  />
                ))}
              </div>
            )}
          </section>
        </>
      ) : (
        <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('communityPage.availableCommunities')}
          </h2>
          {communities.length === 0 ? (
            <div className="mt-4">
              <EmptyState icon={<Home className="h-5 w-5" />} title={t('communityPage.empty')} />
            </div>
          ) : (
            <div className="mt-4 grid gap-3">
              {availableCommunities.map((c) => (
                <AvailableCommunityRow
                  key={c.id}
                  community={c}
                  actionLabel={t('communityPage.actions.join')}
                  loading={joinMutation.isPending && joinMutation.variables === c.id}
                  onAction={() => joinMutation.mutate(c.id)}
                />
              ))}
            </div>
          )}
          {(joinMutation.isError || switchMutation.isError) && (
            <p className="mt-3 text-sm text-red-600 dark:text-red-400">
              {t('communityPage.joinError')}
            </p>
          )}
        </section>
      )}
    </section>
  );
}

function AvailableCommunityRow({
  community,
  actionLabel,
  loading,
  onAction,
}: {
  community: Community;
  actionLabel: string;
  loading: boolean;
  onAction: () => void;
}) {
  return (
    <article className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 dark:bg-slate-800/40 p-4">
      <div>
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">{community.name}</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          {community.lga}, {community.state}
        </p>
      </div>
      <Button size="sm" onClick={onAction} loading={loading}>
        {actionLabel}
      </Button>
    </article>
  );
}

function JoinedCommunityRow({
  community,
  isActive,
  isPrimary,
  switchLabel,
  primaryLabel,
  primaryBadge,
  switching,
  changingPrimary,
  onSwitch,
  onMakePrimary,
}: {
  community: Community;
  isActive: boolean;
  isPrimary: boolean;
  switchLabel: string;
  primaryLabel: string;
  primaryBadge: string;
  switching: boolean;
  changingPrimary: boolean;
  onSwitch: () => void;
  onMakePrimary: () => void;
}) {
  return (
    <article className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 dark:bg-slate-800/40 p-4">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="font-semibold text-slate-900 dark:text-slate-100">{community.name}</h3>
          {isPrimary && (
            <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 dark:bg-amber-500/15 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-amber-800 dark:text-amber-200">
              <Crown className="h-3 w-3" aria-hidden="true" />
              {primaryBadge}
            </span>
          )}
        </div>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          {community.lga}, {community.state}
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          variant="secondary"
          onClick={onMakePrimary}
          loading={changingPrimary}
          disabled={isPrimary}
        >
          {primaryLabel}
        </Button>
        <Button size="sm" onClick={onSwitch} loading={switching} disabled={isActive}>
          {switchLabel}
        </Button>
      </div>
    </article>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50/70 dark:bg-slate-800/40 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600 dark:text-slate-300">
        {label}
      </p>
      <p className="mt-2 text-base font-semibold text-slate-900 dark:text-slate-100">{value}</p>
    </div>
  );
}
