import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import type { ApiResponse, Community } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';

function useCommunities() {
  return useQuery({
    queryKey: ['communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });
}

export function CommunityPage() {
  const meQuery = useMe();
  const communitiesQuery = useCommunities();
  const queryClient = useQueryClient();

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

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white/90 p-6 shadow-sm backdrop-blur">
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
      </header>

      {meQuery.isLoading || communitiesQuery.isLoading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : myCommunity ? (
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
                <article
                  key={c.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4"
                >
                  <div>
                    <h3 className="font-semibold text-slate-900">{c.name}</h3>
                    <p className="mt-1 text-sm text-slate-600">
                      {c.lga}, {c.state}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    onClick={() => joinMutation.mutate(c.id)}
                    loading={joinMutation.isPending && joinMutation.variables === c.id}
                  >
                    Join
                  </Button>
                </article>
              ))}
            </div>
          )}
          {joinMutation.isError && (
            <p className="mt-3 text-sm text-red-600">Could not join. Please try again.</p>
          )}
        </section>
      )}
    </section>
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
