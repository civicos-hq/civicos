import { useEffect, useRef } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import {
  IssueStatus,
  UserRole,
  type ApiResponse,
  type Community,
  type Issue,
} from '@civicos/types';
import { api } from '../lib/api';
import { CommentsSection } from '../components/civic/CommentsSection';
import { ImageGallery } from '../components/ImageLightbox';
import { ShareButton } from '../components/ShareButton';
import { useMe } from '../hooks/useMe';
import { useUpvotedIssues } from '../hooks/useUpvotedIssues';
import { useEnumLabels } from '../hooks/useEnumLabels';

const STAFF_ROLES = new Set<UserRole>([
  UserRole.REPRESENTATIVE,
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
  UserRole.MODERATOR,
]);

const STATUS_FLOW: IssueStatus[] = [
  IssueStatus.OPEN,
  IssueStatus.UNDER_REVIEW,
  IssueStatus.IN_PROGRESS,
  IssueStatus.RESOLVED,
];

const STATUS_TONE: Record<IssueStatus, string> = {
  [IssueStatus.OPEN]: 'bg-rose-100 text-rose-700',
  [IssueStatus.UNDER_REVIEW]: 'bg-amber-100 text-amber-700',
  [IssueStatus.IN_PROGRESS]: 'bg-sky-100 text-sky-700',
  [IssueStatus.RESOLVED]: 'bg-emerald-100 text-emerald-700',
  [IssueStatus.CLOSED]: 'bg-slate-200 text-slate-700',
};

function useIssue(id: string) {
  return useQuery({
    queryKey: ['issue', id],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ issue: Issue }>>(`/api/v1/issues/${id}`);
      return res.data.data.issue;
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

export function IssueDetailPage() {
  const { t, i18n } = useTranslation();
  const enums = useEnumLabels();
  const { id = '' } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const issueQuery = useIssue(id);
  const communitiesQuery = useCommunities();
  const meQuery = useMe();
  const location = useLocation();
  const commentsRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (location.hash === '#comments' && commentsRef.current) {
      commentsRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [location.hash, issueQuery.isLoading]);

  const upvotedIssuesQuery = useUpvotedIssues();
  const hasUpvoted = Boolean(id && upvotedIssuesQuery.data?.has(id));

  const upvoteMutation = useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiResponse<{ upvoted: boolean; upvoteCount: number }>>(
        `/api/v1/issues/${id}/upvote`,
      );
      return res.data.data;
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ['upvotedIssues'] });
      const prev = queryClient.getQueryData<Set<string>>(['upvotedIssues']);
      if (prev && id) {
        const next = new Set(prev);
        next.has(id) ? next.delete(id) : next.add(id);
        queryClient.setQueryData(['upvotedIssues'], next);
      }
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['upvotedIssues'], ctx.prev);
    },
    onSuccess: (data) => {
      if (id) {
        queryClient.setQueryData<Issue>(['issue', id], (prev) =>
          prev ? { ...prev, upvoteCount: data.upvoteCount } : prev,
        );
        queryClient.setQueryData<Set<string>>(['upvotedIssues'], (prev) => {
          const next = new Set(prev ?? []);
          data.upvoted ? next.add(id) : next.delete(id);
          return next;
        });
      }
      queryClient.invalidateQueries({ queryKey: ['issues'] });
    },
  });

  const statusMutation = useMutation({
    mutationFn: async (status: IssueStatus) => {
      await api.patch(`/api/v1/issues/${id}/status`, { status });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['issue', id] });
      queryClient.invalidateQueries({ queryKey: ['issues'] });
    },
  });

  if (issueQuery.isLoading) {
    return <p className="text-sm text-slate-500">{t('common.loading')}</p>;
  }

  if (issueQuery.isError || !issueQuery.data) {
    return (
      <section className="space-y-4">
        <Link to="/issues" className="text-sm font-semibold text-civic-700 hover:underline">
          {t('issueDetail.backToIssues')}
        </Link>
        <p className="text-sm text-red-600">{t('issueDetail.loadError')}</p>
      </section>
    );
  }

  const issue = issueQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === issue.communityId);
  const reportedAt = new Date(issue.createdAt).toLocaleString(i18n.language);
  const isStaff = meQuery.data?.role ? STAFF_ROLES.has(meQuery.data.role) : false;

  return (
    <section className="space-y-6">
      <Link to="/issues" className="text-sm font-semibold text-civic-700 hover:underline">
        {t('issueDetail.backToIssues')}
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              {enums.issueCategory(issue.category)}
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">{issue.title}</h1>
            <p className="mt-2 text-sm text-slate-500">
              {t('issueDetail.reportedAt', { when: reportedAt })}
              {community ? ` · ${community.name}, ${community.lga}` : ''}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[issue.status]}`}
            >
              {enums.issueStatus(issue.status)}
            </span>
            <ShareButton title={issue.title} />
          </div>
        </div>

        <StatusTimeline current={issue.status} />

        {isStaff && (
          <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-slate-100 pt-4">
            <p className="mr-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
              {t('issueDetail.updateStatus')}
            </p>
            {(Object.values(IssueStatus) as IssueStatus[]).map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => statusMutation.mutate(s)}
                disabled={statusMutation.isPending || s === issue.status}
                className={
                  s === issue.status
                    ? `rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[s]} opacity-60`
                    : 'rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-medium text-slate-600 hover:border-civic-300 hover:text-civic-700 disabled:opacity-50'
                }
              >
                {enums.issueStatus(s)}
              </button>
            ))}
            {statusMutation.isError && (
              <span className="text-xs text-red-600">{t('issueDetail.updateStatusError')}</span>
            )}
          </div>
        )}
      </header>

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">{t('issueDetail.description')}</h2>
        <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{issue.description}</p>
        {issue.location && (
          <div className="mt-4 border-t border-slate-100 pt-4">
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
              {t('issueDetail.location')}
            </p>
            <p className="mt-1 text-sm text-slate-700">{issue.location}</p>
          </div>
        )}
      </article>

      {issue.imageUrls && issue.imageUrls.length > 0 && (
        <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">{t('issueDetail.photos')}</h2>
          <ImageGallery filenames={issue.imageUrls} alt={t('issueDetail.photoAlt')} />
        </article>
      )}

      <article className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            {t('issueDetail.communitySupport')}
          </p>
          <p className="mt-1 text-2xl font-semibold text-slate-900">
            {issue.upvoteCount}{' '}
            <span className="text-sm font-normal text-slate-500">
              {t('issueDetail.upvotesCount', { count: issue.upvoteCount })}
            </span>
          </p>
        </div>
        <Button
          onClick={() => upvoteMutation.mutate()}
          loading={upvoteMutation.isPending}
          variant={hasUpvoted ? 'secondary' : 'primary'}
        >
          {hasUpvoted ? t('issueDetail.upvoted') : t('issueDetail.upvote')}
        </Button>
      </article>

      <div id="comments" ref={commentsRef}>
        <CommentsSection entityType="issues" entityId={issue.id} />
      </div>
    </section>
  );
}

function StatusTimeline({ current }: { current: IssueStatus }) {
  const { t } = useTranslation();
  const enums = useEnumLabels();
  if (current === IssueStatus.CLOSED) {
    return (
      <p className="mt-5 text-xs italic text-slate-500">{t('issueDetail.statusClosedNote')}</p>
    );
  }
  const activeIdx = STATUS_FLOW.indexOf(current);
  return (
    <ol className="mt-5 flex flex-wrap items-center gap-3 text-xs">
      {STATUS_FLOW.map((s, i) => {
        const reached = i <= activeIdx;
        const isActive = i === activeIdx;
        return (
          <li key={s} className="flex items-center gap-2">
            <span
              className={
                isActive
                  ? 'flex h-6 w-6 items-center justify-center rounded-full bg-civic-700 text-[10px] font-bold text-white ring-4 ring-civic-100'
                  : reached
                    ? 'flex h-6 w-6 items-center justify-center rounded-full bg-civic-600 text-[10px] font-bold text-white'
                    : 'flex h-6 w-6 items-center justify-center rounded-full border border-slate-200 bg-white text-[10px] font-bold text-slate-400'
              }
            >
              {i + 1}
            </span>
            <span className={reached ? 'font-semibold text-slate-900' : 'text-slate-400'}>
              {enums.issueStatus(s)}
            </span>
            {i < STATUS_FLOW.length - 1 && (
              <span
                className={reached ? 'h-px w-6 bg-civic-600' : 'h-px w-6 bg-slate-200'}
                aria-hidden="true"
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}
