import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Building2, ShieldCheck } from 'lucide-react';
import { OrgKind, type ApiResponse, type Organization } from '@civicos/types';
import { api } from '../lib/api';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';

const KIND_OPTIONS: (OrgKind | 'ALL')[] = [
  'ALL',
  OrgKind.GOVERNMENT,
  OrgKind.AGENCY,
  OrgKind.NGO,
  OrgKind.UTILITY,
  OrgKind.OTHER,
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
  const [kind, setKind] = useState<OrgKind | 'ALL'>('ALL');

  const orgsQuery = useOrganizations(kind);
  const orgs = orgsQuery.data ?? [];

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('organizationsPage.eyebrow')}
        title={t('organizationsPage.title')}
        subtitle={t('organizationsPage.subtitle')}
        meta={meta}
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

      {/* Orgs join by applying at signup — no in-app direct create. */}
      <p className="text-xs text-slate-500">
        {t('organizationsPage.applyPrompt')}{' '}
        <Link to="/register" className="font-semibold text-civic-700 hover:underline">
          {t('organizationsPage.applyCta')}
        </Link>
      </p>
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
