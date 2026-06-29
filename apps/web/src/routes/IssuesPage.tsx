import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Input } from '@civicos/ui';
import { IssueCategory, IssueStatus, type ApiResponse, type Issue } from '@civicos/types';
import { api, uploadImage, uploadUrl } from '../lib/api';
import { useMe } from '../hooks/useMe';

const MAX_IMAGES = 5;
const MAX_IMAGE_MB = 5;

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

const LANES: { label: string; status: IssueStatus; tone: string }[] = [
  { label: 'Open', status: IssueStatus.OPEN, tone: 'bg-rose-100 text-rose-700' },
  { label: 'Under Review', status: IssueStatus.UNDER_REVIEW, tone: 'bg-amber-100 text-amber-700' },
  { label: 'Resolved', status: IssueStatus.RESOLVED, tone: 'bg-emerald-100 text-emerald-700' },
];

type IssueSort = 'newest' | 'oldest' | 'upvotes' | 'comments';

const SORT_OPTIONS: { value: IssueSort; label: string }[] = [
  { value: 'newest', label: 'Newest' },
  { value: 'oldest', label: 'Oldest' },
  { value: 'upvotes', label: 'Most upvoted' },
  { value: 'comments', label: 'Most discussed' },
];

function useIssues(communityId: string | undefined, status: string, category: string) {
  return useQuery({
    queryKey: ['issues', communityId ?? 'all', status, category],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (communityId) params.communityId = communityId;
      if (status) params.status = status;
      if (category) params.category = category;
      const res = await api.get<ApiResponse<{ issues: Issue[] }>>('/api/v1/issues', { params });
      return res.data.data.issues;
    },
  });
}

function sortIssues(issues: Issue[], sort: IssueSort): Issue[] {
  const copy = [...issues];
  switch (sort) {
    case 'oldest':
      return copy.sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1));
    case 'upvotes':
      return copy.sort((a, b) => b.upvoteCount - a.upvoteCount);
    case 'comments':
      return copy.sort((a, b) => b.commentCount - a.commentCount);
    case 'newest':
    default:
      return copy.sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));
  }
}

export function IssuesPage() {
  const meQuery = useMe();
  const communityId = meQuery.data?.communityId;
  const [params, setParams] = useSearchParams();
  const statusFilter = params.get('status') ?? '';
  const categoryFilter = params.get('category') ?? '';
  const sort = (params.get('sort') as IssueSort) || 'newest';

  function setFilter(key: 'status' | 'category' | 'sort', value: string) {
    const next = new URLSearchParams(params);
    if (!value) next.delete(key);
    else next.set(key, value);
    setParams(next, { replace: true });
  }

  // Filtered list for the main grid.
  const issuesQuery = useIssues(communityId, statusFilter, categoryFilter);
  // Unfiltered baseline so the lane tiles always show the community-wide counts.
  const baselineQuery = useIssues(communityId, '', '');
  const [isModalOpen, setModalOpen] = useState(false);

  const issues = issuesQuery.data ?? [];
  const baseline = baselineQuery.data ?? [];
  const hasCommunity = Boolean(communityId);
  const hasFilters = Boolean(statusFilter || categoryFilter);

  const lanes = LANES.map((lane) => ({
    ...lane,
    count: baseline.filter((i) => i.status === lane.status).length,
  }));

  const visible = sortIssues(issues, sort);

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Issue Desk
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">Community signal board</h1>
          </div>
          <Button
            size="sm"
            onClick={() => setModalOpen(true)}
            disabled={!hasCommunity}
            title={hasCommunity ? undefined : 'Join a community first'}
          >
            + Report Issue
          </Button>
        </div>
        {!meQuery.isLoading && !hasCommunity && (
          <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
            You haven't joined a community yet.{' '}
            <Link to="/community" className="font-semibold underline">
              Pick one
            </Link>{' '}
            to start reporting issues.
          </p>
        )}
      </header>

      <div className="grid gap-4 md:grid-cols-3">
        {lanes.map((lane) => (
          <article
            key={lane.label}
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
              {lane.label}
            </p>
            <div className="mt-3 flex items-center justify-between">
              <p className="text-3xl font-semibold text-slate-900">{lane.count}</p>
              <span className={`rounded-full px-2.5 py-1 text-xs font-semibold ${lane.tone}`}>
                {lane.label}
              </span>
            </div>
          </article>
        ))}
      </div>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-semibold text-slate-900">
            {hasFilters ? 'Filtered reports' : 'Recent reports'}
            <span className="ml-2 text-sm font-normal text-slate-500">({visible.length})</span>
          </h2>
          <div className="flex items-center gap-2">
            {hasFilters && (
              <button
                type="button"
                onClick={() => {
                  const next = new URLSearchParams(params);
                  next.delete('status');
                  next.delete('category');
                  setParams(next, { replace: true });
                }}
                className="text-xs font-semibold text-civic-700 hover:underline"
              >
                Clear filters
              </button>
            )}
            <label className="text-xs font-semibold text-slate-500">
              Sort
              <select
                value={sort}
                onChange={(e) => setFilter('sort', e.target.value)}
                className="ml-2 rounded-lg border border-slate-200 bg-white px-2 py-1 text-sm font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-civic-500"
              >
                {SORT_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          <FilterPill
            active={!statusFilter}
            onClick={() => setFilter('status', '')}
            label="All status"
          />
          {(Object.keys(STATUS_LABEL) as IssueStatus[]).map((s) => (
            <FilterPill
              key={s}
              active={statusFilter === s}
              onClick={() => setFilter('status', s)}
              label={STATUS_LABEL[s]}
            />
          ))}
        </div>
        <div className="mt-2 flex flex-wrap gap-2">
          <FilterPill
            active={!categoryFilter}
            onClick={() => setFilter('category', '')}
            label="All categories"
          />
          {(Object.keys(CATEGORY_LABEL) as IssueCategory[]).map((cat) => (
            <FilterPill
              key={cat}
              active={categoryFilter === cat}
              onClick={() => setFilter('category', cat)}
              label={CATEGORY_LABEL[cat]}
            />
          ))}
        </div>

        {issuesQuery.isLoading ? (
          <p className="mt-4 text-sm text-slate-500">Loading…</p>
        ) : visible.length === 0 ? (
          <p className="mt-6 text-sm text-slate-500">
            {hasFilters
              ? 'No issues match the current filters.'
              : hasCommunity
                ? 'No issues yet. Use the button above to report the first one.'
                : 'Join a community to see local reports.'}
          </p>
        ) : (
          <div className="mt-4 grid gap-3">
            {visible.map((issue) => {
              const thumb = issue.imageUrls?.[0];
              const extra = (issue.imageUrls?.length ?? 0) - 1;
              return (
                <Link
                  key={issue.id}
                  to={`/issues/${issue.id}`}
                  className="flex gap-4 rounded-xl border border-slate-200 bg-slate-50/70 p-4 transition hover:border-civic-300 hover:bg-white"
                >
                  {thumb ? (
                    <div className="relative h-20 w-20 flex-shrink-0">
                      <img
                        src={uploadUrl(thumb)}
                        alt=""
                        className="h-full w-full rounded-lg object-cover ring-1 ring-slate-200"
                      />
                      {extra > 0 && (
                        <span className="absolute bottom-1 right-1 rounded-full bg-slate-900/80 px-1.5 py-0.5 text-[10px] font-semibold text-white">
                          +{extra}
                        </span>
                      )}
                    </div>
                  ) : null}
                  <div className="flex min-w-0 flex-1 flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <h3 className="truncate font-semibold text-slate-900">{issue.title}</h3>
                      <p className="mt-1 text-sm text-slate-600">
                        {CATEGORY_LABEL[issue.category]}
                        {issue.location ? ` · ${issue.location}` : ''}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-semibold text-slate-900">
                        {issue.upvoteCount} upvotes
                      </p>
                      <p className="text-xs text-slate-500">
                        {issue.commentCount} {issue.commentCount === 1 ? 'comment' : 'comments'} ·{' '}
                        {STATUS_LABEL[issue.status]}
                      </p>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </section>

      {isModalOpen && hasCommunity && communityId && (
        <ReportIssueModal communityId={communityId} onClose={() => setModalOpen(false)} />
      )}
    </section>
  );
}

// ─── Modal shell ──────────────────────────────────────────────────────────────

function FilterPill({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        active
          ? 'rounded-full bg-civic-700 px-3 py-1 text-xs font-semibold text-white shadow-sm'
          : 'rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-medium text-slate-600 hover:border-civic-300 hover:text-civic-700'
      }
    >
      {label}
    </button>
  );
}

function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
          <button
            type="button"
            className="text-slate-400 hover:text-slate-600"
            onClick={onClose}
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div className="mt-4">{children}</div>
      </div>
    </div>
  );
}

// ─── Report Issue ─────────────────────────────────────────────────────────────

function ReportIssueModal({ communityId, onClose }: { communityId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState<IssueCategory>(IssueCategory.INFRASTRUCTURE);
  const [location, setLocation] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState('');

  const previews = files.map((f) => ({ name: f.name, url: URL.createObjectURL(f) }));
  useEffect(() => {
    return () => previews.forEach((p) => URL.revokeObjectURL(p.url));
  }, [files]);

  function onFilesChange(e: React.ChangeEvent<HTMLInputElement>) {
    const picked = Array.from(e.target.files ?? []);
    e.target.value = ''; // allow re-selecting the same file later
    if (!picked.length) return;
    setError('');

    const tooBig = picked.find((f) => f.size > MAX_IMAGE_MB * 1024 * 1024);
    if (tooBig) {
      setError(`"${tooBig.name}" is over ${MAX_IMAGE_MB}MB.`);
      return;
    }
    const merged = [...files, ...picked].slice(0, MAX_IMAGES);
    setFiles(merged);
  }

  function removeFile(index: number) {
    setFiles(files.filter((_, i) => i !== index));
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const imageUrls = files.length ? await Promise.all(files.map(uploadImage)) : undefined;
      await api.post('/api/v1/issues', {
        title,
        description,
        category,
        communityId,
        location: location.trim() || undefined,
        imageUrls,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['issues'] });
      onClose();
    },
    onError: () => setError('Could not create issue. Check your inputs and try again.'),
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title="Report an issue" onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Streetlight outage on Adeniran"
          required
          minLength={5}
        />

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="description">
            Description
          </label>
          <textarea
            id="description"
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What's happening, and how is it affecting people?"
            required
            minLength={10}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="category">
            Category
          </label>
          <select
            id="category"
            className="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-civic-500"
            value={category}
            onChange={(e) => setCategory(e.target.value as IssueCategory)}
          >
            {(Object.values(IssueCategory) as IssueCategory[]).map((c) => (
              <option key={c} value={c}>
                {CATEGORY_LABEL[c]}
              </option>
            ))}
          </select>
        </div>

        <Input
          label="Location (optional)"
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="Street, landmark, or coordinates"
        />

        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium text-gray-700">
            Photos (optional, up to {MAX_IMAGES})
          </label>
          <label
            className={`flex cursor-pointer items-center justify-center rounded-lg border border-dashed border-gray-300 px-3 py-3 text-sm text-gray-600 transition hover:border-civic-400 hover:bg-civic-50 ${files.length >= MAX_IMAGES ? 'pointer-events-none opacity-50' : ''}`}
          >
            <input
              type="file"
              accept="image/*"
              multiple
              className="hidden"
              onChange={onFilesChange}
              disabled={files.length >= MAX_IMAGES}
            />
            {files.length >= MAX_IMAGES
              ? `Maximum ${MAX_IMAGES} photos selected`
              : `Add photos · ${MAX_IMAGE_MB}MB max each`}
          </label>

          {previews.length > 0 && (
            <ul className="grid grid-cols-3 gap-2">
              {previews.map((p, i) => (
                <li key={p.url} className="relative">
                  <img
                    src={p.url}
                    alt={p.name}
                    className="h-20 w-full rounded-lg object-cover ring-1 ring-slate-200"
                  />
                  <button
                    type="button"
                    onClick={() => removeFile(i)}
                    className="absolute -right-1.5 -top-1.5 rounded-full bg-slate-900 px-1.5 py-0.5 text-[10px] font-bold text-white"
                    aria-label={`Remove ${p.name}`}
                  >
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Report issue
          </Button>
        </div>
      </form>
    </Modal>
  );
}
