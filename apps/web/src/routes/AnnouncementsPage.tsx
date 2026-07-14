import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { Announcement } from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { usePublishedAnnouncements } from '../hooks/useAnnouncements';
import { Megaphone } from 'lucide-react';

// Truncate an announcement body for the list card. The API doesn't
// return a `summary` field the way consultations do — announcements
// are just title + body — so we squeeze a preview out of the body
// itself. Keeping the truncation short so the list stays scannable.
const PREVIEW_LEN = 220;

function preview(body: string): string {
  const trimmed = body.trim();
  if (trimmed.length <= PREVIEW_LEN) return trimmed;
  return trimmed.slice(0, PREVIEW_LEN).replace(/\s+\S*$/, '') + '…';
}

export function AnnouncementsPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  // The API returns up to 20 by default; asking for 50 gives the citizen
  // page a bit more depth without paying for pagination scaffolding yet.
  const query = usePublishedAnnouncements(50);

  const items = (query.data ?? []).sort((a, b) => {
    const at = a.publishedAt ?? a.createdAt;
    const bt = b.publishedAt ?? b.createdAt;
    return at < bt ? 1 : -1;
  });

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('announcementsPage.eyebrow')}
        title={t('announcementsPage.title')}
        subtitle={t('announcementsPage.subtitle')}
        meta={meta}
      />

      {query.isLoading && (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      )}

      {query.isError && (
        <p className="text-sm text-red-600 dark:text-red-400">{t('announcementsPage.loadError')}</p>
      )}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<Megaphone size={20} />}
          title={t('announcementsPage.empty.title')}
          body={t('announcementsPage.empty.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((a: Announcement) => (
          <li key={a.id}>
            <Link
              to={`/announcements/${a.id}`}
              className="block rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-5 shadow-sm transition hover:border-civic-300 dark:hover:border-civic-500 hover:shadow-md"
            >
              <h2 className="font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
                {a.title}
              </h2>
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{preview(a.body)}</p>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-slate-300">
                <span>
                  {t('announcementsPage.by')}{' '}
                  <Link
                    to={`/organizations/${a.organizationId}`}
                    className="font-semibold text-slate-700 dark:text-slate-300 hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {a.authorName}
                  </Link>
                </span>
                {a.publishedAt && (
                  <span>
                    {t('announcementsPage.publishedOn', {
                      date: new Date(a.publishedAt).toLocaleDateString(),
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
