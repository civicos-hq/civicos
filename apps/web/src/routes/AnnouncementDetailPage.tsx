import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { AnnouncementStatus } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { useAnnouncement } from '../hooks/useAnnouncements';

const STATUS_TONE: Record<AnnouncementStatus, string> = {
  [AnnouncementStatus.DRAFT]: 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300',
  [AnnouncementStatus.PUBLISHED]:
    'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  [AnnouncementStatus.ARCHIVED]:
    'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
};

// Read-only citizen view of a single announcement. Any editing controls
// live only on the org-owner surface at /org/:orgId/announcements/:id
// (see OrgAnnouncementDetailPage). Draft rows may still come back from
// the API for archive-style bookmarking, but citizens land here through
// public links so the DRAFT case is rare.
export function AnnouncementDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const query = useAnnouncement(id);

  if (query.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }

  if (query.isError || !query.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/announcements"
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('announcementDetail.back')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">
          {t('announcementDetail.loadError')}
        </p>
      </section>
    );
  }

  const a = query.data;

  return (
    <section className="space-y-6">
      <Link
        to="/announcements"
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('announcementDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t('announcementDetail.eyebrow')}
        title={a.title}
        titleAs="h2"
        subtitle={
          <span>
            {t('announcementDetail.byline', {
              name: a.authorName,
              date: new Date(a.publishedAt ?? a.createdAt).toLocaleDateString(),
            })}
          </span>
        }
      />

      <article className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-4 md:p-6 shadow-sm">
        <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">{a.body}</p>
        <div className="mt-4 flex flex-wrap items-center gap-3 text-xs text-slate-500 dark:text-slate-300">
          <span
            className={
              'rounded-full px-2 py-0.5 font-semibold uppercase tracking-wide ' +
              STATUS_TONE[a.status]
            }
          >
            {t(`orgDashboard.announcementStatus.${a.status}`)}
          </span>
          <Link
            to={`/organizations/${a.organizationId}`}
            className="font-semibold text-civic-700 dark:text-civic-200 hover:underline"
          >
            {t('announcementDetail.viewOrg')}
          </Link>
        </div>
      </article>
    </section>
  );
}
