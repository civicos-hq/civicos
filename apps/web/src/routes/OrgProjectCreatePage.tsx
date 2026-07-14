import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { ProjectStatus } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { getApiError } from '../lib/api';
import { useCommunities } from '../hooks/useCommunities';
import { useCreateProject, nairaToKobo } from '../hooks/useProjects';

const STATUSES: ProjectStatus[] = [
  ProjectStatus.PLANNED,
  ProjectStatus.ACTIVE,
  ProjectStatus.PAUSED,
  ProjectStatus.COMPLETED,
  ProjectStatus.CANCELLED,
];

// Native <input type="date"> gives us YYYY-MM-DD which the API accepts
// as an ISO date-time when we send it back. No round-trip to Date object
// is needed on submit — trust the browser's format.

export function OrgProjectCreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId } = useParams<{ orgId: string }>();
  const communitiesQuery = useCommunities();
  const createMutation = useCreateProject(orgId);

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [status, setStatus] = useState<ProjectStatus>(ProjectStatus.PLANNED);
  const [startDate, setStartDate] = useState('');
  const [expectedEndDate, setExpectedEndDate] = useState('');
  const [budgetNaira, setBudgetNaira] = useState('');
  const [communityId, setCommunityId] = useState('');
  const [error, setError] = useState('');

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      const created = await createMutation.mutateAsync({
        title: title.trim(),
        description,
        status,
        startDate: startDate ? new Date(startDate).toISOString() : undefined,
        expectedEndDate: expectedEndDate ? new Date(expectedEndDate).toISOString() : undefined,
        budgetKobo: budgetNaira ? nairaToKobo(budgetNaira) : undefined,
        communityId: communityId || undefined,
      });
      navigate(`/org/${orgId}/projects/${created.id}`);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgProjectCreate.error'));
    }
  }

  return (
    <section className="space-y-6">
      <Link
        to={`/org/${orgId}?tab=projects`}
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('orgProjectCreate.back')}
      </Link>

      <PageHeader
        eyebrow={t('orgProjectCreate.eyebrow')}
        title={t('orgProjectCreate.title')}
        subtitle={t('orgProjectCreate.subtitle')}
      />

      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgProjectCreate.fields.title')}
          </label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} required minLength={2} />
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgProjectCreate.fields.description')}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            minLength={10}
            rows={6}
            className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgProjectCreate.fields.status')}
          </label>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as ProjectStatus)}
            className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
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
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgProjectCreate.fields.startDate')}
            </label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            />
          </div>
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgProjectCreate.fields.expectedEndDate')}
            </label>
            <input
              type="date"
              value={expectedEndDate}
              onChange={(e) => setExpectedEndDate(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgProjectCreate.fields.budget')}
          </label>
          <div className="mt-1 flex items-center gap-2">
            <span className="text-lg text-slate-500 dark:text-slate-400">₦</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={budgetNaira}
              onChange={(e) => setBudgetNaira(e.target.value)}
              className="w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
              placeholder="e.g. 500000"
            />
          </div>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            {t('orgProjectCreate.fields.budgetHelp')}
          </p>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgProjectCreate.fields.community')}
          </label>
          <select
            value={communityId}
            onChange={(e) => setCommunityId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          >
            <option value="">{t('orgProjectCreate.fields.communityNone')}</option>
            {(communitiesQuery.data ?? []).map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.state} · {c.lga})
              </option>
            ))}
          </select>
        </div>

        {error && (
          <p className="rounded-lg border border-red-200 dark:border-red-500/40 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-300">
            {error}
          </p>
        )}

        <div className="flex justify-end gap-3">
          <Link to={`/org/${orgId}?tab=projects`}>
            <Button variant="ghost">{t('common.cancel')}</Button>
          </Link>
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? t('common.saving') : t('orgProjectCreate.create')}
          </Button>
        </div>
      </form>
    </section>
  );
}
