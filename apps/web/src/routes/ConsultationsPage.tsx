import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ConsultationStatus, type Consultation } from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { uploadUrl } from '../lib/api';
import { useConsultations, useMyConsultationResponses } from '../hooks/useConsultations';
import { MessageSquare } from 'lucide-react';

const STATUS_TONE: Record<ConsultationStatus, string> = {
  [ConsultationStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [ConsultationStatus.PUBLISHED]: 'bg-civic-100 text-civic-700',
  [ConsultationStatus.CLOSED]: 'bg-amber-100 text-amber-700',
};

// Only three filter chips: All / Open (PUBLISHED) / Closed. Draft is
// admin-facing and hidden from the citizen list — the server still
// returns them, but showing them here would just clutter.
const FILTER_STATUS: Array<{ value: '' | ConsultationStatus; i18nKey: string }> = [
  { value: '', i18nKey: 'consultationsPage.filters.all' },
  { value: ConsultationStatus.PUBLISHED, i18nKey: 'consultationsPage.filters.open' },
  { value: ConsultationStatus.CLOSED, i18nKey: 'consultationsPage.filters.closed' },
];

function formatDate(iso?: string) {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function ConsultationsPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const [params, setParams] = useSearchParams();
  const statusFilter = (params.get('status') as ConsultationStatus | null) ?? '';
  const respondedQuery = useMyConsultationResponses();
  // Default to PUBLISHED when there's no filter — citizens land on "what
  // can I actually respond to right now" instead of a mixed list.
  const listQuery = useConsultations({
    status: statusFilter || ConsultationStatus.PUBLISHED,
  });

  function setFilter(value: '' | ConsultationStatus) {
    const next = new URLSearchParams(params);
    if (!value) next.delete('status');
    else next.set('status', value);
    setParams(next, { replace: true });
  }

  // Filter out drafts client-side just in case the server sends them
  // (belt-and-braces — the current server surface returns everything).
  const items: Consultation[] = (listQuery.data ?? []).filter(
    (c) => c.status !== ConsultationStatus.DRAFT,
  );

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('consultationsPage.eyebrow')}
        title={t('consultationsPage.title')}
        subtitle={t('consultationsPage.subtitle')}
        meta={meta}
      />

      <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex flex-wrap gap-2">
          {FILTER_STATUS.map((f) => (
            <button
              key={f.value || 'all'}
              type="button"
              onClick={() => setFilter(f.value)}
              className={
                'rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-wide transition ' +
                (statusFilter === f.value ||
                (!statusFilter && f.value === ConsultationStatus.PUBLISHED)
                  ? 'border-civic-500 bg-civic-50 text-civic-700'
                  : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300')
              }
            >
              {t(f.i18nKey)}
            </button>
          ))}
        </div>
      </section>

      {listQuery.isLoading && <p className="text-sm text-slate-600">{t('common.loading')}</p>}

      {listQuery.isError && (
        <p className="text-sm text-red-600">{t('consultationsPage.loadError')}</p>
      )}

      {!listQuery.isLoading && items.length === 0 && (
        <EmptyState
          icon={<MessageSquare size={20} />}
          title={t('consultationsPage.empty.title')}
          body={t('consultationsPage.empty.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((c) => {
          const responded = respondedQuery.data?.has(c.id) ?? false;
          return (
            <li key={c.id}>
              <Link
                to={`/consultations/${c.id}`}
                className="block overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm transition hover:border-civic-300 hover:shadow-md"
              >
                {c.coverImageUrl && (
                  <img
                    src={uploadUrl(c.coverImageUrl)}
                    alt=""
                    className="h-32 w-full object-cover"
                    loading="lazy"
                  />
                )}
                <div className="p-5">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <h2 className="font-fraunces text-lg font-semibold text-slate-900">
                        {c.title}
                      </h2>
                      <p className="mt-1 text-sm text-slate-600">{c.summary}</p>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <span
                        className={
                          'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
                          STATUS_TONE[c.status]
                        }
                      >
                        {t(`consultationsPage.status.${c.status}`)}
                      </span>
                      {responded && (
                        <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold uppercase tracking-wide text-emerald-700">
                          {t('consultationsPage.youResponded')}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                    <span>
                      {t('consultationsPage.by')}{' '}
                      <span className="font-semibold text-slate-700">{c.authorName}</span>
                    </span>
                    <span>
                      {c.responseCount} {t('consultationsPage.responses')}
                    </span>
                    {c.closesAt && (
                      <span>
                        {c.status === ConsultationStatus.CLOSED
                          ? t('consultationsPage.closedOn', { date: formatDate(c.closedAt) })
                          : t('consultationsPage.closesOn', { date: formatDate(c.closesAt) })}
                      </span>
                    )}
                  </div>
                </div>
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
