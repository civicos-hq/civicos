import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { ConsultationStatus } from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useConsultations, useMyOrganizations } from '../hooks/useConsultations';
import { MessageSquare } from 'lucide-react';

const STATUS_TONE: Record<ConsultationStatus, string> = {
  [ConsultationStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [ConsultationStatus.PUBLISHED]: 'bg-civic-100 text-civic-700',
  [ConsultationStatus.CLOSED]: 'bg-amber-100 text-amber-700',
};

export function OrgDashboardPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const { orgId } = useParams<{ orgId: string }>();

  const { data: memberships = [] } = useMyOrganizations();
  const membership = memberships.find((m) => m.organization.id === orgId);
  // Fetch ALL statuses so DRAFTs are visible to org admins. The citizen
  // list defaulted to PUBLISHED — org admins need the full picture.
  const consultationsQuery = useConsultations({ organizationId: orgId });

  const consultations = (consultationsQuery.data ?? []).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  if (!membership) {
    return (
      <section className="space-y-4">
        <Link to="/org" className="text-sm font-semibold text-civic-700 hover:underline">
          {t('orgDashboard.back')}
        </Link>
        <p className="text-sm text-red-600">{t('orgDashboard.notMember')}</p>
      </section>
    );
  }

  const canAdmin = membership.membership.role === 'OWNER' || membership.membership.role === 'ADMIN';

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('orgDashboard.eyebrow', { org: membership.organization.name })}
        title={t('orgDashboard.title')}
        subtitle={t('orgDashboard.subtitle')}
        meta={meta}
        actions={
          canAdmin && (
            <Link to={`/org/${orgId}/consultations/new`}>
              <Button size="sm">{t('orgDashboard.newConsultation')}</Button>
            </Link>
          )
        }
      />

      {consultationsQuery.isLoading && (
        <p className="text-sm text-slate-600">{t('common.loading')}</p>
      )}

      {!consultationsQuery.isLoading && consultations.length === 0 && (
        <EmptyState
          icon={<MessageSquare size={20} />}
          title={t('orgDashboard.empty.title')}
          body={t('orgDashboard.empty.body')}
        />
      )}

      <ul className="space-y-3">
        {consultations.map((c) => (
          <li key={c.id}>
            <Link
              to={`/org/${orgId}/consultations/${c.id}`}
              className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300 hover:shadow-md"
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <h2 className="font-fraunces text-lg font-semibold text-slate-900">{c.title}</h2>
                  <p className="mt-1 text-sm text-slate-600">{c.summary}</p>
                </div>
                <span
                  className={
                    'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
                    STATUS_TONE[c.status]
                  }
                >
                  {t(`consultationsPage.status.${c.status}`)}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                <span>
                  {c.responseCount} {t('consultationsPage.responses')}
                </span>
                <span>{new Date(c.createdAt).toLocaleDateString()}</span>
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
