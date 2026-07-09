import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import type { CreateConsultationInput } from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { useCommunities } from '../hooks/useCommunities';
import { getApiError } from '../lib/api';
import { useCreateConsultation } from '../hooks/useConsultations';

export function OrgConsultationCreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId } = useParams<{ orgId: string }>();
  const communitiesQuery = useCommunities();
  const createMutation = useCreateConsultation(orgId);

  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [description, setDescription] = useState('');
  const [communityId, setCommunityId] = useState('');
  const [closesAt, setClosesAt] = useState('');
  const [error, setError] = useState('');

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    const input: CreateConsultationInput = {
      title: title.trim(),
      summary: summary.trim(),
      description,
      communityId: communityId || undefined,
      closesAt: closesAt ? new Date(closesAt).toISOString() : undefined,
    };
    try {
      const created = await createMutation.mutateAsync(input);
      // Land directly on the detail page so the user can add questions
      // — creation without follow-through is a dead-end otherwise.
      navigate(`/org/${orgId}/consultations/${created.id}`);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgConsultationCreate.error'));
    }
  }

  return (
    <section className="space-y-6">
      <Link to={`/org/${orgId}`} className="text-sm font-semibold text-civic-700 hover:underline">
        {t('orgConsultationCreate.back')}
      </Link>

      <PageHeader
        eyebrow={t('orgConsultationCreate.eyebrow')}
        title={t('orgConsultationCreate.title')}
        subtitle={t('orgConsultationCreate.subtitle')}
      />

      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationCreate.fields.title')}
          </label>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={200}
            required
            minLength={5}
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationCreate.fields.titleHelp')}
          </p>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationCreate.fields.summary')}
          </label>
          <Input
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            maxLength={500}
            required
            minLength={10}
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationCreate.fields.summaryHelp')}
          </p>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationCreate.fields.description')}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            minLength={10}
            rows={6}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationCreate.fields.descriptionHelp')}
          </p>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationCreate.fields.community')}
          </label>
          <select
            value={communityId}
            onChange={(e) => setCommunityId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          >
            <option value="">{t('orgConsultationCreate.fields.communityAny')}</option>
            {(communitiesQuery.data ?? []).map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.state} · {c.lga})
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationCreate.fields.communityHelp')}
          </p>
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationCreate.fields.closesAt')}
          </label>
          <input
            type="datetime-local"
            value={closesAt}
            onChange={(e) => setClosesAt(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationCreate.fields.closesAtHelp')}
          </p>
        </div>

        {error && (
          <p className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}

        <div className="flex justify-end gap-3">
          <Link to={`/org/${orgId}`}>
            <Button variant="ghost">{t('common.cancel')}</Button>
          </Link>
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending
              ? t('orgConsultationCreate.creating')
              : t('orgConsultationCreate.create')}
          </Button>
        </div>
      </form>
    </section>
  );
}
