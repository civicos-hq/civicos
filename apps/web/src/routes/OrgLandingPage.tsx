import { Navigate, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useMyOrganizations } from '../hooks/useConsultations';
import { Briefcase } from 'lucide-react';

// If the caller belongs to exactly one org they can admin, skip the
// picker and drop them straight into that org's dashboard — a citizen
// who owns their local NGO shouldn't have to click through a one-item
// list every time.
export function OrgLandingPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const { data: memberships = [], isLoading } = useMyOrganizations();

  const admins = memberships.filter(
    (m) => m.membership.role === 'OWNER' || m.membership.role === 'ADMIN',
  );

  if (isLoading) {
    return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  }

  if (admins.length === 1) {
    return <Navigate to={`/org/${admins[0].organization.id}`} replace />;
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('orgLanding.eyebrow')}
        title={t('orgLanding.title')}
        subtitle={t('orgLanding.subtitle')}
        meta={meta}
      />

      {admins.length === 0 ? (
        <EmptyState
          icon={<Briefcase size={20} />}
          title={t('orgLanding.empty.title')}
          body={t('orgLanding.empty.body')}
        />
      ) : (
        <ul className="space-y-3">
          {admins.map(({ organization, membership }) => (
            <li key={organization.id}>
              <Link
                to={`/org/${organization.id}`}
                className="block rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-civic-300 hover:shadow-md"
              >
                <h2 className="font-fraunces text-lg font-semibold text-slate-900">
                  {organization.name}
                </h2>
                <p className="mt-1 text-sm text-slate-600">
                  {t('orgLanding.actingAs', { role: membership.role })}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
