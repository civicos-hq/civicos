import { useEffect, useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { ProjectStatus } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { getApiError } from '../lib/api';
import {
  useDeleteProject,
  useProject,
  useUpdateProject,
  kobopToNaira,
  nairaToKobo,
} from '../hooks/useProjects';
import { useMyOrganizations } from '../hooks/useConsultations';

const TONE: Record<ProjectStatus, string> = {
  [ProjectStatus.PLANNED]: 'bg-slate-200 text-slate-700',
  [ProjectStatus.ACTIVE]: 'bg-civic-100 text-civic-700',
  [ProjectStatus.PAUSED]: 'bg-amber-100 text-amber-700',
  [ProjectStatus.COMPLETED]: 'bg-emerald-100 text-emerald-700',
  [ProjectStatus.CANCELLED]: 'bg-slate-300 text-slate-700',
};

const STATUSES: ProjectStatus[] = [
  ProjectStatus.PLANNED,
  ProjectStatus.ACTIVE,
  ProjectStatus.PAUSED,
  ProjectStatus.COMPLETED,
  ProjectStatus.CANCELLED,
];

// Native date input wants YYYY-MM-DD. The API returns full ISO
// timestamps, so we slice the date portion for the form buffer.
function toDateInput(iso?: string): string {
  return iso ? iso.slice(0, 10) : '';
}

export function OrgProjectDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId, id } = useParams<{ orgId: string; id: string }>();

  const { data: memberships = [] } = useMyOrganizations();
  const membership = memberships.find((m) => m.organization.id === orgId);

  const query = useProject(id);
  const updateMutation = useUpdateProject(id, orgId);
  const deleteMutation = useDeleteProject(id, orgId);

  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    status: ProjectStatus.PLANNED as ProjectStatus,
    startDate: '',
    expectedEndDate: '',
    budgetNaira: '' as string,
  });
  const [error, setError] = useState('');

  useEffect(() => {
    if (query.data) {
      const budget = kobopToNaira(query.data.budgetKobo ?? undefined);
      setForm({
        title: query.data.title,
        description: query.data.description,
        status: query.data.status,
        startDate: toDateInput(query.data.startDate),
        expectedEndDate: toDateInput(query.data.expectedEndDate),
        budgetNaira: budget === '' ? '' : String(budget),
      });
    }
  }, [query.data]);

  if (query.isLoading) return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  if (query.isError || !query.data) {
    return (
      <section className="space-y-4">
        <Link
          to={`/org/${orgId}?tab=projects`}
          className="text-sm font-semibold text-civic-700 hover:underline"
        >
          {t('orgProjectDetail.back')}
        </Link>
        <p className="text-sm text-red-600">{t('orgProjectDetail.loadError')}</p>
      </section>
    );
  }

  const p = query.data;
  const canAdmin =
    !!membership &&
    (membership.membership.role === 'OWNER' || membership.membership.role === 'ADMIN');

  async function onSave(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await updateMutation.mutateAsync({
        title: form.title.trim(),
        description: form.description,
        status: form.status,
        startDate: form.startDate ? new Date(form.startDate).toISOString() : undefined,
        expectedEndDate: form.expectedEndDate
          ? new Date(form.expectedEndDate).toISOString()
          : undefined,
        budgetKobo: form.budgetNaira ? nairaToKobo(form.budgetNaira) : undefined,
      });
      setEditing(false);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgProjectDetail.errors.save'));
    }
  }

  async function runMutation(fn: () => Promise<unknown>) {
    setError('');
    try {
      await fn();
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgProjectDetail.errors.generic'));
    }
  }

  const budgetDisplay = kobopToNaira(p.budgetKobo ?? undefined);

  return (
    <section className="space-y-6">
      <Link
        to={`/org/${orgId}?tab=projects`}
        className="text-sm font-semibold text-civic-700 hover:underline"
      >
        {t('orgProjectDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t(`orgDashboard.projectStatus.${p.status}`)}
        title={p.title}
        titleAs="h2"
        actions={
          canAdmin && !editing ? (
            <div className="flex gap-2">
              <Button size="sm" variant="ghost" onClick={() => setEditing(true)}>
                {t('common.edit')}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  if (window.confirm(t('orgProjectDetail.confirmDelete'))) {
                    runMutation(async () => {
                      await deleteMutation.mutateAsync();
                      navigate(`/org/${orgId}?tab=projects`);
                    });
                  }
                }}
                disabled={deleteMutation.isPending}
              >
                {t('common.delete')}
              </Button>
            </div>
          ) : undefined
        }
      />

      {error && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </p>
      )}

      {editing ? (
        <form onSubmit={onSave} className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-slate-700">
              {t('orgProjectCreate.fields.title')}
            </label>
            <Input
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              required
              minLength={2}
            />
          </div>

          <div>
            <label className="block text-sm font-semibold text-slate-700">
              {t('orgProjectCreate.fields.description')}
            </label>
            <textarea
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              rows={6}
              required
              minLength={10}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            />
          </div>

          <div>
            <label className="block text-sm font-semibold text-slate-700">
              {t('orgProjectCreate.fields.status')}
            </label>
            <select
              value={form.status}
              onChange={(e) => setForm({ ...form, status: e.target.value as ProjectStatus })}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            >
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {t(`orgDashboard.projectStatus.${s}`)}
                </option>
              ))}
            </select>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="block text-sm font-semibold text-slate-700">
                {t('orgProjectCreate.fields.startDate')}
              </label>
              <input
                type="date"
                value={form.startDate}
                onChange={(e) => setForm({ ...form, startDate: e.target.value })}
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
              />
            </div>
            <div>
              <label className="block text-sm font-semibold text-slate-700">
                {t('orgProjectCreate.fields.expectedEndDate')}
              </label>
              <input
                type="date"
                value={form.expectedEndDate}
                onChange={(e) => setForm({ ...form, expectedEndDate: e.target.value })}
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-semibold text-slate-700">
              {t('orgProjectCreate.fields.budget')}
            </label>
            <div className="mt-1 flex items-center gap-2">
              <span className="text-lg text-slate-500">₦</span>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.budgetNaira}
                onChange={(e) => setForm({ ...form, budgetNaira: e.target.value })}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
              />
            </div>
          </div>

          <div className="flex justify-end gap-3">
            <Button
              variant="ghost"
              type="button"
              onClick={() => {
                setEditing(false);
                // Reset form to server state — useEffect will re-sync
                // on next data change anyway.
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t('common.saving') : t('common.save')}
            </Button>
          </div>
        </form>
      ) : (
        <article className="space-y-4 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <span
            className={
              'inline-block rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
              TONE[p.status]
            }
          >
            {t(`orgDashboard.projectStatus.${p.status}`)}
          </span>
          <p className="whitespace-pre-wrap text-sm text-slate-700">{p.description}</p>
          <dl className="grid grid-cols-2 gap-4 text-sm text-slate-600">
            {p.startDate && (
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-400">
                  {t('orgProjectDetail.startDate')}
                </dt>
                <dd>{new Date(p.startDate).toLocaleDateString()}</dd>
              </div>
            )}
            {p.expectedEndDate && (
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-400">
                  {t('orgProjectDetail.expectedEndDate')}
                </dt>
                <dd>{new Date(p.expectedEndDate).toLocaleDateString()}</dd>
              </div>
            )}
            {budgetDisplay !== '' && (
              <div>
                <dt className="text-xs uppercase tracking-wide text-slate-400">
                  {t('orgProjectDetail.budget')}
                </dt>
                <dd>₦{(budgetDisplay as number).toLocaleString()}</dd>
              </div>
            )}
          </dl>
        </article>
      )}
    </section>
  );
}
