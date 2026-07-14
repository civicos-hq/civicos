import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { PetitionStatus, type ApiResponse, type Community, type Petition } from '@civicos/types';
import { api, getApiError } from '../lib/api';
import { CommentsSection } from '../components/civic/CommentsSection';
import { ImageGallery } from '../components/ImageLightbox';
import { ShareButton } from '../components/ShareButton';
import { useEnumLabels } from '../hooks/useEnumLabels';

const STATUS_TONE: Record<PetitionStatus, string> = {
  [PetitionStatus.DRAFT]: 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300',
  [PetitionStatus.ACTIVE]: 'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  [PetitionStatus.CLOSED]: 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
  [PetitionStatus.SUCCESSFUL]:
    'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
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
  const { t, i18n } = useTranslation();
  const enums = useEnumLabels();
  const { id = '' } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const petitionQuery = usePetition(id);
  const communitiesQuery = useCommunities();
  const location = useLocation();
  const commentsRef = useRef<HTMLDivElement | null>(null);
  const [signError, setSignError] = useState('');

  useEffect(() => {
    if (location.hash === '#comments' && commentsRef.current) {
      commentsRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [location.hash, petitionQuery.isLoading]);

  const signMutation = useMutation({
    mutationFn: async () => {
      setSignError('');
      await api.post(`/api/v1/petitions/${id}/sign`);
    },
    onSuccess: () => {
      setSignError('');
      queryClient.invalidateQueries({ queryKey: ['petition', id] });
      queryClient.invalidateQueries({ queryKey: ['petitions'] });
    },
    onError: (err) => {
      const apiError = getApiError(err);
      setSignError(
        apiError?.code === 'EMAIL_NOT_VERIFIED'
          ? t('auth.verify.actionRequired')
          : apiError?.code === 'COMMUNITY_MEMBERSHIP_REQUIRED'
            ? t('petitionDetail.signMembershipRequired')
            : t('petitionDetail.signError'),
      );
    },
  });

  if (petitionQuery.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }

  if (petitionQuery.isError || !petitionQuery.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/petitions"
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('petitionDetail.backToPetitions')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">{t('petitionDetail.loadError')}</p>
      </section>
    );
  }

  const petition = petitionQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === petition.communityId);
  const progress = Math.min(100, Math.round((petition.signatureCount / petition.goal) * 100));
  const createdAt = new Date(petition.createdAt).toLocaleDateString(i18n.language);
  const deadlineLabel = petition.deadline
    ? new Date(petition.deadline).toLocaleDateString(i18n.language)
    : null;
  const daysLeft = petition.deadline ? daysUntil(petition.deadline) : null;
  const canSign = petition.status === PetitionStatus.ACTIVE;

  return (
    <section className="space-y-6">
      <Link
        to="/petitions"
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('petitionDetail.backToPetitions')}
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700 dark:text-civic-200">
              {t('petitionDetail.eyebrow')}
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">
              {petition.title}
            </h1>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
              {t('petitionDetail.startedAt', { when: createdAt })}
              {community ? ` · ${community.name}, ${community.lga}` : ''}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[petition.status]}`}
            >
              {enums.petitionStatus(petition.status)}
            </span>
            <ShareButton title={petition.title} />
          </div>
        </div>
      </header>

      <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {t('petitionDetail.whyThisMatters')}
        </h2>
        <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
          {petition.description}
        </p>
      </article>

      {petition.imageUrls && petition.imageUrls.length > 0 && (
        <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('petitionDetail.photos')}
          </h2>
          <ImageGallery filenames={petition.imageUrls} alt={t('petitionDetail.photoAlt')} />
        </article>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-6 shadow-sm">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600 dark:text-slate-300">
              {t('petitionDetail.signatures')}
            </p>
            <p className="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">
              {petition.signatureCount.toLocaleString(i18n.language)}{' '}
              <span className="text-sm font-normal text-slate-600 dark:text-slate-300">
                {t('petitionDetail.ofTarget', {
                  goal: petition.goal.toLocaleString(i18n.language),
                })}
              </span>
            </p>
          </div>
          {deadlineLabel && (
            <p className="text-sm text-slate-600 dark:text-slate-300">
              {t('petitionDetail.deadline')}{' '}
              <span className="font-medium text-slate-700 dark:text-slate-300">
                {deadlineLabel}
              </span>
              {daysLeft !== null && (
                <span
                  className={
                    daysLeft <= 3
                      ? 'ml-2 font-semibold text-rose-600 dark:text-rose-400'
                      : daysLeft <= 14
                        ? 'ml-2 font-semibold text-amber-600 dark:text-amber-400'
                        : 'ml-2 text-slate-600 dark:text-slate-300'
                  }
                >
                  {daysLeft === 0
                    ? t('petitionDetail.endsToday')
                    : t('petitionDetail.daysLeft', { count: daysLeft })}
                </span>
              )}
            </p>
          )}
        </div>

        <div className="mt-4 h-2.5 rounded-full bg-slate-100 dark:bg-slate-800/60">
          <div
            className="h-full rounded-full bg-gradient-to-r from-civic-700 to-civic-500"
            style={{ width: `${progress}%` }}
          />
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <span className="rounded-full bg-civic-50 px-2.5 py-1 text-sm font-semibold text-civic-700 dark:text-civic-200">
            {t('petitionDetail.percentComplete', { percent: progress })}
          </span>
          <Button
            onClick={() => signMutation.mutate()}
            loading={signMutation.isPending}
            disabled={!canSign}
            title={canSign ? undefined : t('petitionDetail.onlyActive')}
          >
            {t('petitionDetail.sign')}
          </Button>
        </div>
        {signError && <p className="mt-3 text-sm text-red-600 dark:text-red-400">{signError}</p>}
      </article>

      <div id="comments" ref={commentsRef}>
        <CommentsSection entityType="petitions" entityId={petition.id} />
      </div>
    </section>
  );
}
