import { useEffect, useRef } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import { PetitionStatus, type ApiResponse, type Community, type Petition } from '@civicos/types';
import { api } from '../lib/api';
import { CommentsSection } from '../components/civic/CommentsSection';
import { ImageGallery } from '../components/ImageLightbox';
import { ShareButton } from '../components/ShareButton';

const STATUS_LABEL: Record<PetitionStatus, string> = {
  [PetitionStatus.DRAFT]: 'Draft',
  [PetitionStatus.ACTIVE]: 'Active',
  [PetitionStatus.CLOSED]: 'Closed',
  [PetitionStatus.SUCCESSFUL]: 'Successful',
};

const STATUS_TONE: Record<PetitionStatus, string> = {
  [PetitionStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [PetitionStatus.ACTIVE]: 'bg-civic-100 text-civic-700',
  [PetitionStatus.CLOSED]: 'bg-amber-100 text-amber-700',
  [PetitionStatus.SUCCESSFUL]: 'bg-emerald-100 text-emerald-700',
};

function usePetition(id: string) {
  return useQuery({
    queryKey: ['petition', id],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ petition: Petition }>>(`/api/v1/petitions/${id}`);
      return res.data.data.petition;
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

function daysUntil(iso: string): number {
  return Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / (1000 * 60 * 60 * 24)));
}

export function PetitionDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const petitionQuery = usePetition(id);
  const communitiesQuery = useCommunities();
  const location = useLocation();
  const commentsRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (location.hash === '#comments' && commentsRef.current) {
      commentsRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [location.hash, petitionQuery.isLoading]);

  const signMutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/petitions/${id}/sign`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['petition', id] });
      queryClient.invalidateQueries({ queryKey: ['petitions'] });
    },
  });

  if (petitionQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }

  if (petitionQuery.isError || !petitionQuery.data) {
    return (
      <section className="space-y-4">
        <Link to="/petitions" className="text-sm font-semibold text-civic-700 hover:underline">
          ← Back to petitions
        </Link>
        <p className="text-sm text-red-600">
          Couldn't load this petition. It may have been removed.
        </p>
      </section>
    );
  }

  const petition = petitionQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === petition.communityId);
  const progress = Math.min(100, Math.round((petition.signatureCount / petition.goal) * 100));
  const createdAt = new Date(petition.createdAt).toLocaleDateString();
  const deadlineLabel = petition.deadline ? new Date(petition.deadline).toLocaleDateString() : null;
  const daysLeft = petition.deadline ? daysUntil(petition.deadline) : null;
  const canSign = petition.status === PetitionStatus.ACTIVE;

  return (
    <section className="space-y-6">
      <Link to="/petitions" className="text-sm font-semibold text-civic-700 hover:underline">
        ← Back to petitions
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Petition
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">{petition.title}</h1>
            <p className="mt-2 text-sm text-slate-500">
              Started {createdAt}
              {community ? ` · ${community.name}, ${community.lga}` : ''}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[petition.status]}`}
            >
              {STATUS_LABEL[petition.status]}
            </span>
            <ShareButton title={petition.title} />
          </div>
        </div>
      </header>

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Why this matters</h2>
        <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{petition.description}</p>
      </article>

      {petition.imageUrls && petition.imageUrls.length > 0 && (
        <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Photos</h2>
          <ImageGallery filenames={petition.imageUrls} alt="Petition photo" />
        </article>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
              Signatures
            </p>
            <p className="mt-1 text-2xl font-semibold text-slate-900">
              {petition.signatureCount.toLocaleString()}{' '}
              <span className="text-sm font-normal text-slate-500">
                of {petition.goal.toLocaleString()} target
              </span>
            </p>
          </div>
          {deadlineLabel && (
            <p className="text-sm text-slate-500">
              Deadline: <span className="font-medium text-slate-700">{deadlineLabel}</span>
              {daysLeft !== null && (
                <span
                  className={
                    daysLeft <= 3
                      ? 'ml-2 font-semibold text-rose-600'
                      : daysLeft <= 14
                        ? 'ml-2 font-semibold text-amber-600'
                        : 'ml-2 text-slate-500'
                  }
                >
                  {daysLeft === 0
                    ? '· Ends today'
                    : `· ${daysLeft} day${daysLeft === 1 ? '' : 's'} left`}
                </span>
              )}
            </p>
          )}
        </div>

        <div className="mt-4 h-2.5 rounded-full bg-slate-100">
          <div
            className="h-full rounded-full bg-gradient-to-r from-civic-700 to-civic-500"
            style={{ width: `${progress}%` }}
          />
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <span className="rounded-full bg-civic-50 px-2.5 py-1 text-sm font-semibold text-civic-700">
            {progress}% complete
          </span>
          <Button
            onClick={() => signMutation.mutate()}
            loading={signMutation.isPending}
            disabled={!canSign}
            title={canSign ? undefined : 'Only active petitions can be signed'}
          >
            ✎ Sign petition
          </Button>
        </div>
        {signMutation.isError && (
          <p className="mt-3 text-sm text-red-600">Could not sign. Please try again.</p>
        )}
      </article>

      <div id="comments" ref={commentsRef}>
        <CommentsSection entityType="petitions" entityId={petition.id} />
      </div>
    </section>
  );
}
