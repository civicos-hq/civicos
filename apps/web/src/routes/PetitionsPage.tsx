import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { PetitionStatus, type ApiResponse, type Petition } from '@civicos/types';
import { api, uploadImage } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useEnumLabels } from '../hooks/useEnumLabels';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { CommunityGate, CommunityGateLink } from '../components/CommunityGate';
import { FileText } from 'lucide-react';

const MAX_IMAGES = 5;
const MAX_IMAGE_MB = 5;

// Tone maps stay in-page; labels come from enums.petitionStatus.*
const STATUS_TONE: Record<PetitionStatus, string> = {
  [PetitionStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [PetitionStatus.ACTIVE]: 'bg-civic-100 text-civic-700',
  [PetitionStatus.CLOSED]: 'bg-amber-100 text-amber-700',
  [PetitionStatus.SUCCESSFUL]: 'bg-emerald-100 text-emerald-700',
};

type PetitionSort = 'newest' | 'oldest' | 'signatures' | 'deadline';

const SORT_VALUES: PetitionSort[] = ['newest', 'oldest', 'signatures', 'deadline'];

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
  const { t } = useTranslation();
  const enums = useEnumLabels();
  const meta = useTodayMeta();
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
      <PageHeader
        eyebrow={t('petitionsPage.eyebrow')}
        title={t('petitionsPage.title')}
        meta={meta}
        actions={
          <Button
            size="sm"
            onClick={() => setModalOpen(true)}
            disabled={!hasCommunity}
            title={hasCommunity ? undefined : t('petitionsPage.joinCommunityFirst')}
          >
            {t('petitionsPage.newBtn')}
          </Button>
        }
      >
        {!meQuery.isLoading && !hasCommunity && (
          <CommunityGate>
            {t('petitionsPage.noCommunityBanner')}{' '}
            <CommunityGateLink>{t('petitionsPage.pickOne')}</CommunityGateLink>{' '}
            {t('petitionsPage.toStart')}
          </CommunityGate>
        )}
      </PageHeader>

      <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-2">
            <FilterPill
              active={!statusFilter}
              onClick={() => setFilter('status', '')}
              label={t('petitionsPage.filters.all')}
            />
            {(Object.values(PetitionStatus) as PetitionStatus[]).map((s) => (
              <FilterPill
                key={s}
                active={statusFilter === s}
                onClick={() => setFilter('status', s)}
                label={enums.petitionStatus(s)}
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
                {t('petitionsPage.clear')}
              </button>
            )}
            <label className="text-xs font-semibold text-slate-600">
              {t('common.sortBy')}
              <select
                value={sort}
                onChange={(e) => setFilter('sort', e.target.value)}
                className="ml-2 rounded-lg border border-slate-200 bg-white px-2 py-1 text-sm font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-civic-500"
              >
                {SORT_VALUES.map((value) => (
                  <option key={value} value={value}>
                    {t(`petitionsPage.sort.${value}`)}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      </section>

      {petitionsQuery.isLoading ? (
        <p className="text-sm text-slate-600">{t('common.loading')}</p>
      ) : visible.length === 0 ? (
        <EmptyState
          icon={<FileText className="h-5 w-5" />}
          title={
            hasFilter
              ? t('petitionsPage.empty.noMatch')
              : hasCommunity
                ? t('petitionsPage.empty.noneYet')
                : t('petitionsPage.empty.needCommunity')
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {visible.map((petition) => {
            const progress = Math.min(
              100,
              Math.round((petition.signatureCount / petition.goal) * 100),
            );
            const days = daysUntil(petition.deadline);
            const deadlineFragment =
              days === null
                ? ''
                : days === 0
                  ? ` · ${t('petitionsPage.deadline.endsToday')}`
                  : ` · ${t('petitionsPage.deadline.daysLeft', { count: days })}`;
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
                    {enums.petitionStatus(petition.status)}
                  </span>
                </div>
                <p className="mt-2 text-sm text-slate-600">
                  {t('petitionsPage.signaturesOf', {
                    count: petition.signatureCount,
                    goal: petition.goal.toLocaleString(),
                  })}
                </p>

                <div className="mt-4 h-2.5 rounded-full bg-slate-100">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-civic-700 to-civic-500"
                    style={{ width: `${progress}%` }}
                  />
                </div>

                <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm">
                  <span className="rounded-full bg-civic-50 px-2.5 py-1 font-semibold text-civic-700">
                    {t('petitionsPage.percentComplete', { percent: progress })}
                  </span>
                  <span className="text-slate-600">
                    {t('common.commentCount', { count: petition.commentCount })}
                    {deadlineFragment}
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
      aria-pressed={active}
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
  const { t } = useTranslation();
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
            aria-label={t('common.close')}
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
  const { t } = useTranslation();
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
      setError(t('petitionsPage.modal.tooBig', { name: tooBig.name, max: MAX_IMAGE_MB }));
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
    onError: () => setError(t('petitionsPage.modal.genericError')),
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('petitionsPage.modal.title')} onClose={onClose}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label={t('petitionsPage.modal.titleLabel')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder={t('petitionsPage.modal.titlePlaceholder')}
          required
          minLength={5}
        />

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="description">
            {t('petitionsPage.modal.description')}
          </label>
          <textarea
            id="description"
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('petitionsPage.modal.descriptionPlaceholder')}
            required
            minLength={10}
          />
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Input
            label={t('petitionsPage.modal.signatureGoal')}
            type="number"
            min={1}
            value={goal}
            onChange={(e) => setGoal(Number(e.target.value))}
            required
          />
          <Input
            label={t('petitionsPage.modal.deadline')}
            type="date"
            value={deadline}
            onChange={(e) => setDeadline(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium text-gray-700">
            {t('petitionsPage.modal.photos', { max: MAX_IMAGES })}
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
              ? t('petitionsPage.modal.photosFull', { max: MAX_IMAGES })
              : t('petitionsPage.modal.addPhotos', { max: MAX_IMAGE_MB })}
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
                    aria-label={t('petitionsPage.modal.removePhoto', { name: p.name })}
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
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('petitionsPage.modal.submit')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
