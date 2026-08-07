import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { OrgMemberRole, type OrgMember } from '@civicos/types';
import { UserPlus, Trash2 } from 'lucide-react';
import {
  useAddOrgMember,
  useOrgMembers,
  useRemoveOrgMember,
  useUpdateOrgMember,
} from '../hooks/useOrgMembers';
import { getApiError } from '../lib/api';
import { EmptyState } from './EmptyState';

/**
 * Staff list and member management for an organization.
 *
 * A utility — a water board, an electricity distributor — has many people
 * with different jobs, and until this existed there was no way to put any
 * of them on CivicOS: the endpoints were there, but nothing in either app
 * called them, so membership could only be granted by hand against the API.
 *
 * Two different things are shown per person on purpose. `role` is what they
 * may do on CivicOS; `title` is their actual job. A citizen reading an
 * update wants to know they are hearing from the Head of Distribution — and
 * that is not the same question as whether that person may publish.
 */
export function OrgMembers({ orgId, canManage }: { orgId: string; canManage: boolean }) {
  const { t } = useTranslation();
  const { data: members = [], isLoading } = useOrgMembers(orgId);
  const [adding, setAdding] = useState(false);

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-6 dark:border-slate-700 dark:bg-slate-800/70">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="font-fraunces text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('orgMembers.title')}
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
            {t('orgMembers.subtitle')}
          </p>
        </div>
        {canManage && !adding && (
          <Button size="sm" onClick={() => setAdding(true)}>
            <UserPlus className="mr-1.5 h-4 w-4" aria-hidden="true" />
            {t('orgMembers.add')}
          </Button>
        )}
      </div>

      {adding && <AddMemberForm orgId={orgId} onDone={() => setAdding(false)} />}

      {isLoading ? (
        <p className="mt-4 text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      ) : members.length === 0 ? (
        <div className="mt-4">
          <EmptyState icon={<UserPlus className="h-5 w-5" />} title={t('orgMembers.empty')} />
        </div>
      ) : (
        <ul className="mt-4 grid gap-3">
          {members.map((m) => (
            <MemberRow key={m.id} orgId={orgId} member={m} canManage={canManage} />
          ))}
        </ul>
      )}
    </section>
  );
}

function AddMemberForm({ orgId, onDone }: { orgId: string; onDone: () => void }) {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [title, setTitle] = useState('');
  const [role, setRole] = useState<OrgMemberRole>(OrgMemberRole.STAFF);
  const add = useAddOrgMember(orgId);
  const apiError = getApiError(add.error);

  return (
    <form
      className="mt-4 grid gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 sm:grid-cols-2 dark:border-slate-700 dark:bg-slate-800/40"
      onSubmit={(e) => {
        e.preventDefault();
        add.mutate({ email, role, title }, { onSuccess: onDone });
      }}
    >
      <label className="flex flex-col gap-1 text-sm sm:col-span-2">
        <span className="font-medium text-slate-700 dark:text-slate-200">
          {t('orgMembers.form.email')}
        </span>
        <input
          type="email"
          required
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-800"
          placeholder={t('orgMembers.form.emailPlaceholder')}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        {/* Membership is invitation-only — the person must already have an
            account, and this form cannot create one for them. */}
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {t('orgMembers.form.emailHint')}
        </span>
      </label>

      <label className="flex flex-col gap-1 text-sm">
        <span className="font-medium text-slate-700 dark:text-slate-200">
          {t('orgMembers.form.jobTitle')}
        </span>
        <input
          type="text"
          maxLength={120}
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-800"
          placeholder={t('orgMembers.form.jobTitlePlaceholder')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </label>

      <label className="flex flex-col gap-1 text-sm">
        <span className="font-medium text-slate-700 dark:text-slate-200">
          {t('orgMembers.form.role')}
        </span>
        <select
          className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-800"
          value={role}
          onChange={(e) => setRole(e.target.value as OrgMemberRole)}
        >
          <option value={OrgMemberRole.STAFF}>{t('orgMembers.roles.STAFF')}</option>
          <option value={OrgMemberRole.ADMIN}>{t('orgMembers.roles.ADMIN')}</option>
          <option value={OrgMemberRole.OWNER}>{t('orgMembers.roles.OWNER')}</option>
        </select>
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {t(`orgMembers.roleHints.${role}`)}
        </span>
      </label>

      {apiError && (
        <p className="text-sm text-red-600 sm:col-span-2 dark:text-red-400">{apiError.message}</p>
      )}

      <div className="flex gap-2 sm:col-span-2">
        <Button type="submit" size="sm" loading={add.isPending}>
          {t('orgMembers.form.submit')}
        </Button>
        <Button type="button" size="sm" variant="secondary" onClick={onDone}>
          {t('common.cancel')}
        </Button>
      </div>
    </form>
  );
}

function MemberRow({
  orgId,
  member,
  canManage,
}: {
  orgId: string;
  member: OrgMember;
  canManage: boolean;
}) {
  const { t } = useTranslation();
  const update = useUpdateOrgMember(orgId);
  const remove = useRemoveOrgMember(orgId);
  const [confirming, setConfirming] = useState(false);

  return (
    <li className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-700 dark:bg-slate-800/40">
      <div className="min-w-0">
        <p className="font-semibold text-slate-900 dark:text-slate-100">{member.userName}</p>
        {member.title && (
          <p className="text-sm text-slate-600 dark:text-slate-300">{member.title}</p>
        )}
        <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
          {t('orgMembers.joined', { date: new Date(member.joinedAt).toLocaleDateString() })}
        </p>
      </div>

      <div className="flex items-center gap-2">
        {canManage ? (
          <select
            className="rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-slate-600 dark:bg-slate-800"
            value={member.role}
            disabled={update.isPending}
            onChange={(e) =>
              update.mutate({ userId: member.userId, role: e.target.value as OrgMemberRole })
            }
            aria-label={t('orgMembers.changeRoleFor', { name: member.userName })}
          >
            <option value={OrgMemberRole.STAFF}>{t('orgMembers.roles.STAFF')}</option>
            <option value={OrgMemberRole.ADMIN}>{t('orgMembers.roles.ADMIN')}</option>
            <option value={OrgMemberRole.OWNER}>{t('orgMembers.roles.OWNER')}</option>
          </select>
        ) : (
          <span className="rounded-full bg-slate-200 px-2.5 py-1 text-xs font-medium text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {t(`orgMembers.roles.${member.role}`)}
          </span>
        )}

        {canManage &&
          (confirming ? (
            <div className="flex items-center gap-1.5">
              <Button
                size="sm"
                variant="danger"
                loading={remove.isPending}
                onClick={() => remove.mutate(member.userId)}
              >
                {t('orgMembers.confirmRemove')}
              </Button>
              <Button size="sm" variant="secondary" onClick={() => setConfirming(false)}>
                {t('common.cancel')}
              </Button>
            </div>
          ) : (
            <button
              type="button"
              className="rounded-lg p-2 text-slate-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"
              onClick={() => setConfirming(true)}
              aria-label={t('orgMembers.removeMember', { name: member.userName })}
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
            </button>
          ))}
      </div>
    </li>
  );
}
