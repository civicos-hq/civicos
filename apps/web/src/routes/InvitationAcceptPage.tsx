import { useEffect } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { Building2, MailX } from 'lucide-react';
import { useAcceptInvitation, useInvitationPreview } from '../hooks/useOrgInvitations';
import { useOptionalMe } from '../hooks/useMe';
import { getApiError } from '../lib/api';
import { PageHeader, useTodayMeta } from '../components/PageHeader';

/**
 * The page an invitation link opens.
 *
 * Public, because the whole point is that the invitee may not have an
 * account yet: they need to see who is asking and what for before deciding
 * whether to create one. The preview endpoint is deliberately thin for the
 * same reason — anyone holding the link can read this.
 *
 * Four states worth getting right, because each has a different next step:
 * link is dead, nobody is signed in, the wrong account is signed in, and
 * the right one is.
 */
export function InvitationAcceptPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();

  const { data: me, isLoading: meLoading } = useOptionalMe();
  const preview = useInvitationPreview(token);
  const accept = useAcceptInvitation(token);

  // Land them inside the organization they just joined rather than on a
  // success message they have to act on.
  useEffect(() => {
    if (accept.isSuccess) {
      navigate(`/org/${accept.data.organizationId}`, { replace: true });
    }
  }, [accept.isSuccess, accept.data, navigate]);

  if (preview.isLoading || meLoading) {
    return <p className="p-6 text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>;
  }

  // Expired, already used, or withdrawn — the server does not distinguish
  // them, so neither can this page.
  if (preview.isError || !preview.data) {
    return (
      <Shell
        title={t('invitation.invalidTitle')}
        subtitle={t('invitation.invalidSubtitle')}
        meta={meta}
      >
        <div className="flex items-start gap-3">
          <MailX className="mt-0.5 h-5 w-5 text-slate-400" aria-hidden="true" />
          <div>
            <p className="text-sm text-slate-600 dark:text-slate-300">
              {t('invitation.invalidBody')}
            </p>
            <Link
              to="/discover"
              className="mt-3 inline-block text-sm font-semibold text-civic-700 hover:underline dark:text-civic-200"
            >
              {t('invitation.goToCivicOS')}
            </Link>
          </div>
        </div>
      </Shell>
    );
  }

  const invite = preview.data;
  const signedInAs = me?.email?.toLowerCase();
  const invitedEmail = invite.email.toLowerCase();
  const wrongAccount = Boolean(signedInAs) && signedInAs !== invitedEmail;
  const acceptError = getApiError(accept.error);

  return (
    <Shell
      title={t('invitation.title', { org: invite.organizationName })}
      subtitle={t('invitation.subtitle', {
        inviter: invite.invitedByName,
        org: invite.organizationName,
      })}
      meta={meta}
    >
      <div className="flex items-start gap-3">
        <span className="rounded-xl bg-civic-100 p-2 text-civic-700 dark:bg-civic-500/15 dark:text-civic-300">
          <Building2 className="h-5 w-5" aria-hidden="true" />
        </span>
        <div className="flex-1">
          <dl className="grid gap-1.5 text-sm">
            <Row label={t('invitation.fields.organization')} value={invite.organizationName} />
            <Row label={t('invitation.fields.role')} value={t(`orgMembers.roles.${invite.role}`)} />
            {invite.title && <Row label={t('invitation.fields.jobTitle')} value={invite.title} />}
            <Row label={t('invitation.fields.sentTo')} value={invite.email} />
            <Row
              label={t('invitation.fields.expires')}
              value={new Date(invite.expiresAt).toLocaleDateString()}
            />
          </dl>

          {/* Nobody signed in — they either have an account or need one,
              and either way the invited address is the one to use. */}
          {!me && (
            <div className="mt-4 space-y-3">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                {t('invitation.signInPrompt', { email: invite.email })}
              </p>
              <div className="flex flex-wrap gap-2">
                <Link
                  to={`/register?email=${encodeURIComponent(invite.email)}&redirect=${encodeURIComponent(`/invitations/${token}`)}`}
                >
                  <Button size="sm">{t('invitation.createAccount')}</Button>
                </Link>
                <Link to={`/login?redirect=${encodeURIComponent(`/invitations/${token}`)}`}>
                  <Button size="sm" variant="secondary">
                    {t('invitation.signIn')}
                  </Button>
                </Link>
              </div>
            </div>
          )}

          {/* Signed in as somebody else. The common cause is a personal
              account in the browser and a work address on the invitation,
              so name both rather than just refusing. */}
          {wrongAccount && (
            <div className="mt-4 space-y-3">
              <p className="rounded-lg bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
                {t('invitation.wrongAccount', { invited: invite.email, current: me?.email })}
              </p>
              <Link to={`/login?redirect=${encodeURIComponent(`/invitations/${token}`)}`}>
                <Button size="sm" variant="secondary">
                  {t('invitation.switchAccount')}
                </Button>
              </Link>
            </div>
          )}

          {me && !wrongAccount && (
            <div className="mt-4">
              {acceptError && (
                <p className="mb-3 text-sm text-red-600 dark:text-red-400">{acceptError.message}</p>
              )}
              <Button onClick={() => accept.mutate()} loading={accept.isPending}>
                {t('invitation.accept')}
              </Button>
            </div>
          )}
        </div>
      </div>
    </Shell>
  );
}

function Shell({
  title,
  subtitle,
  meta,
  children,
}: {
  title: string;
  subtitle: string;
  meta: ReturnType<typeof useTodayMeta>;
  children: React.ReactNode;
}) {
  return (
    <section className="mx-auto max-w-2xl space-y-6 p-4 md:p-6">
      <PageHeader eyebrow="CivicOS" title={title} subtitle={subtitle} meta={meta} />
      <article className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-800/70">
        {children}
      </article>
    </section>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-wrap gap-x-2">
      <dt className="text-slate-500 dark:text-slate-400">{label}</dt>
      <dd className="font-medium text-slate-900 dark:text-slate-100">{value}</dd>
    </div>
  );
}
