import { useState, type FormEvent } from 'react';
import axios from 'axios';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { UserRole, type Community } from '@civicos/types';
import { api } from '../lib/api';
import { useCommunities } from '../hooks/useCommunities';
import { useMe } from '../hooks/useMe';
import { Modal } from '../components/Modal';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { Home, Crown } from 'lucide-react';

const ADMIN_ROLES = new Set<UserRole>([
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
]);

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function CommunityPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const meQuery = useMe();
  const communitiesQuery = useCommunities();
  const queryClient = useQueryClient();
  const [isNewOpen, setNewOpen] = useState(false);

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
  const isAdmin = me?.role ? ADMIN_ROLES.has(me.role) : false;
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
        actions={
          isAdmin ? (
            <Button size="sm" onClick={() => setNewOpen(true)}>
              {t('communityPage.newCommunity')}
            </Button>
          ) : undefined
        }
      />

      {meQuery.isLoading || communitiesQuery.isLoading ? (
        <p className="text-sm text-slate-600">{t('common.loading')}</p>
      ) : activeCommunity ? (
        <>
          <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">
              {t('communityPage.activeCommunity')}
            </h2>
            <div className="mt-3 grid gap-4 md:grid-cols-3">
              <Stat label={t('communityPage.stats.state')} value={activeCommunity.state} />
              <Stat label={t('communityPage.stats.lga')} value={activeCommunity.lga} />
              <Stat label={t('communityPage.stats.country')} value={activeCommunity.country} />
            </div>
            {activeCommunity.description && (
              <p className="mt-4 text-sm text-slate-600">{activeCommunity.description}</p>
            )}
          </article>

          <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">
              {t('communityPage.joinedCommunities')}
            </h2>
            <p className="mt-1 text-sm text-slate-600">{t('communityPage.joinedSub')}</p>
            <p className="mt-2 text-xs text-slate-500">{t('communityPage.primary.explainer')}</p>
            {primaryError && <p className="mt-2 text-sm text-red-600">{primaryError}</p>}
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

          <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">
              {t('communityPage.availableCommunities')}
            </h2>
            <p className="mt-1 text-sm text-slate-600">{t('communityPage.availableSub')}</p>
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
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">
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
            <p className="mt-3 text-sm text-red-600">{t('communityPage.joinError')}</p>
          )}
        </section>
      )}

      {isNewOpen && (
        <NewCommunityModal
          existingSlugs={communities.map((c) => c.slug)}
          onClose={() => setNewOpen(false)}
        />
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
    <article className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
      <div>
        <h3 className="font-semibold text-slate-900">{community.name}</h3>
        <p className="mt-1 text-sm text-slate-600">
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
    <article className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="font-semibold text-slate-900">{community.name}</h3>
          {isPrimary && (
            <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-amber-800">
              <Crown className="h-3 w-3" aria-hidden="true" />
              {primaryBadge}
            </span>
          )}
        </div>
        <p className="mt-1 text-sm text-slate-600">
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
    <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600">{label}</p>
      <p className="mt-2 text-base font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function NewCommunityModal({
  existingSlugs,
  onClose,
}: {
  existingSlugs: string[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [slugTouched, setSlugTouched] = useState(false);
  const [state, setState] = useState('');
  const [lga, setLga] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  const effectiveSlug = slugTouched ? slug : slugify(name);

  const mutation = useMutation({
    mutationFn: async () => {
      await api.post('/api/v1/communities', {
        name: name.trim(),
        slug: effectiveSlug,
        state: state.trim(),
        lga: lga.trim(),
        description: description.trim() || undefined,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['communities'] });
      onClose();
    },
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    if (existingSlugs.includes(effectiveSlug)) {
      setError(t('communityPage.modal.slugTaken', { slug: effectiveSlug }));
      return;
    }
    mutation.mutate();
  }

  return (
    <Modal title={t('communityPage.modal.title')} onClose={onClose}>
      <form className="grid gap-3" onSubmit={submit}>
        <label className="text-sm text-slate-700">
          {t('communityPage.modal.name')}
          <Input
            className="mt-1.5"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('communityPage.modal.namePlaceholder')}
            required
            minLength={2}
          />
        </label>

        <label className="text-sm text-slate-700">
          {t('communityPage.modal.slug')}
          <Input
            className="mt-1.5"
            value={effectiveSlug}
            onChange={(e) => {
              setSlugTouched(true);
              setSlug(slugify(e.target.value));
            }}
            placeholder={t('communityPage.modal.slugPlaceholder')}
            required
            minLength={2}
            pattern="[a-z0-9-]+"
          />
          <span className="mt-1 block text-xs text-slate-600">
            {t('communityPage.modal.slugHint')}
          </span>
        </label>

        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm text-slate-700">
            {t('communityPage.modal.state')}
            <Input
              className="mt-1.5"
              value={state}
              onChange={(e) => setState(e.target.value)}
              placeholder={t('communityPage.modal.statePlaceholder')}
              required
            />
          </label>
          <label className="text-sm text-slate-700">
            {t('communityPage.modal.lga')}
            <Input
              className="mt-1.5"
              value={lga}
              onChange={(e) => setLga(e.target.value)}
              placeholder={t('communityPage.modal.lgaPlaceholder')}
              required
            />
          </label>
        </div>

        <label className="text-sm text-slate-700">
          {t('communityPage.modal.description')}{' '}
          <span className="text-slate-400">{t('communityPage.modal.descriptionOptional')}</span>
          <textarea
            className="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('communityPage.modal.descriptionPlaceholder')}
          />
        </label>

        {(error || mutation.isError) && (
          <p className="text-sm text-red-600">{error || t('communityPage.modal.genericError')}</p>
        )}

        <div className="mt-2 flex items-center justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" size="sm" loading={mutation.isPending}>
            {t('communityPage.modal.create')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
