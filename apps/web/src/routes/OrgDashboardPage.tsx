import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import {
  AnnouncementStatus,
  AssignmentStatus,
  ConsultationStatus,
  ProjectStatus,
  type Announcement,
  type Consultation,
  type IssueAssignment,
  type Project,
} from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useConsultations, useMyOrganizations } from '../hooks/useConsultations';
import { useOrgAnnouncements } from '../hooks/useAnnouncements';
import { useOrgProjects, kobopToNaira } from '../hooks/useProjects';
import {
  useDeleteAssignment,
  useOrgAssignments,
  useUpdateAssignmentStatus,
} from '../hooks/useAssignments';
import { MessageSquare, Megaphone, Briefcase, Inbox } from 'lucide-react';

// Tab type mirrors the sub-sections on the dashboard. Adding a tab is
// low-effort: add to this union, add a case in the switch, add an i18n key.
type Tab = 'consultations' | 'announcements' | 'projects' | 'assignments';

const TABS: Array<{ id: Tab; i18n: string }> = [
  { id: 'consultations', i18n: 'orgDashboard.tabs.consultations' },
  { id: 'announcements', i18n: 'orgDashboard.tabs.announcements' },
  { id: 'projects', i18n: 'orgDashboard.tabs.projects' },
  { id: 'assignments', i18n: 'orgDashboard.tabs.assignments' },
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

const PROJECT_TONE: Record<ProjectStatus, string> = {
  [ProjectStatus.PLANNED]: 'bg-slate-200 text-slate-700',
  [ProjectStatus.ACTIVE]: 'bg-civic-100 text-civic-700',
  [ProjectStatus.PAUSED]: 'bg-amber-100 text-amber-700',
  [ProjectStatus.COMPLETED]: 'bg-emerald-100 text-emerald-700',
  [ProjectStatus.CANCELLED]: 'bg-slate-300 text-slate-700',
};

const ASSIGNMENT_TONE: Record<AssignmentStatus, string> = {
  [AssignmentStatus.RECEIVED]: 'bg-slate-200 text-slate-700',
  [AssignmentStatus.IN_PROGRESS]: 'bg-civic-100 text-civic-700',
  [AssignmentStatus.COMPLETED]: 'bg-emerald-100 text-emerald-700',
  [AssignmentStatus.REJECTED]: 'bg-rose-100 text-rose-700',
};

const ASSIGNMENT_STATUSES: AssignmentStatus[] = [
  AssignmentStatus.RECEIVED,
  AssignmentStatus.IN_PROGRESS,
  AssignmentStatus.COMPLETED,
  AssignmentStatus.REJECTED,
];

function formatNaira(kobo?: number): string {
  const n = kobopToNaira(kobo);
  if (n === '') return '';
  return `₦${n.toLocaleString()}`;
}

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
      {activeTab === 'projects' && <ProjectsSection orgId={orgId} canAdmin={canAdmin} />}
      {activeTab === 'assignments' && <AssignmentsSection orgId={orgId} canAdmin={canAdmin} />}
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

function ProjectsSection({ orgId, canAdmin }: { orgId: string | undefined; canAdmin: boolean }) {
  const { t } = useTranslation();
  const query = useOrgProjects(orgId);
  const items = ((query.data ?? []) as Project[]).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        {canAdmin && (
          <Link to={`/org/${orgId}/projects/new`}>
            <Button size="sm">{t('orgDashboard.newProject')}</Button>
          </Link>
        )}
      </div>

      {query.isLoading && <p className="text-sm text-slate-600">{t('common.loading')}</p>}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<Briefcase size={20} />}
          title={t('orgDashboard.emptyProjects.title')}
          body={t('orgDashboard.emptyProjects.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((p) => (
          <li key={p.id}>
            <Link
              to={`/org/${orgId}/projects/${p.id}`}
              className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300 hover:shadow-md"
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <h2 className="font-fraunces text-lg font-semibold text-slate-900">{p.title}</h2>
                  <p className="mt-1 line-clamp-2 text-sm text-slate-600">{p.description}</p>
                </div>
                <span
                  className={
                    'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
                    PROJECT_TONE[p.status]
                  }
                >
                  {t(`orgDashboard.projectStatus.${p.status}`)}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                {p.budgetKobo !== null && p.budgetKobo !== undefined && (
                  <span>{formatNaira(p.budgetKobo)}</span>
                )}
                {p.startDate && (
                  <span>
                    {t('orgDashboard.startsOn', {
                      date: new Date(p.startDate).toLocaleDateString(),
                    })}
                  </span>
                )}
                {p.expectedEndDate && (
                  <span>
                    {t('orgDashboard.endsOn', {
                      date: new Date(p.expectedEndDate).toLocaleDateString(),
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

function AssignmentsSection({ orgId, canAdmin }: { orgId: string | undefined; canAdmin: boolean }) {
  const { t } = useTranslation();
  const query = useOrgAssignments(orgId);
  const items = ((query.data ?? []) as IssueAssignment[]).sort((a, b) =>
    a.createdAt < b.createdAt ? 1 : -1,
  );

  return (
    <div className="space-y-4">
      {query.isLoading && <p className="text-sm text-slate-600">{t('common.loading')}</p>}

      {!query.isLoading && items.length === 0 && (
        <EmptyState
          icon={<Inbox size={20} />}
          title={t('orgDashboard.emptyAssignments.title')}
          body={t('orgDashboard.emptyAssignments.body')}
        />
      )}

      <ul className="space-y-3">
        {items.map((a) => (
          <AssignmentRow key={a.id} orgId={orgId} assignment={a} canAdmin={canAdmin} />
        ))}
      </ul>
    </div>
  );
}

function AssignmentRow({
  orgId,
  assignment,
  canAdmin,
}: {
  orgId: string | undefined;
  assignment: IssueAssignment;
  canAdmin: boolean;
}) {
  const { t } = useTranslation();
  const updateStatus = useUpdateAssignmentStatus(assignment.id, orgId, assignment.issueId);
  const deleteMutation = useDeleteAssignment(assignment.id, orgId, assignment.issueId);

  return (
    <li className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <Link
            to={`/issues/${assignment.issueId}`}
            className="font-fraunces text-base font-semibold text-slate-900 hover:underline"
          >
            {t('orgDashboard.assignmentIssueLink', {
              id: assignment.issueId.slice(0, 8),
            })}
          </Link>
          {assignment.note && (
            <p className="mt-1 text-sm italic text-slate-600">&ldquo;{assignment.note}&rdquo;</p>
          )}
          <p className="mt-1 text-xs text-slate-500">
            {t('orgDashboard.assignmentAssignedBy', {
              name: assignment.assignedByName,
              date: new Date(assignment.createdAt).toLocaleDateString(),
            })}
          </p>
        </div>
        <span
          className={
            'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
            ASSIGNMENT_TONE[assignment.status]
          }
        >
          {t(`orgDashboard.assignmentStatus.${assignment.status}`)}
        </span>
      </div>

      {canAdmin && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <label className="text-xs font-semibold text-slate-600">
            {t('orgDashboard.assignmentSetStatus')}
          </label>
          <select
            value={assignment.status}
            onChange={(e) => updateStatus.mutate({ status: e.target.value as AssignmentStatus })}
            disabled={updateStatus.isPending}
            className="rounded-lg border border-slate-300 px-2 py-1 text-xs shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          >
            {ASSIGNMENT_STATUSES.map((s) => (
              <option key={s} value={s}>
                {t(`orgDashboard.assignmentStatus.${s}`)}
              </option>
            ))}
          </select>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              if (window.confirm(t('orgDashboard.assignmentConfirmDelete'))) {
                deleteMutation.mutate();
              }
            }}
            disabled={deleteMutation.isPending}
          >
            {t('orgDashboard.assignmentDrop')}
          </Button>
        </div>
      )}
    </li>
  );
}
