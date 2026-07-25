import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Sparkles } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import { PageHeader } from '../components/PageHeader';
import { DraftWithAIPanel } from '../components/civic/DraftWithAIPanel';
import { getApiError } from '../lib/api';
import { useCreateAnnouncement } from '../hooks/useAnnouncements';
import { useMyOrganizations } from '../hooks/useConsultations';

export function OrgAnnouncementCreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId } = useParams<{ orgId: string }>();
  const createMutation = useCreateAnnouncement(orgId);

  // Read the caller's org membership to feed the AI draft prompt (name +
  // kind sharpen tone / audience calibration). Not load-bearing — if the
  // fetch is still pending or missing, drafting still works with less
  // context.
  const { data: memberships = [] } = useMyOrganizations();
  const org = memberships.find((m) => m.organization.id === orgId)?.organization;

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  // Track when the current title + body were populated by CivicAI so the
  // form can display an "AI-generated · review" affordance until the human
  // makes an edit. Clear as soon as the field is manually touched.
  const [titleFromAI, setTitleFromAI] = useState(false);
  const [bodyFromAI, setBodyFromAI] = useState(false);
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
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('orgAnnouncementCreate.back')}
      </Link>

      <PageHeader
        eyebrow={t('orgAnnouncementCreate.eyebrow')}
        title={t('orgAnnouncementCreate.title')}
        subtitle={t('orgAnnouncementCreate.subtitle')}
      />

      <DraftWithAIPanel
        orgName={org?.name}
        orgKind={org?.kind}
        onApply={(draft) => {
          setTitle(draft.title);
          setBody(draft.body);
          setTitleFromAI(true);
          setBodyFromAI(true);
        }}
      />

      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <div className="flex items-center justify-between">
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgAnnouncementCreate.fields.title')}
            </label>
            {titleFromAI && <AIFieldBadge />}
          </div>
          <Input
            value={title}
            onChange={(e) => {
              setTitle(e.target.value);
              setTitleFromAI(false);
            }}
            required
            minLength={2}
          />
        </div>

        <div>
          <div className="flex items-center justify-between">
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300">
              {t('orgAnnouncementCreate.fields.body')}
            </label>
            {bodyFromAI && <AIFieldBadge />}
          </div>
          <textarea
            value={body}
            onChange={(e) => {
              setBody(e.target.value);
              setBodyFromAI(false);
            }}
            required
            minLength={10}
            rows={10}
            className="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-300">
            {t('orgAnnouncementCreate.fields.bodyHelp')}
          </p>
        </div>

        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
          <input
            type="checkbox"
            checked={publishNow}
            onChange={(e) => setPublishNow(e.target.checked)}
            className="rounded text-civic-600 dark:text-civic-300 focus:ring-civic-500"
          />
          {t('orgAnnouncementCreate.publishNow')}
        </label>

        {error && (
          <p className="rounded-lg border border-red-200 dark:border-red-500/40 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-700 dark:text-red-300">
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

// AIFieldBadge marks a form field that still contains unedited AI output.
// Clears the moment the human edits the field — keeps the promise that
// "you'll always know when text came from CivicAI" while staying out of
// the way once the admin has made it their own.
function AIFieldBadge() {
  const { t } = useTranslation();
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-civic-100 dark:bg-civic-500/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-civic-700 dark:text-civic-200">
      <Sparkles className="h-3 w-3" aria-hidden="true" />
      {t('civicai.draft.fieldBadge')}
    </span>
  );
}
