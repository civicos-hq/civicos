import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import {
  IssueCategory,
  IssueStatus,
  type ApiResponse,
  type Community,
  type Issue,
} from '@civicos/types';
import { api, uploadUrl } from '../lib/api';
import { CommentsSection } from '../components/civic/CommentsSection';

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

  const upvoteMutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/issues/${id}/upvote`);
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
          <span
            className={`rounded-full px-3 py-1 text-xs font-semibold ${STATUS_TONE[issue.status]}`}
          >
            {STATUS_LABEL[issue.status]}
          </span>
        </div>
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
          <div className="mt-4 grid gap-3 sm:grid-cols-2 md:grid-cols-3">
            {issue.imageUrls.map((filename) => (
              <a
                key={filename}
                href={uploadUrl(filename)}
                target="_blank"
                rel="noreferrer"
                className="block overflow-hidden rounded-xl ring-1 ring-slate-200 transition hover:ring-civic-400"
              >
                <img
                  src={uploadUrl(filename)}
                  alt="Issue photo"
                  className="h-40 w-full object-cover"
                />
              </a>
            ))}
          </div>
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
        <Button onClick={() => upvoteMutation.mutate()} loading={upvoteMutation.isPending}>
          ▲ Upvote
        </Button>
      </article>

      <CommentsSection entityType="issues" entityId={issue.id} />
    </section>
  );
}
