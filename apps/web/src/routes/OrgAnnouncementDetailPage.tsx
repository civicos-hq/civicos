import { useEffect, useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { AnnouncementStatus } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { getApiError } from '../lib/api';
import {
  useAnnouncement,
  useArchiveAnnouncement,
  useDeleteAnnouncement,
  usePublishAnnouncement,
  useUpdateAnnouncement,
} from '../hooks/useAnnouncements';
import { useMyOrganizations } from '../hooks/useConsultations';

const TONE: Record<AnnouncementStatus, string> = {
  [AnnouncementStatus.DRAFT]: 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300',
  [AnnouncementStatus.PUBLISHED]:
    'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  [AnnouncementStatus.ARCHIVED]:
    'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
};

// Announcements are lower-stakes than consultations — the API allows
// editing title/body across all statuses, and there's no citizen-side
// commitment to the previous version (like signatures on a petition).
// So the edit form stays available past DRAFT, unlike consultations.
export function OrgAnnouncementDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId, id } = useParams<{ orgId: string; id: string }>();

  const { data: memberships = [] } = useMyOrganizations();
  const membership = memberships.find((m) => m.organization.id === orgId);

  const query = useAnnouncement(id);
  const updateMutation = useUpdateAnnouncement(id, orgId);
  const publishMutation = usePublishAnnouncement(id, orgId);
  const archiveMutation = useArchiveAnnouncement(id, orgId);
  const deleteMutation = useDeleteAnnouncement(id, orgId);

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [error, setError] = useState('');

  // Sync the form buffer once the underlying announcement loads so
  // clicking "Edit" doesn't wipe the fields to empty.
  useEffect(() => {
    if (query.data) {
      setTitle(query.data.title);
      setBody(query.data.body);
    }
  }, [query.data]);

  if (query.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }
  if (query.isError || !query.data) {
    return (
      <section className="space-y-4">
        <Link
          to={`/org/${orgId}?tab=announcements`}
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('orgAnnouncementDetail.back')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">
          {t('orgAnnouncementDetail.loadError')}
        </p>
      </section>
    );
  }

  const a = query.data;
  const canAdmin =
    !!membership &&
    (membership.membership.role === 'OWNER' || membership.membership.role === 'ADMIN');
  const isDraft = a.status === AnnouncementStatus.DRAFT;
  const isPublished = a.status === AnnouncementStatus.PUBLISHED;

  async function onSaveEdit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await updateMutation.mutateAsync({ title: title.trim(), body });
      setEditing(false);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgAnnouncementDetail.errors.save'));
    }
  }

  async function runMutation(fn: () => Promise<unknown>) {
    setError('');
    try {
      await fn();
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgAnnouncementDetail.errors.generic'));
    }
  }

  return (
    <section className="space-y-6">
      <Link
        to={`/org/${orgId}?tab=announcements`}
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('orgAnnouncementDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t(`orgDashboard.announcementStatus.${a.status}`)}
        title={a.title}
        titleAs="h2"
        actions={
          canAdmin && !editing ? (
            <div className="flex gap-2">
              <Button size="sm" variant="ghost" onClick={() => setEditing(true)}>
                {t('common.edit')}
              </Button>
              {isDraft && (
                <>
                  <Button
                    size="sm"
                    onClick={() => runMutation(() => publishMutation.mutateAsync())}
                    disabled={publishMutation.isPending}
                  >
                    {publishMutation.isPending
                      ? t('orgAnnouncementDetail.publishing')
                      : t('orgAnnouncementDetail.publish')}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      if (window.confirm(t('orgAnnouncementDetail.confirmDelete'))) {
                        runMutation(async () => {
                          await deleteMutation.mutateAsync();
                          navigate(`/org/${orgId}?tab=announcements`);
                        });
                      }
                    }}
                    disabled={deleteMutation.isPending}
                  >
                    {t('common.delete')}
                  </Button>
                </>
              )}
              {isPublished && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    if (window.confirm(t('orgAnnouncementDetail.confirmArchive'))) {
                      runMutation(() => archiveMutation.mutateAsync());
                    }
                  }}
                  disabled={archiveMutation.isPending}
                >
                  {archiveMutation.isPending
                    ? t('orgAnnouncementDetail.archiving')
                    : t('orgAnnouncementDetail.archive')}
                </Button>
              )}
            </div>
          ) : undefined
        }
      />

      {error && (
        <p className="rounded-lg border border-red-200 dark:border-red-500/40 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      )}

      <p className="text-xs text-slate-500 dark:text-slate-300">
        {t('orgAnnouncementDetail.byline', {
          name: a.authorName,
          date: new Date(a.createdAt).toLocaleDateString(),
        })}
        {a.publishedAt && (
          <>
            {' · '}
            {t('orgDashboard.publishedOn', {
              date: new Date(a.publishedAt).toLocaleDateString(),
            })}
          </>
        )}
      </p>

      {editing ? (
        <form onSubmit={onSaveEdit} className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgAnnouncementCreate.fields.title')}
            </label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
              minLength={2}
            />
          </div>
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgAnnouncementCreate.fields.body')}
            </label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={10}
              required
              minLength={10}
              className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button
              variant="ghost"
              type="button"
              onClick={() => {
                setEditing(false);
                setTitle(a.title);
                setBody(a.body);
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
        <article className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-4 md:p-6 shadow-sm">
          <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">{a.body}</p>
          <span
            className={
              'mt-4 inline-block rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
              TONE[a.status]
            }
          >
            {t(`orgDashboard.announcementStatus.${a.status}`)}
          </span>
        </article>
      )}
    </section>
  );
}
