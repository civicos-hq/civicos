import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Building2, ShieldCheck } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import {
  UserRole,
  OrgKind,
  OrgJurisdiction,
  type ApiResponse,
  type Organization,
} from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { Modal } from '../components/Modal';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';

const CREATOR_ROLES = new Set<UserRole>([
  UserRole.GOVERNMENT_ADMIN,
  UserRole.PLATFORM_ADMIN,
  UserRole.NGO,
]);

const KIND_OPTIONS: (OrgKind | 'ALL')[] = [
  'ALL',
  OrgKind.GOVERNMENT,
  OrgKind.AGENCY,
  OrgKind.NGO,
  OrgKind.UTILITY,
  OrgKind.OTHER,
];

const JURISDICTION_OPTIONS: OrgJurisdiction[] = [
  OrgJurisdiction.NATIONAL,
  OrgJurisdiction.STATE,
  OrgJurisdiction.LGA,
  OrgJurisdiction.COMMUNITY,
];

function useOrganizations(kind: OrgKind | 'ALL') {
  return useQuery({
    queryKey: ['organizations', kind],
    queryFn: async () => {
      const params = kind !== 'ALL' ? { kind } : undefined;
      const res = await api.get<ApiResponse<{ organizations: Organization[] }>>(
        '/api/v1/organizations',
        { params },
      );
      return res.data.data.organizations;
    },
  });
}

export function OrganizationsPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const meQuery = useMe();
  const [kind, setKind] = useState<OrgKind | 'ALL'>('ALL');
  const [isModalOpen, setModalOpen] = useState(false);

  const orgsQuery = useOrganizations(kind);
  const orgs = orgsQuery.data ?? [];
  const canCreate = meQuery.data?.role ? CREATOR_ROLES.has(meQuery.data.role) : false;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('organizationsPage.eyebrow')}
        title={t('organizationsPage.title')}
        subtitle={t('organizationsPage.subtitle')}
        meta={meta}
        actions={
          canCreate ? (
            <Button size="sm" onClick={() => setModalOpen(true)}>
              {t('organizationsPage.newBtn')}
            </Button>
          ) : undefined
        }
      >
        <div className="mt-4 flex flex-wrap gap-2">
          {KIND_OPTIONS.map((k) => (
            <FilterPill
              key={k}
              active={kind === k}
              onClick={() => setKind(k)}
              label={t(`organizationsPage.kinds.${k}`)}
            />
          ))}
        </div>
      </PageHeader>

      {orgsQuery.isLoading ? (
        <p className="text-sm text-slate-600">{t('common.loading')}</p>
      ) : orgs.length === 0 ? (
        <EmptyState icon={<Building2 className="h-5 w-5" />} title={t('organizationsPage.empty')} />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {orgs.map((org) => (
            <OrganizationCard key={org.id} org={org} />
          ))}
        </div>
      )}

      {isModalOpen && <NewOrganizationModal onClose={() => setModalOpen(false)} />}
    </section>
  );
}

function FilterPill({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`rounded-full border px-3 py-1 text-xs font-semibold transition ${
        active
          ? 'border-civic-500 bg-civic-100 text-civic-700'
          : 'border-slate-200 bg-white text-slate-600 hover:border-civic-300'
      }`}
    >
      {label}
    </button>
  );
}

function OrganizationCard({ org }: { org: Organization }) {
  const { t } = useTranslation();
  return (
    <Link
      to={`/organizations/${org.id}`}
      className="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-lg font-semibold text-slate-900">{org.name}</h2>
            {org.verified && (
              <ShieldCheck
                className="h-4 w-4 flex-shrink-0 text-emerald-600"
                aria-label={t('organizationsPage.card.verified')}
              />
            )}
          </div>
          <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700">
            {t(`organizationsPage.kinds.${org.kind}`)} ·{' '}
            {t(`organizationsPage.jurisdictions.${org.jurisdiction}`)}
          </p>
        </div>
      </div>

      {org.description && <p className="line-clamp-2 text-sm text-slate-600">{org.description}</p>}

      <div className="mt-auto flex flex-wrap gap-3 text-xs text-slate-600">
        <span>{t('organizationsPage.card.members', { count: org.memberCount })}</span>
        <span>·</span>
        <span>{t('organizationsPage.card.announcements', { count: org.announcementCount })}</span>
        <span>·</span>
        <span>{t('organizationsPage.card.projects', { count: org.projectCount })}</span>
        <span>·</span>
        <span>{t('organizationsPage.card.reports', { count: org.assignmentCount })}</span>
      </div>
    </Link>
  );
}

function NewOrganizationModal({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [kind, setKind] = useState<OrgKind>(OrgKind.GOVERNMENT);
  const [jurisdiction, setJurisdiction] = useState<OrgJurisdiction>(OrgJurisdiction.LGA);
  const [description, setDescription] = useState('');
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation({
    mutationFn: async () => {
      await api.post('/api/v1/organizations', {
        name,
        slug: slug.trim().toLowerCase(),
        kind,
        jurisdiction,
        description: description.trim() || undefined,
        email: email.trim() || undefined,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
      onClose();
    },
    onError: () => setError(t('organizationsPage.modal.genericError')),
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  return (
    <Modal title={t('organizationsPage.modal.title')} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          label={t('organizationsPage.modal.fields.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          minLength={2}
        />
        <Input
          label={t('organizationsPage.modal.fields.slug')}
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          required
          minLength={2}
          placeholder="e.g. lagos-water-corp"
        />

        <div className="grid gap-3 md:grid-cols-2">
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-gray-700" htmlFor="new-org-kind">
              {t('organizationsPage.modal.fields.kind')}
            </label>
            <select
              id="new-org-kind"
              value={kind}
              onChange={(e) => setKind(e.target.value as OrgKind)}
              className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-civic-500"
            >
              {[
                OrgKind.GOVERNMENT,
                OrgKind.AGENCY,
                OrgKind.NGO,
                OrgKind.UTILITY,
                OrgKind.OTHER,
              ].map((k) => (
                <option key={k} value={k}>
                  {t(`organizationsPage.kinds.${k}`)}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-gray-700" htmlFor="new-org-jurisdiction">
              {t('organizationsPage.modal.fields.jurisdiction')}
            </label>
            <select
              id="new-org-jurisdiction"
              value={jurisdiction}
              onChange={(e) => setJurisdiction(e.target.value as OrgJurisdiction)}
              className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-civic-500"
            >
              {JURISDICTION_OPTIONS.map((j) => (
                <option key={j} value={j}>
                  {t(`organizationsPage.jurisdictions.${j}`)}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-sm font-medium text-gray-700" htmlFor="new-org-desc">
            {t('organizationsPage.modal.fields.description')}
          </label>
          <textarea
            id="new-org-desc"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
          />
        </div>

        <Input
          label={t('organizationsPage.modal.fields.email')}
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            {t('organizationsPage.modal.save')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
