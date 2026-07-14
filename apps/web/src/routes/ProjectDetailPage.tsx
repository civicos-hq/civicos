import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ProjectStatus } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { kobopToNaira, useProject } from '../hooks/useProjects';
import { useProjectProgressUpdates } from '../hooks/useProgressUpdates';

const TONE: Record<ProjectStatus, string> = {
  [ProjectStatus.PLANNED]: 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300',
  [ProjectStatus.ACTIVE]: 'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  [ProjectStatus.PAUSED]: 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
  [ProjectStatus.COMPLETED]:
    'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
  [ProjectStatus.CANCELLED]: 'bg-slate-300 text-slate-700 dark:text-slate-300',
};

// Read-only citizen view of a project. Any editing controls live only
// on the org-owner surface at /org/:orgId/projects/:id. The public
// progress-update feed is fetched here so citizens can see what the
// responsible org has actually done.
export function ProjectDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const query = useProject(id);
  const updatesQuery = useProjectProgressUpdates(id);

  if (query.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>;
  }
  if (query.isError || !query.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/projects"
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('projectDetail.back')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">{t('projectDetail.loadError')}</p>
      </section>
    );
  }

  const p = query.data;
  const budgetDisplay = kobopToNaira(p.budgetKobo ?? undefined);
  const updates = updatesQuery.data ?? [];

  return (
    <section className="space-y-6">
      <Link
        to="/projects"
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('projectDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t(`orgDashboard.projectStatus.${p.status}`)}
        title={p.title}
        titleAs="h2"
      />

      <article className="space-y-4 rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <span
          className={
            'inline-block rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
            TONE[p.status]
          }
        >
          {t(`orgDashboard.projectStatus.${p.status}`)}
        </span>

        <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
          {p.description}
        </p>

        <dl className="grid grid-cols-1 gap-4 text-sm text-slate-600 dark:text-slate-400 md:grid-cols-2">
          <div>
            <dt className="text-xs uppercase tracking-wide text-slate-400">
              {t('projectDetail.owner')}
            </dt>
            <dd>
              <Link
                to={`/organizations/${p.organizationId}`}
                className="font-semibold text-civic-700 dark:text-civic-200 hover:underline"
              >
                {t('projectDetail.viewOrg')}
              </Link>
            </dd>
          </div>
          {p.startDate && (
            <div>
              <dt className="text-xs uppercase tracking-wide text-slate-400">
                {t('projectDetail.startDate')}
              </dt>
              <dd>{new Date(p.startDate).toLocaleDateString()}</dd>
            </div>
          )}
          {p.expectedEndDate && (
            <div>
              <dt className="text-xs uppercase tracking-wide text-slate-400">
                {t('projectDetail.expectedEndDate')}
              </dt>
              <dd>{new Date(p.expectedEndDate).toLocaleDateString()}</dd>
            </div>
          )}
          {budgetDisplay !== '' && (
            <div>
              <dt className="text-xs uppercase tracking-wide text-slate-400">
                {t('projectDetail.budget')}
              </dt>
              <dd>₦{(budgetDisplay as number).toLocaleString()}</dd>
            </div>
          )}
        </dl>
      </article>

      <section className="space-y-3 rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <h3 className="font-fraunces text-base font-semibold text-slate-900 dark:text-slate-100">
          {t('projectDetail.progressHeading')}
        </h3>
        {updatesQuery.isLoading && (
          <p className="text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>
        )}
        {!updatesQuery.isLoading && updates.length === 0 && (
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t('projectDetail.progressEmpty')}
          </p>
        )}
        <ul className="space-y-3">
          {updates.map((u) => (
            <li key={u.id} className="border-l-2 border-civic-400 pl-3">
              <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
                {u.body}
              </p>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {u.authorName} · {new Date(u.createdAt).toLocaleDateString()}
              </p>
            </li>
          ))}
        </ul>
      </section>
    </section>
  );
}
