import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { PageHeader } from '../components/PageHeader';
import { getApiError } from '../lib/api';
import { useCreateAnnouncement } from '../hooks/useAnnouncements';

export function OrgAnnouncementCreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId } = useParams<{ orgId: string }>();
  const createMutation = useCreateAnnouncement(orgId);

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  // `publish` is a compound choice: save-as-draft (default) vs publish
  // immediately. The API supports the second via a boolean on create.
  const [publishNow, setPublishNow] = useState(false);
  const [error, setError] = useState('');

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      const created = await createMutation.mutateAsync({
        title: title.trim(),
        body,
        publish: publishNow,
      });
      // Even on immediate-publish, land the user on the detail page so
      // they can share the URL or make follow-up edits (title/body still
      // editable while PUBLISHED, unlike consultation questions).
      navigate(`/org/${orgId}/announcements/${created.id}`);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgAnnouncementCreate.error'));
    }
  }

  return (
    <section className="space-y-6">
      <Link
        to={`/org/${orgId}?tab=announcements`}
        className="text-sm font-semibold text-civic-700 hover:underline"
      >
        {t('orgAnnouncementCreate.back')}
      </Link>

      <PageHeader
        eyebrow={t('orgAnnouncementCreate.eyebrow')}
        title={t('orgAnnouncementCreate.title')}
        subtitle={t('orgAnnouncementCreate.subtitle')}
      />

      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgAnnouncementCreate.fields.title')}
          </label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} required minLength={2} />
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgAnnouncementCreate.fields.body')}
          </label>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            required
            minLength={10}
            rows={10}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgAnnouncementCreate.fields.bodyHelp')}
          </p>
        </div>

        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={publishNow}
            onChange={(e) => setPublishNow(e.target.checked)}
            className="rounded text-civic-600 focus:ring-civic-500"
          />
          {t('orgAnnouncementCreate.publishNow')}
        </label>

        {error && (
          <p className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}

        <div className="flex justify-end gap-3">
          <Link to={`/org/${orgId}?tab=announcements`}>
            <Button variant="ghost">{t('common.cancel')}</Button>
          </Link>
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending
              ? t('common.saving')
              : publishNow
                ? t('orgAnnouncementCreate.saveAndPublish')
                : t('orgAnnouncementCreate.saveDraft')}
          </Button>
        </div>
      </form>
    </section>
  );
}
