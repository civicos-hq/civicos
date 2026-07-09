import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import {
  AnnouncementStatus,
  ConsultationStatus,
  type Announcement,
  type Consultation,
} from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useConsultations, useMyOrganizations } from '../hooks/useConsultations';
import { useOrgAnnouncements } from '../hooks/useAnnouncements';
import { MessageSquare, Megaphone } from 'lucide-react';

// Tab type mirrors the sub-sections on the dashboard. Adding a tab is
// low-effort: add to this union, add a case in the switch, add an i18n key.
type Tab = 'consultations' | 'announcements';

const TABS: Array<{ id: Tab; i18n: string }> = [
  { id: 'consultations', i18n: 'orgDashboard.tabs.consultations' },
  { id: 'announcements', i18n: 'orgDashboard.tabs.announcements' },
];

const CONSULTATION_TONE: Record<ConsultationStatus, string> = {
  [ConsultationStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [ConsultationStatus.PUBLISHED]: 'bg-civic-100 text-civic-700',
  [ConsultationStatus.CLOSED]: 'bg-amber-100 text-amber-700',
};

const ANNOUNCEMENT_TONE: Record<AnnouncementStatus, string> = {
  [AnnouncementStatus.DRAFT]: 'bg-slate-200 text-slate-700',
  [AnnouncementStatus.PUBLISHED]: 'bg-civic-100 text-civic-700',
  [AnnouncementStatus.ARCHIVED]: 'bg-amber-100 text-amber-700',
};

export function OrgDashboardPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const { orgId } = useParams<{ orgId: string }>();
  const [params, setParams] = useSearchParams();
  const activeTab = (params.get('tab') as Tab | null) ?? 'consultations';

  const { data: memberships = [] } = useMyOrganizations();
  const membership = memberships.find((m) => m.organization.id === orgId);

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

  function switchTab(next: Tab) {
    const p = new URLSearchParams(params);
    p.set('tab', next);
    setParams(p, { replace: true });
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('orgDashboard.eyebrow', { org: membership.organization.name })}
        title={t('orgDashboard.title')}
        subtitle={t('orgDashboard.subtitle')}
        meta={meta}
      />

      <nav
        className="flex flex-wrap gap-2 border-b border-slate-200"
        aria-label={t('orgDashboard.tabsAria')}
      >
        {TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => switchTab(tab.id)}
            className={
              'border-b-2 px-3 py-2 text-sm font-semibold transition ' +
              (activeTab === tab.id
                ? 'border-civic-600 text-civic-700'
                : 'border-transparent text-slate-500 hover:text-slate-700')
            }
          >
            {t(tab.i18n)}
          </button>
        ))}
      </nav>

      {activeTab === 'consultations' && <ConsultationsSection orgId={orgId} canAdmin={canAdmin} />}
      {activeTab === 'announcements' && <AnnouncementsSection orgId={orgId} canAdmin={canAdmin} />}
    </section>
  );
}

function ConsultationsSection({
  orgId,
  canAdmin,
}: {
  orgId: string | undefined;
  canAdmin: boolean;
}) {
  const { t } = useTranslation();
  const query = useConsultations({ organizationId: orgId });
  const items = ((query.data ?? []) as Consultation[]).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        {canAdmin && (
          <Link to={`/org/${orgId}/consultations/new`}>
            <Button size="sm">{t('orgDashboard.newConsultation')}</Button>
          </Link>
        )}
      </div>

      {query.isLoading && <p className="text-sm text-slate-600">{t('common.loading')}</p>}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<MessageSquare size={20} />}
          title={t('orgDashboard.emptyConsultations.title')}
          body={t('orgDashboard.emptyConsultations.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((c) => (
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
                    CONSULTATION_TONE[c.status]
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
    </div>
  );
}

function AnnouncementsSection({
  orgId,
  canAdmin,
}: {
  orgId: string | undefined;
  canAdmin: boolean;
}) {
  const { t } = useTranslation();
  const query = useOrgAnnouncements(orgId);
  const items = ((query.data ?? []) as Announcement[]).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        {canAdmin && (
          <Link to={`/org/${orgId}/announcements/new`}>
            <Button size="sm">{t('orgDashboard.newAnnouncement')}</Button>
          </Link>
        )}
      </div>

      {query.isLoading && <p className="text-sm text-slate-600">{t('common.loading')}</p>}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<Megaphone size={20} />}
          title={t('orgDashboard.emptyAnnouncements.title')}
          body={t('orgDashboard.emptyAnnouncements.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((a) => (
          <li key={a.id}>
            <Link
              to={`/org/${orgId}/announcements/${a.id}`}
              className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300 hover:shadow-md"
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <h2 className="font-fraunces text-lg font-semibold text-slate-900">{a.title}</h2>
                  <p className="mt-1 line-clamp-2 text-sm text-slate-600">{a.body}</p>
                </div>
                <span
                  className={
                    'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
                    ANNOUNCEMENT_TONE[a.status]
                  }
                >
                  {t(`orgDashboard.announcementStatus.${a.status}`)}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                <span>{a.authorName}</span>
                <span>{new Date(a.createdAt).toLocaleDateString()}</span>
                {a.publishedAt && (
                  <span>
                    {t('orgDashboard.publishedOn', {
                      date: new Date(a.publishedAt).toLocaleDateString(),
                    })}
                  </span>
                )}
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
