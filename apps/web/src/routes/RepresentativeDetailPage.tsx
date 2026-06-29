import { useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Mail, Phone, Globe, Pencil } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import { UserRole, type ApiResponse, type Community, type Representative } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useFollowedReps } from '../hooks/useFollowedReps';
import { Avatar, FollowButton } from './RepresentativesPage';
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
  const { id = '' } = useParams<{ id: string }>();
  const repQuery = useRepresentative(id);
  const communitiesQuery = useCommunities();
  const followsQuery = useFollowedReps();
  const meQuery = useMe();
  const [isEditOpen, setEditOpen] = useState(false);
  const isAdmin = meQuery.data?.role ? ADMIN_ROLES.has(meQuery.data.role) : false;

  if (repQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (repQuery.isError || !repQuery.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/representatives"
          className="text-sm font-semibold text-civic-700 hover:underline"
        >
          ← Back to representatives
        </Link>
        <p className="text-sm text-red-600">Couldn't load this representative.</p>
      </section>
    );
  }

  const rep = repQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === rep.communityId);
  const isFollowing = followsQuery.data?.has(rep.id) ?? false;

  return (
    <section className="space-y-6">
      <Link to="/representatives" className="text-sm font-semibold text-civic-700 hover:underline">
        ← Back to representatives
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-5">
          <Avatar name={rep.name} src={rep.avatarUrl} size={96} />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Representative
            </p>
            <h1 className="mt-1 text-3xl font-semibold text-slate-900">
              {rep.title} {rep.name}
            </h1>
            <p className="mt-1 text-sm text-slate-600">
              {rep.position} · {rep.constituency}
              {community ? ` · ${community.name}` : ''}
            </p>
            {rep.party && (
              <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700">
                {rep.party}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            {isAdmin && (
              <Button size="sm" variant="secondary" onClick={() => setEditOpen(true)}>
                <Pencil className="h-3.5 w-3.5" />
                Edit
              </Button>
            )}
            <FollowButton representativeId={rep.id} isFollowing={isFollowing} />
          </div>
        </div>
      </header>

      <div className="grid gap-4 md:grid-cols-2">
        <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Response rate
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">{rep.responseRate}%</p>
          <p className="mt-1 text-sm text-slate-500">Of citizen messages answered.</p>
        </article>
        <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Followers
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">
            {rep.followerCount.toLocaleString()}
          </p>
          <p className="mt-1 text-sm text-slate-500">Citizens subscribed to updates.</p>
        </article>
      </div>

      {rep.bio && (
        <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">About</h2>
          <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{rep.bio}</p>
        </article>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Contact</h2>
        <p className="mt-1 text-sm text-slate-500">
          For private matters. For public questions, post below — replies are visible to everyone.
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
          <p className="mt-4 text-sm italic text-slate-500">
            No direct contact details on file. Post a question below to reach this representative
            publicly.
          </p>
        )}
      </article>

      <CommentsSection entityType="representatives" entityId={rep.id} />

      {isEditOpen && <EditRepresentativeModal rep={rep} onClose={() => setEditOpen(false)} />}
    </section>
  );
}

function EditRepresentativeModal({ rep, onClose }: { rep: Representative; onClose: () => void }) {
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
      if (name !== rep.name) payload.name = name;
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
    onError: () => setError('Could not save changes. Double-check email/website format.'),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={`Edit ${rep.name}`} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid gap-3 md:grid-cols-[100px,1fr]">
          <Input label="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
          <Input
            label="Full name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
          />
        </div>
        <Input label="Position" value={position} onChange={(e) => setPosition(e.target.value)} />
        <div className="grid gap-3 md:grid-cols-2">
          <Input
            label="Constituency"
            value={constituency}
            onChange={(e) => setConstituency(e.target.value)}
          />
          <Input label="Party" value={party} onChange={(e) => setParty(e.target.value)} />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="edit-bio">
            Bio
          </label>
          <textarea
            id="edit-bio"
            rows={3}
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            value={bio}
            onChange={(e) => setBio(e.target.value)}
          />
        </div>

        <fieldset className="rounded-lg border border-slate-200 p-3">
          <legend className="px-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
            Contact
          </legend>
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <Input
              label="Phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
          </div>
          <div className="mt-3">
            <Input
              label="Website"
              type="url"
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
            />
          </div>
        </fieldset>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Save
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
      className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-700 hover:border-civic-300 hover:bg-civic-50 hover:text-civic-700"
    >
      <span className="text-slate-400">{icon}</span>
      {label}
    </a>
  );
}
