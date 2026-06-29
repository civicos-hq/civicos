import { useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Input } from '@civicos/ui';
import { UserRole, type ApiResponse, type Community } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { Modal } from '../components/Modal';

const ADMIN_ROLES = new Set<UserRole>([
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
]);

function useCommunities() {
  return useQuery({
    queryKey: ['communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function CommunityPage() {
  const meQuery = useMe();
  const communitiesQuery = useCommunities();
  const queryClient = useQueryClient();
  const [isNewOpen, setNewOpen] = useState(false);

  const joinMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.post('/api/v1/auth/me/community', { communityId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me'] });
    },
  });

  const me = meQuery.data;
  const communities = communitiesQuery.data ?? [];
  const myCommunity = communities.find((c) => c.id === me?.communityId);
  const isAdmin = me?.role ? ADMIN_ROLES.has(me.role) : false;

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white/90 p-6 shadow-sm backdrop-blur">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.16em] text-civic-700">
              Community
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">
              {myCommunity ? myCommunity.name : 'Find your community'}
            </h1>
            <p className="mt-3 max-w-2xl text-sm text-slate-600">
              {myCommunity
                ? `You're a member of ${myCommunity.name} (${myCommunity.lga}, ${myCommunity.state}). Reports and petitions you create will belong to this community.`
                : 'Join a community to start reporting issues and signing petitions.'}
            </p>
          </div>
          {isAdmin && (
            <Button size="sm" onClick={() => setNewOpen(true)}>
              + New community
            </Button>
          )}
        </div>
      </header>

      {meQuery.isLoading || communitiesQuery.isLoading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : myCommunity ? (
        <>
          <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">Your community</h2>
            <div className="mt-3 grid gap-4 md:grid-cols-3">
              <Stat label="State" value={myCommunity.state} />
              <Stat label="LGA" value={myCommunity.lga} />
              <Stat label="Country" value={myCommunity.country} />
            </div>
            {myCommunity.description && (
              <p className="mt-4 text-sm text-slate-600">{myCommunity.description}</p>
            )}
          </article>

          <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">Switch community</h2>
            <p className="mt-1 text-sm text-slate-500">
              You can move to another community at any time.
            </p>
            <div className="mt-4 grid gap-3">
              {communities
                .filter((c) => c.id !== myCommunity.id)
                .map((c) => (
                  <CommunityRow
                    key={c.id}
                    community={c}
                    actionLabel="Switch"
                    loading={joinMutation.isPending && joinMutation.variables === c.id}
                    onAction={() => joinMutation.mutate(c.id)}
                  />
                ))}
            </div>
          </section>
        </>
      ) : (
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Available communities</h2>
          {communities.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500">
              No communities are available yet. Check back soon.
            </p>
          ) : (
            <div className="mt-4 grid gap-3">
              {communities.map((c) => (
                <CommunityRow
                  key={c.id}
                  community={c}
                  actionLabel="Join"
                  loading={joinMutation.isPending && joinMutation.variables === c.id}
                  onAction={() => joinMutation.mutate(c.id)}
                />
              ))}
            </div>
          )}
          {joinMutation.isError && (
            <p className="mt-3 text-sm text-red-600">Could not join. Please try again.</p>
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

function CommunityRow({
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

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">{label}</p>
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
      setError(`Slug "${effectiveSlug}" is already in use. Pick another.`);
      return;
    }
    mutation.mutate();
  }

  return (
    <Modal title="New community" onClose={onClose}>
      <form className="grid gap-3" onSubmit={submit}>
        <label className="text-sm text-slate-700">
          Name
          <Input
            className="mt-1.5"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Ikeja Central"
            required
            minLength={2}
          />
        </label>

        <label className="text-sm text-slate-700">
          Slug
          <Input
            className="mt-1.5"
            value={effectiveSlug}
            onChange={(e) => {
              setSlugTouched(true);
              setSlug(slugify(e.target.value));
            }}
            placeholder="ikeja-central"
            required
            minLength={2}
            pattern="[a-z0-9-]+"
          />
          <span className="mt-1 block text-xs text-slate-500">
            Lowercase letters, numbers, and hyphens only.
          </span>
        </label>

        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm text-slate-700">
            State
            <Input
              className="mt-1.5"
              value={state}
              onChange={(e) => setState(e.target.value)}
              placeholder="Lagos"
              required
            />
          </label>
          <label className="text-sm text-slate-700">
            LGA
            <Input
              className="mt-1.5"
              value={lga}
              onChange={(e) => setLga(e.target.value)}
              placeholder="Ikeja"
              required
            />
          </label>
        </div>

        <label className="text-sm text-slate-700">
          Description <span className="text-slate-400">(optional)</span>
          <textarea
            className="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What this community is about"
          />
        </label>

        {(error || mutation.isError) && (
          <p className="text-sm text-red-600">
            {error || 'Could not create community. Please try again.'}
          </p>
        )}

        <div className="mt-2 flex items-center justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="sm" loading={mutation.isPending}>
            Create community
          </Button>
        </div>
      </form>
    </Modal>
  );
}
