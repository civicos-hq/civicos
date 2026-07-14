import { useState, type FormEvent } from 'react';
import axios from 'axios';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import type { ApiResponse, Community, User } from '@civicos/types';
import { api, signOut as signOutRemote } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useEnumLabels } from '../hooks/useEnumLabels';
import { PageHeader, useTodayMeta } from '../components/PageHeader';

function useCommunity(id: string | undefined) {
  return useQuery({
    queryKey: ['community', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ community: Community }>>(`/api/v1/communities/${id}`);
      return res.data.data.community;
    },
  });
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[parts.length - 1]![0]!).toUpperCase();
}

const ROLE_TONE: Record<string, string> = {
  CITIZEN: 'bg-slate-100 dark:bg-slate-800/60 text-slate-700 dark:text-slate-300',
  REPRESENTATIVE: 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300',
  GOVERNMENT_ADMIN: 'bg-sky-100 dark:bg-sky-500/15 text-sky-700 dark:text-sky-300',
  PLATFORM_ADMIN: 'bg-civic-100 dark:bg-civic-500/15 text-civic-700 dark:text-civic-200',
  NGO: 'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
  MODERATOR: 'bg-purple-100 dark:bg-purple-500/15 text-purple-700 dark:text-purple-300',
};

export function ProfilePage() {
  const { t, i18n } = useTranslation();
  const enums = useEnumLabels();
  const meta = useTodayMeta();
  const navigate = useNavigate();
  const meQuery = useMe();
  const me = meQuery.data;
  const communityQuery = useCommunity(me?.activeCommunityId);

  async function signOut() {
    // Server-side revoke of the refresh-token family, then wipe local state.
    await signOutRemote();
    navigate('/login');
  }

  if (meQuery.isLoading) {
    return (
      <p className="text-sm text-slate-600 dark:text-slate-400">
        {t('profilePage.loadingProfile')}
      </p>
    );
  }

  if (!me) {
    return <p className="text-sm text-red-600 dark:text-red-400">{t('profilePage.loadError')}</p>;
  }

  const roleTone =
    ROLE_TONE[me.role] ?? 'bg-slate-100 dark:bg-slate-800/60 text-slate-700 dark:text-slate-300';
  const community = communityQuery.data;
  const membershipCount = me.memberships.length;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('profilePage.eyebrow')}
        title={
          <span className="profile-header-identity">
            <span className="profile-header-avatar" aria-hidden="true">
              {initials(me.name)}
            </span>
            {me.name}
          </span>
        }
        subtitle={me.email}
        meta={meta}
        actions={
          <span
            className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${roleTone}`}
          >
            {enums.userRole(me.role)}
          </span>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
        <AccountSection user={me} language={i18n.language} />

        <aside className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('profilePage.community.heading')}
          </h2>
          {communityQuery.isLoading ? (
            <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>
          ) : community ? (
            <div className="mt-3 space-y-2 text-sm text-slate-700 dark:text-slate-300">
              <p className="text-base font-semibold text-slate-900 dark:text-slate-100">
                {community.name}
              </p>
              <p className="text-slate-600 dark:text-slate-400">
                {community.lga}, {community.state}
              </p>
              {community.description && (
                <p className="pt-1 text-slate-600 dark:text-slate-400">{community.description}</p>
              )}
              <p className="pt-1 text-xs font-medium uppercase tracking-[0.12em] text-slate-500 dark:text-slate-400">
                {t('profilePage.community.joinedCount', { count: membershipCount })}
              </p>
              <Link
                to="/community"
                className="mt-2 inline-block text-sm font-semibold text-civic-700 dark:text-civic-200 hover:text-civic-800"
              >
                {t('profilePage.community.manage')} →
              </Link>
            </div>
          ) : (
            <div className="mt-3 space-y-2 text-sm text-slate-600 dark:text-slate-400">
              <p>{t('profilePage.community.empty')}</p>
              <Link
                to="/community"
                className="inline-block text-sm font-semibold text-civic-700 dark:text-civic-200 hover:text-civic-800"
              >
                {t('profilePage.community.findMine')} →
              </Link>
            </div>
          )}
        </aside>
      </div>

      <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {t('profilePage.session.heading')}
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          {t('profilePage.session.sub')}
        </p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={signOut}>
          {t('profilePage.session.signOut')}
        </Button>
      </section>

      <ApprovalSection user={me} />

      <DangerZone />
    </section>
  );
}

// DangerZone — self-service account deletion. Requires typing "DELETE"
// so an accidental button press can't purge anyone. On success, the
// server anonymizes PII + revokes refresh tokens; the client clears its
// own local session and navigates to the homepage. Matches the
// commitment in the privacy notice (§7 — deletion within 30 days).
function DangerZone() {
  const { t } = useTranslation();
  const [confirmText, setConfirmText] = useState('');
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const CONFIRM = 'DELETE';

  async function handleDelete() {
    setError('');
    setPending(true);
    try {
      await api.delete('/api/v1/auth/me', {
        data: { reason: reason.trim() || undefined },
      });
      // Clear local session state and bounce to the marketing homepage.
      try {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
      } catch {
        // ignore
      }
      window.location.href = '/';
    } catch (err) {
      const msg =
        (err as { response?: { data?: { message?: string } } }).response?.data?.message ??
        t('profilePage.delete.errorGeneric');
      setError(msg);
      setPending(false);
    }
  }

  return (
    <section className="rounded-2xl border border-red-200 bg-red-50/50 dark:bg-red-500/10 p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-red-900">{t('profilePage.delete.heading')}</h2>
      <p className="mt-2 text-sm text-red-800">{t('profilePage.delete.sub')}</p>
      <ul className="mt-3 list-disc pl-5 text-sm text-red-800 space-y-1">
        <li>{t('profilePage.delete.bullets.pii')}</li>
        <li>{t('profilePage.delete.bullets.content')}</li>
        <li>{t('profilePage.delete.bullets.sessions')}</li>
        <li>{t('profilePage.delete.bullets.irreversible')}</li>
      </ul>

      <div className="mt-4 space-y-3">
        <div>
          <label
            htmlFor="delete-reason"
            className="block text-xs font-semibold text-slate-700 dark:text-slate-300"
          >
            {t('profilePage.delete.reasonLabel')}
          </label>
          <textarea
            id="delete-reason"
            rows={2}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-red-400"
            placeholder={t('profilePage.delete.reasonPlaceholder')}
          />
        </div>

        <div>
          <label
            htmlFor="delete-confirm"
            className="block text-xs font-semibold text-slate-700 dark:text-slate-300"
          >
            {t('profilePage.delete.confirmLabel', { confirm: CONFIRM })}
          </label>
          <input
            id="delete-confirm"
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-red-400 mono"
            placeholder={CONFIRM}
            autoComplete="off"
          />
        </div>

        {error && (
          <p className="text-sm text-red-900 bg-red-100 dark:bg-red-500/15 border border-red-300 rounded-lg p-2">
            {error}
          </p>
        )}

        <Button
          variant="primary"
          size="sm"
          onClick={handleDelete}
          disabled={confirmText !== CONFIRM || pending}
          className="bg-red-600 hover:bg-red-700 focus:ring-red-400"
        >
          {pending ? t('profilePage.delete.pending') : t('profilePage.delete.button')}
        </Button>
      </div>
    </section>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-600 dark:text-slate-400">
        {label}
      </dt>
      <dd className="mt-1 text-sm text-slate-900 dark:text-slate-100">{value}</dd>
    </div>
  );
}

function ApprovalSection({ user }: { user: User }) {
  const { t } = useTranslation();
  const enums = useEnumLabels();

  if (user.requestedAccountType === 'CITIZEN' && user.approvalStatus === 'NONE') {
    return null;
  }

  return (
    <section className="rounded-2xl border border-amber-200 dark:border-amber-500/40 bg-amber-50/60 dark:bg-amber-500/10 p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-amber-950">{t('profilePage.approval.heading')}</h2>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <Field
          label={t('profilePage.approval.requestedType')}
          value={enums.requestedAccountType(user.requestedAccountType)}
        />
        <Field
          label={t('profilePage.approval.status')}
          value={enums.approvalStatus(user.approvalStatus)}
        />
      </div>
      <p className="mt-3 text-sm text-amber-900">
        {t(`profilePage.approval.messages.${user.approvalStatus}`)}
      </p>
      {user.approvalNote && (
        <p className="mt-2 text-sm text-amber-900">
          <strong>{t('profilePage.approval.reviewNote')}:</strong> {user.approvalNote}
        </p>
      )}
    </section>
  );
}

function AccountSection({ user, language }: { user: User; language: string }) {
  const { t } = useTranslation();
  const enums = useEnumLabels();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);
  const [error, setError] = useState('');

  function startEdit() {
    setName(user.name);
    setEmail(user.email);
    setError('');
    setEditing(true);
  }

  function cancel() {
    setEditing(false);
    setError('');
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const payload: { name?: string; email?: string } = {};
      if (name.trim() !== user.name) payload.name = name.trim();
      if (email.trim() !== user.email) payload.email = email.trim();
      if (Object.keys(payload).length === 0) return user;
      const res = await api.patch<ApiResponse<{ user: User }>>('/api/v1/auth/me', payload);
      return res.data.data.user;
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated);
      setEditing(false);
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err)) {
        const data = err.response?.data;
        const code: string | undefined = data?.error?.code ?? data?.code;
        if (code === 'EMAIL_ALREADY_IN_USE') {
          setError(t('profilePage.errors.emailInUse'));
          return;
        }
        if (code === 'DISPOSABLE_EMAIL_DOMAIN') {
          setError(t('profilePage.errors.disposableEmail'));
          return;
        }
        if (code === 'VALIDATION_ERROR') {
          setError(t('profilePage.errors.validation'));
          return;
        }
      }
      setError(t('profilePage.errors.generic'));
    },
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  const formatJoined = new Intl.DateTimeFormat(language, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(new Date(user.createdAt));

  if (!editing) {
    return (
      <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
        <div className="flex items-start justify-between gap-3">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('profilePage.account')}
          </h2>
          <Button size="sm" variant="secondary" onClick={startEdit}>
            {t('common.edit')}
          </Button>
        </div>
        <dl className="mt-4 grid gap-3 sm:grid-cols-2">
          <Field label={t('profilePage.fields.fullName')} value={user.name} />
          <Field label={t('profilePage.fields.email')} value={user.email} />
          <Field label={t('profilePage.fields.role')} value={enums.userRole(user.role)} />
          <Field label={t('profilePage.fields.memberSince')} value={formatJoined} />
        </dl>
      </section>
    );
  }

  const dirty = name.trim() !== user.name || email.trim() !== user.email;

  return (
    <section className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
        {t('profilePage.editAccount')}
      </h2>
      <form className="mt-4 grid gap-3 sm:grid-cols-2" onSubmit={submit}>
        <label className="text-sm text-slate-700 dark:text-slate-300">
          {t('profilePage.fields.fullName')}
          <Input
            className="mt-1.5"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
            maxLength={100}
          />
        </label>
        <label className="text-sm text-slate-700 dark:text-slate-300">
          {t('profilePage.fields.email')}
          <Input
            className="mt-1.5"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>

        {error && <p className="sm:col-span-2 text-sm text-red-600 dark:text-red-400">{error}</p>}

        <div className="sm:col-span-2 mt-2 flex items-center justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={cancel}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" size="sm" loading={mutation.isPending} disabled={!dirty}>
            {t('profilePage.saveChanges')}
          </Button>
        </div>
      </form>
    </section>
  );
}
