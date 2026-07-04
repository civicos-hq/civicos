import { useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Megaphone, Briefcase, Inbox, Mail, Phone, Globe, ShieldCheck } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import {
  UserRole,
  AnnouncementStatus,
  AssignmentStatus,
  type ApiResponse,
  type Announcement,
  type IssueAssignment,
  type Organization,
  type Project,
  type ProgressUpdate,
} from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useRelativeTime } from '../hooks/useRelativeTime';
import { Modal } from '../components/Modal';
import { EmptyState } from '../components/EmptyState';
import { ReportButton } from '../components/civic/ReportButton';

const PLATFORM_ADMIN_ROLE: UserRole = UserRole.PLATFORM_ADMIN;

function useOrganization(id: string) {
  return useQuery({
    queryKey: ['organization', id],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ organization: Organization }>>(
        `/api/v1/organizations/${id}`,
      );
      return res.data.data.organization;
    },
    enabled: Boolean(id),
  });
}

function useAnnouncements(orgId: string) {
  return useQuery({
    queryKey: ['organization-announcements', orgId],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcements: Announcement[] }>>(
        `/api/v1/organizations/${orgId}/announcements`,
      );
      return res.data.data.announcements;
    },
    enabled: Boolean(orgId),
  });
}

function useProjects(orgId: string) {
  return useQuery({
    queryKey: ['organization-projects', orgId],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ projects: Project[] }>>(
        `/api/v1/organizations/${orgId}/projects`,
      );
      return res.data.data.projects;
    },
    enabled: Boolean(orgId),
  });
}

function useAssignments(orgId: string, enabled: boolean) {
  return useQuery({
    queryKey: ['organization-assignments', orgId],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ assignments: IssueAssignment[] }>>(
        `/api/v1/organizations/${orgId}/assignments`,
      );
      return res.data.data.assignments;
    },
    enabled: enabled && Boolean(orgId),
  });
}

export function OrganizationDetailPage() {
  const { t } = useTranslation();
  const { id = '' } = useParams<{ id: string }>();
  const orgQuery = useOrganization(id);
  const meQuery = useMe();
  const org = orgQuery.data;

  // A caller who is either PLATFORM_ADMIN OR a member of this org sees the
  // admin panel. Member check is done implicitly by the backend on the
  // assignments query — if it 403s, we hide the panel.
  const isPlatformAdmin = meQuery.data?.role === PLATFORM_ADMIN_ROLE;
  const asgQuery = useAssignments(id, Boolean(meQuery.data));
  const isMember = isPlatformAdmin || asgQuery.isSuccess;

  const annQuery = useAnnouncements(id);
  const projQuery = useProjects(id);

  if (orgQuery.isLoading || !org) {
    return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  }
  if (orgQuery.isError) {
    return (
      <section className="space-y-4">
        <Link to="/organizations" className="text-sm font-semibold text-civic-700 hover:underline">
          {t('organizationDetail.backToList')}
        </Link>
        <p className="text-sm text-red-600">{t('organizationDetail.loadError')}</p>
      </section>
    );
  }

  return (
    <section className="space-y-6">
      <Link to="/organizations" className="text-sm font-semibold text-civic-700 hover:underline">
        {t('organizationDetail.backToList')}
      </Link>

      <OrgHeader org={org} />

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">
            <Megaphone className="mr-2 inline h-4 w-4 text-civic-700" aria-hidden="true" />
            {t('organizationDetail.sections.announcements')}
          </h2>
          {isMember && <NewAnnouncementButton orgId={id} />}
        </div>
        <AnnouncementList
          items={annQuery.data ?? []}
          isLoading={annQuery.isLoading}
          isMember={isMember}
        />
      </section>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">
            <Briefcase className="mr-2 inline h-4 w-4 text-civic-700" aria-hidden="true" />
            {t('organizationDetail.sections.projects')}
          </h2>
          {isMember && <NewProjectButton orgId={id} />}
        </div>
        <ProjectList items={projQuery.data ?? []} isLoading={projQuery.isLoading} />
      </section>

      {isMember && (
        <section className="space-y-3">
          <h2 className="text-lg font-semibold text-slate-900">
            <Inbox className="mr-2 inline h-4 w-4 text-civic-700" aria-hidden="true" />
            {t('organizationDetail.sections.assignments')}
          </h2>
          <AssignmentList items={asgQuery.data ?? []} isLoading={asgQuery.isLoading} orgId={id} />
        </section>
      )}
    </section>
  );
}

function OrgHeader({ org }: { org: Organization }) {
  const { t } = useTranslation();
  return (
    <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <div className="flex flex-wrap items-start gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h1 className="text-3xl font-semibold text-slate-900">{org.name}</h1>
            {org.verified && (
              <ShieldCheck
                className="h-5 w-5 text-emerald-600"
                aria-label={t('organizationsPage.card.verified')}
              />
            )}
          </div>
          <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700">
            {t(`organizationsPage.kinds.${org.kind}`)} ·{' '}
            {t(`organizationsPage.jurisdictions.${org.jurisdiction}`)}
            {org.state ? ` · ${org.state}` : ''}
            {org.lga ? ` / ${org.lga}` : ''}
          </p>
          {org.description && (
            <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{org.description}</p>
          )}
          <div className="mt-4 flex flex-wrap gap-2">
            {org.email && (
              <ContactChip
                href={`mailto:${org.email}`}
                icon={<Mail className="h-4 w-4" />}
                label={org.email}
              />
            )}
            {org.phone && (
              <ContactChip
                href={`tel:${org.phone.replace(/\s+/g, '')}`}
                icon={<Phone className="h-4 w-4" />}
                label={org.phone}
              />
            )}
            {org.website && (
              <ContactChip
                href={org.website}
                icon={<Globe className="h-4 w-4" />}
                label={org.website.replace(/^https?:\/\//, '')}
                external
              />
            )}
          </div>
        </div>
      </div>
    </header>
  );
}

function ContactChip({
  href,
  icon,
  label,
  external,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  external?: boolean;
}) {
  return (
    <a
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
      className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-700 hover:border-civic-300 hover:bg-civic-50 hover:text-civic-700"
    >
      <span className="text-slate-400">{icon}</span>
      {label}
    </a>
  );
}

function AnnouncementList({
  items,
  isLoading,
  isMember,
}: {
  items: Announcement[];
  isLoading: boolean;
  isMember: boolean;
}) {
  const { t } = useTranslation();
  const relative = useRelativeTime();
  if (isLoading) return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Megaphone className="h-5 w-5" />}
        title={t('organizationDetail.emptyAnnouncements')}
      />
    );
  }
  return (
    <div className="space-y-3">
      {items.map((a) => (
        <article
          key={a.id}
          className={`rounded-xl border p-4 shadow-sm ${
            a.status === AnnouncementStatus.PUBLISHED
              ? 'border-slate-200 bg-white'
              : 'border-amber-300 bg-amber-50/60'
          }`}
        >
          <div className="flex items-baseline justify-between gap-2">
            <h3 className="text-base font-semibold text-slate-900">{a.title}</h3>
            <span className="text-xs text-slate-500">
              {a.publishedAt ? relative(a.publishedAt) : relative(a.createdAt)}
            </span>
          </div>
          {isMember && a.status !== AnnouncementStatus.PUBLISHED && (
            <p className="mt-1 text-[11px] font-semibold uppercase tracking-wider text-amber-700">
              {t(`organizationDetail.announcementStatus.${a.status}`)}
            </p>
          )}
          <p className="mt-2 whitespace-pre-wrap text-sm text-slate-700">{a.body}</p>
          <div className="mt-3 flex items-center justify-between gap-2">
            <p className="text-xs text-slate-500">
              {t('organizationDetail.byAuthor', { name: a.authorName })}
            </p>
            <ReportButton contentType="ANNOUNCEMENT" contentId={a.id} />
          </div>
        </article>
      ))}
    </div>
  );
}

function NewAnnouncementButton({ orgId }: { orgId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button size="sm" variant="secondary" onClick={() => setOpen(true)}>
        {t('organizationDetail.newAnnouncement')}
      </Button>
      {open && <NewAnnouncementModal orgId={orgId} onClose={() => setOpen(false)} />}
    </>
  );
}

function NewAnnouncementModal({ orgId, onClose }: { orgId: string; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [publish, setPublish] = useState(true);
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/organizations/${orgId}/announcements`, {
        title,
        body,
        publish,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-announcements', orgId] });
      queryClient.invalidateQueries({ queryKey: ['organization', orgId] });
      onClose();
    },
    onError: () => setError(t('organizationDetail.genericError')),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('organizationDetail.announcementModal.title')} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          label={t('organizationDetail.announcementModal.fields.title')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          minLength={2}
        />
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="ann-body">
            {t('organizationDetail.announcementModal.fields.body')}
          </label>
          <textarea
            id="ann-body"
            rows={6}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            required
            minLength={10}
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
          />
        </div>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={publish} onChange={(e) => setPublish(e.target.checked)} />
          {t('organizationDetail.announcementModal.publishNow')}
        </label>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('common.save')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function ProjectList({ items, isLoading }: { items: Project[]; isLoading: boolean }) {
  const { t } = useTranslation();
  if (isLoading) return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Briefcase className="h-5 w-5" />}
        title={t('organizationDetail.emptyProjects')}
      />
    );
  }
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {items.map((p) => (
        <article key={p.id} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between gap-2">
            <h3 className="text-base font-semibold text-slate-900">{p.title}</h3>
            <span className="rounded-full bg-civic-100 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-civic-700">
              {t(`organizationDetail.projectStatus.${p.status}`)}
            </span>
          </div>
          <p className="mt-2 line-clamp-3 text-sm text-slate-600">{p.description}</p>
          {typeof p.budgetKobo === 'number' && (
            <p className="mt-2 text-xs text-slate-500">
              {t('organizationDetail.projectBudget', {
                amount: new Intl.NumberFormat('en-NG', {
                  style: 'currency',
                  currency: 'NGN',
                  maximumFractionDigits: 0,
                }).format(p.budgetKobo / 100),
              })}
            </p>
          )}
        </article>
      ))}
    </div>
  );
}

function NewProjectButton({ orgId }: { orgId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button size="sm" variant="secondary" onClick={() => setOpen(true)}>
        {t('organizationDetail.newProject')}
      </Button>
      {open && <NewProjectModal orgId={orgId} onClose={() => setOpen(false)} />}
    </>
  );
}

function NewProjectModal({ orgId, onClose }: { orgId: string; onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [budgetNGN, setBudgetNGN] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: async () => {
      const payload: Record<string, string | number> = { title, description };
      const budget = Number(budgetNGN);
      if (!Number.isNaN(budget) && budget > 0) {
        payload.budgetKobo = Math.round(budget * 100);
      }
      await api.post(`/api/v1/organizations/${orgId}/projects`, payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-projects', orgId] });
      queryClient.invalidateQueries({ queryKey: ['organization', orgId] });
      onClose();
    },
    onError: () => setError(t('organizationDetail.genericError')),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('organizationDetail.projectModal.title')} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          label={t('organizationDetail.projectModal.fields.title')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          minLength={2}
        />
        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="proj-desc">
            {t('organizationDetail.projectModal.fields.description')}
          </label>
          <textarea
            id="proj-desc"
            rows={5}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            minLength={10}
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
          />
        </div>
        <Input
          label={t('organizationDetail.projectModal.fields.budgetNGN')}
          type="number"
          min={0}
          step={1000}
          value={budgetNGN}
          onChange={(e) => setBudgetNGN(e.target.value)}
        />
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('common.save')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function AssignmentList({
  items,
  isLoading,
  orgId,
}: {
  items: IssueAssignment[];
  isLoading: boolean;
  orgId: string;
}) {
  const { t } = useTranslation();
  const relative = useRelativeTime();
  if (isLoading) return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Inbox className="h-5 w-5" />}
        title={t('organizationDetail.emptyAssignments')}
      />
    );
  }
  return (
    <div className="space-y-3">
      {items.map((a) => (
        <AssignmentRow key={a.id} assignment={a} orgId={orgId} relative={relative} />
      ))}
    </div>
  );
}

function AssignmentRow({
  assignment,
  orgId,
  relative,
}: {
  assignment: IssueAssignment;
  orgId: string;
  relative: (iso: string) => string;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [responseBody, setResponseBody] = useState('');
  const [showResponse, setShowResponse] = useState(false);

  const statusMutation = useMutation({
    mutationFn: async (status: AssignmentStatus) => {
      await api.patch(`/api/v1/assignments/${assignment.id}`, { status });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-assignments', orgId] });
    },
  });

  const respondMutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/organizations/${orgId}/progress-updates`, {
        issueId: assignment.issueId,
        body: responseBody,
        isPublic: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['issue-progress', assignment.issueId] });
      setResponseBody('');
      setShowResponse(false);
    },
  });

  return (
    <article className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-baseline justify-between gap-2">
        <p className="text-sm font-semibold text-slate-900">
          <Link to={`/issues/${assignment.issueId}`} className="text-civic-700 hover:underline">
            {t('organizationDetail.assignmentIssueLink', {
              id: assignment.issueId.slice(0, 8),
            })}
          </Link>
        </p>
        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-slate-700">
          {t(`organizationDetail.assignmentStatus.${assignment.status}`)}
        </span>
      </div>
      {assignment.note && <p className="mt-2 text-sm text-slate-600">{assignment.note}</p>}
      <p className="mt-2 text-xs text-slate-500">
        {t('organizationDetail.assignmentAssignedBy', {
          name: assignment.assignedByName,
          when: relative(assignment.createdAt),
        })}
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        {assignment.status !== AssignmentStatus.IN_PROGRESS && (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => statusMutation.mutate(AssignmentStatus.IN_PROGRESS)}
            loading={statusMutation.isPending}
          >
            {t('organizationDetail.markInProgress')}
          </Button>
        )}
        {assignment.status !== AssignmentStatus.COMPLETED && (
          <Button
            size="sm"
            variant="secondary"
            onClick={() => statusMutation.mutate(AssignmentStatus.COMPLETED)}
            loading={statusMutation.isPending}
          >
            {t('organizationDetail.markCompleted')}
          </Button>
        )}
        <Button size="sm" onClick={() => setShowResponse((v) => !v)}>
          {t('organizationDetail.postResponse')}
        </Button>
      </div>

      {showResponse && (
        <form
          className="mt-3 space-y-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (responseBody.trim().length >= 2) respondMutation.mutate();
          }}
        >
          <textarea
            rows={3}
            value={responseBody}
            onChange={(e) => setResponseBody(e.target.value)}
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            placeholder={t('organizationDetail.responsePlaceholder')}
          />
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              size="sm"
              variant="secondary"
              onClick={() => setShowResponse(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="submit"
              size="sm"
              loading={respondMutation.isPending}
              disabled={responseBody.trim().length < 2}
            >
              {t('organizationDetail.postResponse')}
            </Button>
          </div>
        </form>
      )}
    </article>
  );
}

export function useIssueProgressUpdates(issueId: string) {
  return useQuery({
    queryKey: ['issue-progress', issueId],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ updates: ProgressUpdate[] }>>(
        `/api/v1/issues/${issueId}/progress-updates`,
      );
      return res.data.data.updates;
    },
    enabled: Boolean(issueId),
  });
}
