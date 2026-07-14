import { useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Mail, Phone, Globe, Pencil } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import { UserRole, type ApiResponse, type Community, type Representative } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useFollowedReps } from '../hooks/useFollowedReps';
import { Avatar, FollowButton, stripTitleFromName } from './RepresentativesPage';
import { CommentsSection } from '../components/civic/CommentsSection';
import { Modal } from '../components/Modal';

const ADMIN_ROLES = new Set<UserRole>([
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
]);

function useRepresentative(id: string) {
  return useQuery({
    queryKey: ['representative', id],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ representative: Representative }>>(
        `/api/v1/representatives/${id}`,
      );
      return res.data.data.representative;
    },
    enabled: Boolean(id),
  });
}

function useCommunities() {
  return useQuery({
    queryKey: ['communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });
}

export function RepresentativeDetailPage() {
  const { t, i18n } = useTranslation();
  const { id = '' } = useParams<{ id: string }>();
  const repQuery = useRepresentative(id);
  const communitiesQuery = useCommunities();
  const followsQuery = useFollowedReps();
  const meQuery = useMe();
  const [isEditOpen, setEditOpen] = useState(false);
  const isAdmin = meQuery.data?.role ? ADMIN_ROLES.has(meQuery.data.role) : false;

  if (repQuery.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>;
  }
  if (repQuery.isError || !repQuery.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/representatives"
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('representativeDetail.backToRepresentatives')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">
          {t('representativeDetail.loadError')}
        </p>
      </section>
    );
  }

  const rep = repQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === rep.communityId);
  const isFollowing = followsQuery.data?.has(rep.id) ?? false;

  return (
    <section className="space-y-6">
      <Link
        to="/representatives"
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('representativeDetail.backToRepresentatives')}
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-5">
          <Avatar name={rep.name} src={rep.avatarUrl} size={96} />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700 dark:text-civic-200">
              {t('representativeDetail.eyebrow')}
            </p>
            <h1 className="mt-1 text-3xl font-semibold text-slate-900 dark:text-slate-100">
              {rep.title} {stripTitleFromName(rep.title, rep.name)}
            </h1>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {rep.position} · {rep.constituency}
              {community ? ` · ${community.name}` : ''}
            </p>
            {rep.party && (
              <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700 dark:text-civic-200">
                {rep.party}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            {isAdmin && (
              <Button size="sm" variant="secondary" onClick={() => setEditOpen(true)}>
                <Pencil className="h-3.5 w-3.5" />
                {t('representativeDetail.edit')}
              </Button>
            )}
            <FollowButton representativeId={rep.id} isFollowing={isFollowing} />
          </div>
        </div>
      </header>

      <div className="grid gap-4 md:grid-cols-2">
        <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600 dark:text-slate-400">
            {t('representativeDetail.stats.responseRate')}
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">
            {rep.responseRate}%
          </p>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            {t('representativeDetail.stats.responseRateSub')}
          </p>
        </article>
        <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600 dark:text-slate-400">
            {t('representativeDetail.stats.followers')}
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">
            {rep.followerCount.toLocaleString(i18n.language)}
          </p>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            {t('representativeDetail.stats.followersSub')}
          </p>
        </article>
      </div>

      {rep.bio && (
        <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('representativeDetail.about')}
          </h2>
          <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
            {rep.bio}
          </p>
        </article>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {t('representativeDetail.contact.heading')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          {t('representativeDetail.contact.sub')}
        </p>
        {rep.email || rep.phone || rep.website ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {rep.email && (
              <ContactLink
                href={`mailto:${rep.email}`}
                icon={<Mail className="h-4 w-4" />}
                label={rep.email}
              />
            )}
            {rep.phone && (
              <ContactLink
                href={`tel:${rep.phone.replace(/\s+/g, '')}`}
                icon={<Phone className="h-4 w-4" />}
                label={rep.phone}
              />
            )}
            {rep.website && (
              <ContactLink
                href={rep.website}
                icon={<Globe className="h-4 w-4" />}
                label={rep.website.replace(/^https?:\/\//, '')}
                external
              />
            )}
          </div>
        ) : (
          <p className="mt-4 text-sm italic text-slate-600 dark:text-slate-400">
            {t('representativeDetail.contact.empty')}
          </p>
        )}
      </article>

      <CommentsSection entityType="representatives" entityId={rep.id} />

      {isEditOpen && <EditRepresentativeModal rep={rep} onClose={() => setEditOpen(false)} />}
    </section>
  );
}

function EditRepresentativeModal({ rep, onClose }: { rep: Representative; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState(rep.name);
  const [title, setTitle] = useState(rep.title);
  const [position, setPosition] = useState(rep.position);
  const [constituency, setConstituency] = useState(rep.constituency);
  const [party, setParty] = useState(rep.party ?? '');
  const [bio, setBio] = useState(rep.bio ?? '');
  const [email, setEmail] = useState(rep.email ?? '');
  const [phone, setPhone] = useState(rep.phone ?? '');
  const [website, setWebsite] = useState(rep.website ?? '');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: async () => {
      const payload: Record<string, string> = {};
      const cleanName = stripTitleFromName(title, name);
      if (cleanName !== rep.name) payload.name = cleanName;
      if (title !== rep.title) payload.title = title;
      if (position !== rep.position) payload.position = position;
      if (constituency !== rep.constituency) payload.constituency = constituency;
      if (party !== (rep.party ?? '')) payload.party = party;
      if (bio !== (rep.bio ?? '')) payload.bio = bio;
      if (email !== (rep.email ?? '')) payload.email = email;
      if (phone !== (rep.phone ?? '')) payload.phone = phone;
      if (website !== (rep.website ?? '')) payload.website = website;
      if (Object.keys(payload).length === 0) return rep;
      const res = await api.patch<ApiResponse<{ representative: Representative }>>(
        `/api/v1/representatives/${rep.id}`,
        payload,
      );
      return res.data.data.representative;
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(['representative', rep.id], updated);
      queryClient.invalidateQueries({ queryKey: ['representatives'] });
      onClose();
    },
    onError: () => setError(t('representativeDetail.editModal.genericError')),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('representativeDetail.editModal.title', { name: rep.name })} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid gap-3 md:grid-cols-[100px,1fr]">
          <Input
            label={t('representativeDetail.editModal.fields.title')}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
          <Input
            label={t('representativeDetail.editModal.fields.fullName')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
          />
        </div>
        <Input
          label={t('representativeDetail.editModal.fields.position')}
          value={position}
          onChange={(e) => setPosition(e.target.value)}
        />
        <div className="grid gap-3 md:grid-cols-2">
          <Input
            label={t('representativeDetail.editModal.fields.constituency')}
            value={constituency}
            onChange={(e) => setConstituency(e.target.value)}
          />
          <Input
            label={t('representativeDetail.editModal.fields.party')}
            value={party}
            onChange={(e) => setParty(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label
            className="text-sm font-medium text-gray-700 dark:text-gray-300"
            htmlFor="edit-bio"
          >
            {t('representativeDetail.editModal.fields.bio')}
          </label>
          <textarea
            id="edit-bio"
            rows={3}
            className="w-full rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            value={bio}
            onChange={(e) => setBio(e.target.value)}
          />
        </div>

        <fieldset className="rounded-lg border border-slate-200 p-3">
          <legend className="px-2 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-400">
            {t('representativeDetail.editModal.contactLegend')}
          </legend>
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label={t('representativeDetail.editModal.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Input
              label={t('representativeDetail.editModal.phone')}
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
          </div>
          <div className="mt-3">
            <Input
              label={t('representativeDetail.editModal.website')}
              type="url"
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
            />
          </div>
        </fieldset>

        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('representativeDetail.editModal.save')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function ContactLink({
  href,
  icon,
  label,
  external,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  external?: boolean;
}) {
  return (
    <a
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
      className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 dark:bg-slate-800/40 px-3 py-1.5 text-sm font-medium text-slate-700 dark:text-slate-300 hover:border-civic-300 dark:hover:border-civic-500 hover:bg-civic-50 dark:hover:bg-civic-500/10 hover:text-civic-700 dark:hover:text-civic-200"
    >
      <span className="text-slate-400">{icon}</span>
      {label}
    </a>
  );
}
