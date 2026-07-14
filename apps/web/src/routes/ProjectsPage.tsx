import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ProjectStatus, type Project } from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useCitizenProjects } from '../hooks/useProjects';
import { Briefcase } from 'lucide-react';

const STATUS_TONE: Record<ProjectStatus, string> = {
  [ProjectStatus.PLANNED]: 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300',
  [ProjectStatus.ACTIVE]: 'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  [ProjectStatus.PAUSED]: 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
  [ProjectStatus.COMPLETED]:
    'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
  [ProjectStatus.CANCELLED]: 'bg-slate-300 text-slate-700 dark:text-slate-300',
};

// Only three status filter chips at the top — Active / Completed / All —
// so citizens land on the most relevant view. Planned + Paused + Cancelled
// are still returned by "All" but rarely deserve their own chip on a
// citizen surface. Admins get the full picture on the org dashboard.
const FILTERS: Array<{ value: '' | ProjectStatus; i18nKey: string }> = [
  { value: '', i18nKey: 'projectsPage.filters.all' },
  { value: ProjectStatus.ACTIVE, i18nKey: 'projectsPage.filters.active' },
  { value: ProjectStatus.COMPLETED, i18nKey: 'projectsPage.filters.completed' },
];

function formatNaira(kobo?: number | null): string {
  if (kobo === undefined || kobo === null) return '';
  return `₦${(kobo / 100).toLocaleString()}`;
}

export function ProjectsPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const [params, setParams] = useSearchParams();
  const statusFilter = (params.get('status') as ProjectStatus | null) ?? '';
  const query = useCitizenProjects({ status: statusFilter });

  function setStatus(next: '' | ProjectStatus) {
    const p = new URLSearchParams(params);
    if (next) p.set('status', next);
    else p.delete('status');
    setParams(p, { replace: true });
  }

  const items = ((query.data ?? []) as Project[]).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('projectsPage.eyebrow')}
        title={t('projectsPage.title')}
        subtitle={t('projectsPage.subtitle')}
        meta={meta}
      />

      <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-4 shadow-sm">
        <div className="flex flex-wrap gap-2">
          {FILTERS.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setStatus(f.value)}
              className={
                'rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-wide transition ' +
                (statusFilter === f.value
                  ? 'border-civic-500 bg-civic-50 dark:bg-civic-500/10 text-civic-700 dark:text-civic-200'
                  : 'border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 text-slate-600 dark:text-slate-400 hover:border-slate-300 dark:hover:border-slate-600')
              }
            >
              {t(f.i18nKey)}
            </button>
          ))}
        </div>
      </section>

      {query.isLoading && (
        <p className="text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>
      )}

      {query.isError && (
        <p className="text-sm text-red-600 dark:text-red-400">{t('projectsPage.loadError')}</p>
      )}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<Briefcase size={20} />}
          title={t('projectsPage.empty.title')}
          body={t('projectsPage.empty.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((p) => (
          <li key={p.id}>
            <Link
              to={`/projects/${p.id}`}
              className="block rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-5 shadow-sm transition hover:border-civic-300 dark:hover:border-civic-500 hover:shadow-md"
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <h2 className="font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
                    {p.title}
                  </h2>
                  <p className="mt-1 line-clamp-2 text-sm text-slate-600 dark:text-slate-400">
                    {p.description}
                  </p>
                </div>
                <span
                  className={
                    'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
                    STATUS_TONE[p.status]
                  }
                >
                  {t(`orgDashboard.projectStatus.${p.status}`)}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                <Link
                  to={`/organizations/${p.organizationId}`}
                  onClick={(e) => e.stopPropagation()}
                  className="font-semibold text-slate-700 dark:text-slate-300 hover:underline"
                >
                  {t('projectsPage.viewOrg')}
                </Link>
                {formatNaira(p.budgetKobo) !== '' && <span>{formatNaira(p.budgetKobo)}</span>}
                {p.startDate && (
                  <span>
                    {t('projectsPage.startsOn', {
                      date: new Date(p.startDate).toLocaleDateString(),
                    })}
                  </span>
                )}
                {p.expectedEndDate && (
                  <span>
                    {t('projectsPage.endsOn', {
                      date: new Date(p.expectedEndDate).toLocaleDateString(),
                    })}
                  </span>
                )}
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
