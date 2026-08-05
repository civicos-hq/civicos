import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Megaphone, Pencil, Trash2 } from 'lucide-react';
import { Button } from '@civicos/ui';
import { useRelativeTime } from '../../hooks/useRelativeTime';
import {
  useArchiveRepAnnouncement,
  useCreateRepAnnouncement,
  useDeleteRepAnnouncement,
  usePublishRepAnnouncement,
  useRepAnnouncements,
  useRepAnnouncementsManage,
  useUpdateRepAnnouncement,
  type RepAnnouncement,
} from '../../hooks/useRepAnnouncements';
import { EmptyState } from '../EmptyState';
import { CommentsSection } from './CommentsSection';
import { ReportButton } from './ReportButton';

/**
 * What a representative has said to their constituents, on their public
 * profile.
 *
 * Two audiences, one component. A visitor sees published announcements. The
 * representative whose profile this is also sees their drafts and archived
 * posts, plus the controls to write and publish.
 *
 * Ownership is decided by the server: the manage query 403s for anyone else,
 * and a failed query simply means the authoring UI does not render. The client
 * never gets its own copy of that rule to disagree with.
 */
export function RepAnnouncements({
  repId,
  repName,
  couldOwn,
}: {
  repId: string;
  repName: string;
  /** Gate the ownership probe so ordinary citizens never generate a 403. */
  couldOwn: boolean;
}) {
  const { t } = useTranslation();
  const publicList = useRepAnnouncements(repId);
  const manage = useRepAnnouncementsManage(repId, couldOwn);
  const isOwner = manage.isSuccess;

  // The owner reads their own list; everyone else reads the public one.
  const items = isOwner ? (manage.data ?? []) : (publicList.data ?? []);

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h2 className="flex items-center gap-2 font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
          <Megaphone className="h-4 w-4 text-civic-700 dark:text-civic-300" aria-hidden="true" />
          {t('repAnnouncements.heading')}
        </h2>
        {isOwner && <NewAnnouncement repId={repId} />}
      </div>

      {isOwner && (
        <p className="text-sm text-slate-600 dark:text-slate-300">
          {t('repAnnouncements.ownerHint')}
        </p>
      )}

      {publicList.isLoading && !isOwner && (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      )}

      {items.length === 0 && !publicList.isLoading && (
        <EmptyState
          icon={<Megaphone className="h-6 w-6" aria-hidden="true" />}
          title={t(isOwner ? 'repAnnouncements.emptyOwnerTitle' : 'repAnnouncements.emptyTitle', {
            name: repName,
          })}
          body={t(isOwner ? 'repAnnouncements.emptyOwnerBody' : 'repAnnouncements.emptyBody')}
        />
      )}

      <ol className="space-y-3">
        {items.map((a) => (
          <AnnouncementCard key={a.id} a={a} repId={repId} isOwner={isOwner} />
        ))}
      </ol>
    </section>
  );
}

function AnnouncementCard({
  a,
  repId,
  isOwner,
}: {
  a: RepAnnouncement;
  repId: string;
  isOwner: boolean;
}) {
  const { t } = useTranslation();
  const relative = useRelativeTime();
  const [editing, setEditing] = useState(false);
  const [showThread, setShowThread] = useState(false);
  const publish = usePublishRepAnnouncement(repId);
  const archive = useArchiveRepAnnouncement(repId);
  const remove = useDeleteRepAnnouncement(repId);

  const tone =
    a.status === 'PUBLISHED'
      ? 'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200'
      : a.status === 'DRAFT'
        ? 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300'
        : 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300';

  return (
    <li className="rounded-2xl border border-slate-200 dark:border-slate-700 p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <h3 className="font-semibold text-slate-900 dark:text-slate-100">{a.title}</h3>
        {/* Status is shown to the owner only. A visitor sees nothing but
            published posts, so a badge would be noise. */}
        {isOwner && (
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase ${tone}`}>
            {t(`repAnnouncements.status.${a.status}`)}
          </span>
        )}
      </div>

      <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-200">
        {a.body}
      </p>

      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
        {a.publishedAt ? relative(a.publishedAt) : t('repAnnouncements.notPublished')}
      </p>

      {isOwner && (
        <div className="mt-3 flex flex-wrap gap-2">
          {a.status === 'DRAFT' && (
            <>
              {/* Publishing notifies every follower, so it says so on the
                  button rather than in a tooltip nobody opens. */}
              <Button size="sm" onClick={() => publish.mutate(a.id)} loading={publish.isPending}>
                {t('repAnnouncements.publish')}
              </Button>
              <Button size="sm" variant="secondary" onClick={() => setEditing(true)}>
                <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                {t('common.edit')}
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => remove.mutate(a.id)}
                loading={remove.isPending}
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                {t('common.delete')}
              </Button>
            </>
          )}
          {a.status === 'PUBLISHED' && (
            <Button
              size="sm"
              variant="secondary"
              onClick={() => archive.mutate(a.id)}
              loading={archive.isPending}
            >
              {t('repAnnouncements.archive')}
            </Button>
          )}
        </div>
      )}

      {editing && <EditForm repId={repId} announcement={a} onDone={() => setEditing(false)} />}

      {/* The thread only exists once it is public. A draft has never been
          seen, and an archived one has been withdrawn — reopening replies
          under a retracted statement would put words beneath something the
          representative no longer stands behind. */}
      {a.status === 'PUBLISHED' && (
        <div className="mt-4 border-t border-slate-200 dark:border-slate-700 pt-3">
          <button
            type="button"
            onClick={() => setShowThread((v) => !v)}
            className="text-sm font-semibold text-civic-700 hover:underline dark:text-civic-200"
          >
            {showThread
              ? t('repAnnouncements.hideReplies')
              : t('repAnnouncements.showReplies', { count: a.commentCount })}
          </button>
          {showThread && (
            <div className="mt-2">
              <CommentsSection
                entityType="repAnnouncements"
                entityId={a.id}
                basePath={`/api/v1/representatives/${repId}/announcements/${a.id}/comments`}
              />
            </div>
          )}
          {!isOwner && (
            <div className="mt-2">
              <ReportButton contentType="REPRESENTATIVE_ANNOUNCEMENT" contentId={a.id} />
            </div>
          )}
        </div>
      )}
    </li>
  );
}

function NewAnnouncement({ repId }: { repId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  if (!open) {
    return (
      <Button size="sm" onClick={() => setOpen(true)}>
        {t('repAnnouncements.new')}
      </Button>
    );
  }
  return <ComposeForm repId={repId} onDone={() => setOpen(false)} />;
}

function ComposeForm({ repId, onDone }: { repId: string; onDone: () => void }) {
  const { t } = useTranslation();
  const create = useCreateRepAnnouncement(repId);
  return (
    <AnnouncementForm
      submitLabel={t('repAnnouncements.saveDraft')}
      pending={create.isPending}
      onSubmit={async (title, body) => {
        await create.mutateAsync({ title, body });
        onDone();
      }}
      onCancel={onDone}
    />
  );
}

function EditForm({
  repId,
  announcement,
  onDone,
}: {
  repId: string;
  announcement: RepAnnouncement;
  onDone: () => void;
}) {
  const { t } = useTranslation();
  const update = useUpdateRepAnnouncement(repId);
  return (
    <AnnouncementForm
      initialTitle={announcement.title}
      initialBody={announcement.body}
      submitLabel={t('common.save')}
      pending={update.isPending}
      onSubmit={async (title, body) => {
        await update.mutateAsync({ id: announcement.id, title, body });
        onDone();
      }}
      onCancel={onDone}
    />
  );
}

function AnnouncementForm({
  initialTitle = '',
  initialBody = '',
  submitLabel,
  pending,
  onSubmit,
  onCancel,
}: {
  initialTitle?: string;
  initialBody?: string;
  submitLabel: string;
  pending: boolean;
  onSubmit: (title: string, body: string) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const [title, setTitle] = useState(initialTitle);
  const [body, setBody] = useState(initialBody);
  const [error, setError] = useState('');
  const canSubmit = title.trim().length >= 4 && body.trim().length >= 20 && !pending;

  async function handle(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await onSubmit(title.trim(), body.trim());
    } catch (err) {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? t('repAnnouncements.saveError'));
    }
  }

  const field =
    'w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-slate-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500';

  return (
    <form
      onSubmit={handle}
      className="mt-3 space-y-2 rounded-xl bg-slate-50 dark:bg-slate-800/60 p-3"
    >
      <input
        name="repAnnouncementTitle"
        className={field}
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={t('repAnnouncements.titlePlaceholder')}
        maxLength={200}
      />
      <textarea
        name="repAnnouncementBody"
        rows={4}
        className={field}
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder={t('repAnnouncements.bodyPlaceholder')}
      />
      {/* Saving never publishes. Said here so nobody discovers it by
          accidentally notifying a whole constituency. */}
      <p className="text-xs text-slate-500 dark:text-slate-400">
        {t('repAnnouncements.draftNote')}
      </p>
      {error && (
        <p className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-500/10 p-2 text-sm text-red-900 dark:text-red-100">
          {error}
        </p>
      )}
      <div className="flex justify-end gap-2">
        <Button type="button" size="sm" variant="secondary" onClick={onCancel}>
          {t('common.cancel')}
        </Button>
        <Button type="submit" size="sm" disabled={!canSubmit} loading={pending}>
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}
