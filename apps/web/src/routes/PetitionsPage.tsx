import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Input } from '@civicos/ui';
import { PetitionStatus, type ApiResponse, type Petition } from '@civicos/types';
import { api, uploadImage } from '../lib/api';
import { useMe } from '../hooks/useMe';

const MAX_IMAGES = 5;
const MAX_IMAGE_MB = 5;

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

type PetitionSort = 'newest' | 'oldest' | 'signatures' | 'deadline';

const SORT_OPTIONS: { value: PetitionSort; label: string }[] = [
  { value: 'newest', label: 'Newest' },
  { value: 'oldest', label: 'Oldest' },
  { value: 'signatures', label: 'Most signed' },
  { value: 'deadline', label: 'Closing soon' },
];

function usePetitions(communityId: string | undefined, status: string) {
  return useQuery({
    queryKey: ['petitions', communityId ?? 'all', status],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (communityId) params.communityId = communityId;
      if (status) params.status = status;
      const res = await api.get<ApiResponse<{ petitions: Petition[] }>>('/api/v1/petitions', {
        params,
      });
      return res.data.data.petitions;
    },
  });
}

function sortPetitions(list: Petition[], sort: PetitionSort): Petition[] {
  const copy = [...list];
  switch (sort) {
    case 'oldest':
      return copy.sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1));
    case 'signatures':
      return copy.sort((a, b) => b.signatureCount - a.signatureCount);
    case 'deadline':
      // Petitions with a deadline first, soonest first. No-deadline sink to bottom.
      return copy.sort((a, b) => {
        if (a.deadline && b.deadline)
          return new Date(a.deadline).getTime() - new Date(b.deadline).getTime();
        if (a.deadline) return -1;
        if (b.deadline) return 1;
        return 0;
      });
    case 'newest':
    default:
      return copy.sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));
  }
}

function daysUntil(deadlineISO?: string) {
  if (!deadlineISO) return null;
  const ms = new Date(deadlineISO).getTime() - Date.now();
  return Math.max(0, Math.ceil(ms / (1000 * 60 * 60 * 24)));
}

export function PetitionsPage() {
  const meQuery = useMe();
  const communityId = meQuery.data?.communityId;
  const [params, setParams] = useSearchParams();
  const statusFilter = params.get('status') ?? '';
  const sort = (params.get('sort') as PetitionSort) || 'newest';

  function setFilter(key: 'status' | 'sort', value: string) {
    const next = new URLSearchParams(params);
    if (!value) next.delete(key);
    else next.set(key, value);
    setParams(next, { replace: true });
  }

  const petitionsQuery = usePetitions(communityId, statusFilter);
  const [isModalOpen, setModalOpen] = useState(false);

  const petitions = petitionsQuery.data ?? [];
  const hasCommunity = Boolean(communityId);
  const visible = sortPetitions(petitions, sort);
  const hasFilter = Boolean(statusFilter);

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Petition Studio
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">
              Organize collective voice
            </h1>
          </div>
          <Button
            size="sm"
            onClick={() => setModalOpen(true)}
            disabled={!hasCommunity}
            title={hasCommunity ? undefined : 'Join a community first'}
          >
            + New Petition
          </Button>
        </div>
        {!meQuery.isLoading && !hasCommunity && (
          <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
            You haven't joined a community yet.{' '}
            <Link to="/community" className="font-semibold underline">
              Pick one
            </Link>{' '}
            to start petitions.
          </p>
        )}
      </header>

      <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-2">
            <FilterPill
              active={!statusFilter}
              onClick={() => setFilter('status', '')}
              label="All"
            />
            {(Object.keys(STATUS_LABEL) as PetitionStatus[]).map((s) => (
              <FilterPill
                key={s}
                active={statusFilter === s}
                onClick={() => setFilter('status', s)}
                label={STATUS_LABEL[s]}
              />
            ))}
          </div>
          <div className="flex items-center gap-2">
            {hasFilter && (
              <button
                type="button"
                onClick={() => setFilter('status', '')}
                className="text-xs font-semibold text-civic-700 hover:underline"
              >
                Clear
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
      </section>

      {petitionsQuery.isLoading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : visible.length === 0 ? (
        <article className="rounded-2xl border border-dashed border-slate-300 bg-white/60 p-8 text-center text-sm text-slate-500">
          {hasFilter
            ? 'No petitions match the current filter.'
            : hasCommunity
              ? 'No petitions yet. Start the first one in your community.'
              : 'Join a community to see local petitions.'}
        </article>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {visible.map((petition) => {
            const progress = Math.min(
              100,
              Math.round((petition.signatureCount / petition.goal) * 100),
            );
            const days = daysUntil(petition.deadline);
            return (
              <Link
                key={petition.id}
                to={`/petitions/${petition.id}`}
                className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
              >
                <div className="flex items-start justify-between gap-3">
                  <h2 className="text-lg font-semibold text-slate-900">{petition.title}</h2>
                  <span
                    className={`whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ${STATUS_TONE[petition.status]}`}
                  >
                    {STATUS_LABEL[petition.status]}
                  </span>
                </div>
                <p className="mt-2 text-sm text-slate-600">
                  {petition.signatureCount.toLocaleString()} signatures out of{' '}
                  {petition.goal.toLocaleString()} target
                </p>

                <div className="mt-4 h-2.5 rounded-full bg-slate-100">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-civic-700 to-civic-500"
                    style={{ width: `${progress}%` }}
                  />
                </div>

                <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm">
                  <span className="rounded-full bg-civic-50 px-2.5 py-1 font-semibold text-civic-700">
                    {progress}% complete
                  </span>
                  <span className="text-slate-500">
                    {petition.commentCount} {petition.commentCount === 1 ? 'comment' : 'comments'}
                    {days !== null
                      ? ` · ${days === 0 ? 'Ends today' : `${days} day${days === 1 ? '' : 's'} left`}`
                      : ''}
                  </span>
                </div>
              </Link>
            );
          })}
        </div>
      )}

      {isModalOpen && hasCommunity && communityId && (
        <NewPetitionModal communityId={communityId} onClose={() => setModalOpen(false)} />
      )}
    </section>
  );
}

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

// ─── Modal shell ──────────────────────────────────────────────────────────────

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

// ─── New Petition ─────────────────────────────────────────────────────────────

function NewPetitionModal({ communityId, onClose }: { communityId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [goal, setGoal] = useState(500);
  const [deadline, setDeadline] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState('');

  const previews = files.map((f) => ({ name: f.name, url: URL.createObjectURL(f) }));
  useEffect(() => {
    return () => previews.forEach((p) => URL.revokeObjectURL(p.url));
  }, [files]);

  function onFilesChange(e: React.ChangeEvent<HTMLInputElement>) {
    const picked = Array.from(e.target.files ?? []);
    e.target.value = '';
    if (!picked.length) return;
    setError('');
    const tooBig = picked.find((f) => f.size > MAX_IMAGE_MB * 1024 * 1024);
    if (tooBig) {
      setError(`"${tooBig.name}" is over ${MAX_IMAGE_MB}MB.`);
      return;
    }
    setFiles([...files, ...picked].slice(0, MAX_IMAGES));
  }

  function removeFile(index: number) {
    setFiles(files.filter((_, i) => i !== index));
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const imageUrls = files.length ? await Promise.all(files.map(uploadImage)) : undefined;
      await api.post('/api/v1/petitions', {
        title,
        description,
        goal,
        communityId,
        deadline: deadline ? new Date(deadline).toISOString() : undefined,
        imageUrls,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['petitions'] });
      onClose();
    },
    onError: () => setError('Could not create petition. Check your inputs and try again.'),
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title="Start a petition" onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Repair primary school access road"
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
            placeholder="Explain the change you want and why it matters."
            required
            minLength={10}
          />
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Input
            label="Signature goal"
            type="number"
            min={1}
            value={goal}
            onChange={(e) => setGoal(Number(e.target.value))}
            required
          />
          <Input
            label="Deadline (optional)"
            type="date"
            value={deadline}
            onChange={(e) => setDeadline(e.target.value)}
          />
        </div>

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
            Publish petition
          </Button>
        </div>
      </form>
    </Modal>
  );
}
