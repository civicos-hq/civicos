import { Navigate, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { UserRole } from '@civicos/types';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { useMyOrganizations } from '../hooks/useConsultations';
import { useMe } from '../hooks/useMe';
import { useProvisionRepresentativeOffice } from '../hooks/useRepresentativeOffice';
import { getApiError } from '../lib/api';
import { Button } from '@civicos/ui';
import { Briefcase, Landmark } from 'lucide-react';

// If the caller belongs to exactly one org they can admin, skip the
// picker and drop them straight into that org's dashboard — a citizen
// who owns their local NGO shouldn't have to click through a one-item
// list every time.
export function OrgLandingPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const { data: memberships = [], isLoading } = useMyOrganizations();
  const { data: me } = useMe();
  const provision = useProvisionRepresentativeOffice();

  const admins = memberships.filter(
    (m) => m.membership.role === 'OWNER' || m.membership.role === 'ADMIN',
  );

  if (isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }

  if (admins.length === 1) {
    return <Navigate to={`/org/${admins[0].organization.id}`} replace />;
  }

  // An approved representative with no office yet. Everything an
  // organization can do — campaigns, projects, consultations,
  // announcements — becomes available the moment this exists, because the
  // office IS an organization and they own it.
  if (admins.length === 0 && me?.role === UserRole.REPRESENTATIVE) {
    return <SetUpOfficeCard meta={meta} provision={provision} />;
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
                className="block rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:border-slate-700 dark:bg-slate-800/70 p-4 md:p-5 shadow-sm transition hover:border-civic-300 dark:hover:border-civic-500 hover:shadow-md"
              >
                <h2 className="font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
                  {organization.name}
                </h2>
                <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
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

/**
 * One-time setup for an elected representative's constituency office.
 *
 * Behind a button rather than created on page load: this publishes a new,
 * publicly listed entity under the representative's own name, and that
 * should be a decision they make rather than a side effect of navigation.
 */
function SetUpOfficeCard({
  meta,
  provision,
}: {
  meta: ReturnType<typeof useTodayMeta>;
  provision: ReturnType<typeof useProvisionRepresentativeOffice>;
}) {
  const { t } = useTranslation();
  const apiError = getApiError(provision.error);
  // An unclaimed profile is an administrative gap, not the caller doing
  // something wrong, so it gets its own message naming who can fix it.
  const unclaimed = apiError?.code === 'REPRESENTATIVE_UNCLAIMED';

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('repOffice.eyebrow')}
        title={t('repOffice.title')}
        subtitle={t('repOffice.subtitle')}
        meta={meta}
      />

      <article className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-800/70">
        <div className="flex items-start gap-3">
          <span className="rounded-xl bg-civic-100 p-2 text-civic-700 dark:bg-civic-500/15 dark:text-civic-300">
            <Landmark className="h-5 w-5" aria-hidden="true" />
          </span>
          <div className="flex-1">
            <h2 className="font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
              {t('repOffice.cardTitle')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              {t('repOffice.cardBody')}
            </p>
            <ul className="mt-3 grid gap-1.5 text-sm text-slate-600 dark:text-slate-300">
              <li>• {t('repOffice.capabilities.campaigns')}</li>
              <li>• {t('repOffice.capabilities.projects')}</li>
              <li>• {t('repOffice.capabilities.consultations')}</li>
              <li>• {t('repOffice.capabilities.announcements')}</li>
            </ul>

            {/* Provisioning grants drafting, not money. Saying so here
                avoids a representative publishing a campaign and only then
                discovering it cannot take a naira until an admin has
                verified the office and a payout account is connected. */}
            <p className="mt-3 rounded-lg bg-amber-50 p-3 text-xs text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
              {t('repOffice.fundingNote')}
            </p>

            {apiError && (
              <p className="mt-3 text-sm text-red-600 dark:text-red-400">
                {unclaimed ? t('repOffice.unclaimedError') : apiError.message}
              </p>
            )}

            <div className="mt-4">
              <Button
                onClick={() => provision.mutate()}
                loading={provision.isPending}
                disabled={unclaimed}
              >
                {t('repOffice.action')}
              </Button>
            </div>
          </div>
        </div>
      </article>
    </section>
  );
}
