import { useEffect, useRef } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import {
  IssueCategory,
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

const CATEGORY_LABEL: Record<IssueCategory, string> = {
  [IssueCategory.INFRASTRUCTURE]: 'Infrastructure',
  [IssueCategory.HEALTH]: 'Health',
  [IssueCategory.EDUCATION]: 'Education',
  [IssueCategory.SECURITY]: 'Security',
  [IssueCategory.ENVIRONMENT]: 'Environment',
  [IssueCategory.UTILITIES]: 'Utilities',
  [IssueCategory.TRANSPORT]: 'Transport',
  [IssueCategory.OTHER]: 'Other',
};

const STATUS_LABEL: Record<IssueStatus, string> = {
  [IssueStatus.OPEN]: 'Open',
  [IssueStatus.UNDER_REVIEW]: 'Under Review',
  [IssueStatus.IN_PROGRESS]: 'In Progress',
  [IssueStatus.RESOLVED]: 'Resolved',
  [IssueStatus.CLOSED]: 'Closed',
};

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
  const { id = '' } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const issueQuery = useIssue(id);
  const communitiesQuery = useCommunities();
  const meQuery = useMe();
  const location = useLocation();
  const commentsRef = useRef<HTMLDivElement | null>(null);

  // If the user arrived from a notification link like /issues/:id#comments,
  // scroll to the discussion thread once the page is mounted.
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
    // Update the upvoted-set optimistically and reconcile with the server
    // response so the button flips state immediately, no full refetch needed.
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
      // Server is the source of truth for the count. Patch the cached issue
      // instead of refetching so the number changes without a flash.
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
    return <p className="text-sm text-slate-500">Loading…</p>;
  }

  if (issueQuery.isError || !issueQuery.data) {
    return (
      <section className="space-y-4">
        <Link to="/issues" className="text-sm font-semibold text-civic-700 hover:underline">
          ← Back to issues
        </Link>
        <p className="text-sm text-red-600">Couldn't load this issue. It may have been removed.</p>
      </section>
    );
  }

  const issue = issueQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === issue.communityId);
  const reportedAt = new Date(issue.createdAt).toLocaleString();
  const isStaff = meQuery.data?.role ? STAFF_ROLES.has(meQuery.data.role) : false;

  return (
    <section className="space-y-6">
      <Link to="/issues" className="text-sm font-semibold text-civic-700 hover:underline">
        ← Back to issues
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              {CATEGORY_LABEL[issue.category]}
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">{issue.title}</h1>
            <p className="mt-2 text-sm text-slate-500">
              Reported {reportedAt}
              {community ? ` · ${community.name}, ${community.lga}` : ''}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[issue.status]}`}
            >
              {STATUS_LABEL[issue.status]}
            </span>
            <ShareButton title={issue.title} />
          </div>
        </div>

        <StatusTimeline current={issue.status} />

        {isStaff && (
          <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-slate-100 pt-4">
            <p className="mr-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
              Update status
            </p>
            {(Object.keys(STATUS_LABEL) as IssueStatus[]).map((s) => (
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
                {STATUS_LABEL[s]}
              </button>
            ))}
            {statusMutation.isError && (
              <span className="text-xs text-red-600">Could not update.</span>
            )}
          </div>
        )}
      </header>

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Description</h2>
        <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{issue.description}</p>
        {issue.location && (
          <div className="mt-4 border-t border-slate-100 pt-4">
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
              Location
            </p>
            <p className="mt-1 text-sm text-slate-700">{issue.location}</p>
          </div>
        )}
      </article>

      {issue.imageUrls && issue.imageUrls.length > 0 && (
        <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Photos</h2>
          <ImageGallery filenames={issue.imageUrls} alt="Issue photo" />
        </article>
      )}

      <article className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Community support
          </p>
          <p className="mt-1 text-2xl font-semibold text-slate-900">
            {issue.upvoteCount}{' '}
            <span className="text-sm font-normal text-slate-500">
              {issue.upvoteCount === 1 ? 'upvote' : 'upvotes'}
            </span>
          </p>
        </div>
        <Button
          onClick={() => upvoteMutation.mutate()}
          loading={upvoteMutation.isPending}
          variant={hasUpvoted ? 'secondary' : 'primary'}
        >
          {hasUpvoted ? '✓ Upvoted' : '▲ Upvote'}
        </Button>
      </article>

      <div id="comments" ref={commentsRef}>
        <CommentsSection entityType="issues" entityId={issue.id} />
      </div>
    </section>
  );
}

function StatusTimeline({ current }: { current: IssueStatus }) {
  if (current === IssueStatus.CLOSED) {
    return (
      <p className="mt-5 text-xs italic text-slate-500">
        This issue was closed without resolution.
      </p>
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
              {STATUS_LABEL[s]}
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
