import { useEffect, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { UserRole, type ApiResponse, type Representative } from '@civicos/types';
import { api, uploadImage, uploadUrl } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useFollowedReps } from '../hooks/useFollowedReps';
import { Modal } from '../components/Modal';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { CommunityGate, CommunityGateLink } from '../components/CommunityGate';
import { Users } from 'lucide-react';

const ADMIN_ROLES = new Set<UserRole>([
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
]);

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
  const communityId = meQuery.data?.communityId;
  const repsQuery = useRepresentatives(communityId);
  const followsQuery = useFollowedReps();
  const [isModalOpen, setModalOpen] = useState(false);

  const reps = repsQuery.data ?? [];
  const followedSet = followsQuery.data ?? new Set<string>();
  const isAdmin = meQuery.data?.role ? ADMIN_ROLES.has(meQuery.data.role) : false;
  const hasCommunity = Boolean(communityId);

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('representativesPage.eyebrow')}
        title={t('representativesPage.title')}
        subtitle={t('representativesPage.subtitle')}
        meta={meta}
        actions={
          isAdmin ? (
            <Button size="sm" onClick={() => setModalOpen(true)} disabled={!hasCommunity}>
              {t('representativesPage.newBtn')}
            </Button>
          ) : undefined
        }
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
        <p className="text-sm text-slate-600">{t('common.loading')}</p>
      ) : reps.length === 0 ? (
        <EmptyState icon={<Users className="h-5 w-5" />} title={t('representativesPage.empty')} />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {reps.map((rep) => (
            <Link
              key={rep.id}
              to={`/representatives/${rep.id}`}
              className="flex gap-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
            >
              <Avatar name={rep.name} src={rep.avatarUrl} />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <h2 className="truncate text-lg font-semibold text-slate-900">
                    {rep.title} {stripTitleFromName(rep.title, rep.name)}
                  </h2>
                  <FollowButton representativeId={rep.id} isFollowing={followedSet.has(rep.id)} />
                </div>
                <p className="mt-1 text-sm text-slate-600">
                  {rep.position} · {rep.constituency}
                </p>
                {rep.party && (
                  <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700">
                    {rep.party}
                  </p>
                )}
                <div className="mt-3 flex flex-wrap gap-3 text-xs text-slate-600">
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

      {isModalOpen && hasCommunity && communityId && (
        <NewRepresentativeModal communityId={communityId} onClose={() => setModalOpen(false)} />
      )}
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
        className="flex-shrink-0 rounded-full object-cover ring-2 ring-slate-100"
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
      className="flex flex-shrink-0 items-center justify-center rounded-full bg-civic-100 font-semibold text-civic-700 ring-2 ring-slate-100"
      style={{ width: size, height: size, fontSize: Math.round(size / 2.8) }}
    >
      {initials || '?'}
    </div>
  );
}

// ─── New Representative ───────────────────────────────────────────────────────

function NewRepresentativeModal({
  communityId,
  onClose,
}: {
  communityId: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [title, setTitle] = useState('Hon.');
  const [position, setPosition] = useState('');
  const [constituency, setConstituency] = useState('');
  const [party, setParty] = useState('');
  const [bio, setBio] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [website, setWebsite] = useState('');
  const [avatar, setAvatar] = useState<File | null>(null);
  const [error, setError] = useState('');

  const preview = avatar ? URL.createObjectURL(avatar) : null;
  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [preview]);

  function onAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0] ?? null;
    e.target.value = '';
    setError('');
    if (file && file.size > 5 * 1024 * 1024) {
      setError(t('representativesPage.modal.avatarTooBig', { name: file.name }));
      return;
    }
    setAvatar(file);
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const avatarUrl = avatar ? await uploadImage(avatar) : undefined;
      await api.post('/api/v1/representatives', {
        name: stripTitleFromName(title, name),
        title,
        position,
        constituency,
        communityId,
        party: party.trim() || undefined,
        bio: bio.trim() || undefined,
        avatarUrl,
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
        website: website.trim() || undefined,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['representatives'] });
      onClose();
    },
    onError: () => setError(t('representativesPage.modal.genericError')),
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('representativesPage.modal.title')} onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="flex items-center gap-4">
          {preview ? (
            <img
              src={preview}
              alt={t('representativesPage.modal.avatarPreview')}
              className="h-16 w-16 rounded-full object-cover ring-2 ring-slate-100"
            />
          ) : (
            <div className="h-16 w-16 rounded-full bg-slate-100 ring-2 ring-slate-100" />
          )}
          <label className="cursor-pointer text-sm font-semibold text-civic-700 hover:underline">
            <input type="file" accept="image/*" className="hidden" onChange={onAvatarChange} />
            {avatar
              ? t('representativesPage.modal.changePhoto')
              : t('representativesPage.modal.uploadPhoto')}
          </label>
        </div>

        <div className="grid gap-4 md:grid-cols-[100px,1fr]">
          <Input
            label={t('representativesPage.modal.fields.title')}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t('representativesPage.modal.fields.titlePlaceholder')}
            required
          />
          <Input
            label={t('representativesPage.modal.fields.fullName')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('representativesPage.modal.fields.fullNamePlaceholder')}
            required
            minLength={2}
          />
        </div>

        <Input
          label={t('representativesPage.modal.fields.position')}
          value={position}
          onChange={(e) => setPosition(e.target.value)}
          placeholder={t('representativesPage.modal.fields.positionPlaceholder')}
          required
        />
        <div className="grid gap-4 md:grid-cols-2">
          <Input
            label={t('representativesPage.modal.fields.constituency')}
            value={constituency}
            onChange={(e) => setConstituency(e.target.value)}
            placeholder={t('representativesPage.modal.fields.constituencyPlaceholder')}
            required
          />
          <Input
            label={t('representativesPage.modal.fields.party')}
            value={party}
            onChange={(e) => setParty(e.target.value)}
            placeholder={t('representativesPage.modal.fields.partyPlaceholder')}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="bio">
            {t('representativesPage.modal.fields.bio')}
          </label>
          <textarea
            id="bio"
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            rows={3}
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            placeholder={t('representativesPage.modal.fields.bioPlaceholder')}
          />
        </div>

        <fieldset className="rounded-lg border border-slate-200 p-3">
          <legend className="px-2 text-xs font-semibold uppercase tracking-wide text-slate-600">
            {t('representativesPage.modal.contact.legend')}
          </legend>
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label={t('representativesPage.modal.contact.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('representativesPage.modal.contact.emailPlaceholder')}
            />
            <Input
              label={t('representativesPage.modal.contact.phone')}
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder={t('representativesPage.modal.contact.phonePlaceholder')}
            />
          </div>
          <div className="mt-3">
            <Input
              label={t('representativesPage.modal.contact.website')}
              type="url"
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
              placeholder={t('representativesPage.modal.contact.websitePlaceholder')}
            />
          </div>
        </fieldset>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('representativesPage.modal.submit')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
