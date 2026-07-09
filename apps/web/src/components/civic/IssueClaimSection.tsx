import { useMemo, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Input } from '@civicos/ui';
import { AssignmentStatus, type IssueAssignment, type MyOrgMembership } from '@civicos/types';
import { getApiError } from '../../lib/api';
import { useMyOrganizations } from '../../hooks/useConsultations';
import { useCreateAssignment, useIssueAssignments } from '../../hooks/useAssignments';

const TONE: Record<AssignmentStatus, string> = {
  [AssignmentStatus.RECEIVED]: 'bg-slate-200 text-slate-700',
  [AssignmentStatus.IN_PROGRESS]: 'bg-civic-100 text-civic-700',
  [AssignmentStatus.COMPLETED]: 'bg-emerald-100 text-emerald-700',
  [AssignmentStatus.REJECTED]: 'bg-rose-100 text-rose-700',
};

/**
 * The "take responsibility" block on an issue page.
 *
 * Two things happen here:
 *   1. Everyone (logged in or not) sees the list of orgs already on this
 *      issue — that's a public accountability signal.
 *   2. Users who are OWNER/ADMIN of at least one org see a form to
 *      claim the issue on behalf of one of those orgs, with an optional
 *      note. Orgs already assigned are filtered out of the picker so
 *      you can't try to double-claim from the same org.
 */
export function IssueClaimSection({ issueId }: { issueId: string }) {
  const { t } = useTranslation();

  const assignmentsQuery = useIssueAssignments(issueId);
  const membershipsQuery = useMyOrganizations();

  const assignments = assignmentsQuery.data ?? [];
  const memberships = membershipsQuery.data ?? [];

  // Orgs the caller can act as, minus any already on this issue. Prevents
  // wasted round-trips (the server would 409 on duplicate assignments).
  const claimableOrgs: MyOrgMembership[] = useMemo(() => {
    const assignedOrgIds = new Set(assignments.map((a) => a.organizationId));
    return memberships.filter(
      (m) =>
        (m.membership.role === 'OWNER' || m.membership.role === 'ADMIN') &&
        !assignedOrgIds.has(m.organization.id),
    );
  }, [assignments, memberships]);

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h3 className="mb-3 font-fraunces text-base font-semibold text-slate-900">
        {t('issueClaim.heading')}
      </h3>

      {assignments.length === 0 ? (
        <p className="text-sm text-slate-600">{t('issueClaim.none')}</p>
      ) : (
        <ul className="space-y-2">
          {assignments.map((a) => (
            <AssignmentRow key={a.id} assignment={a} />
          ))}
        </ul>
      )}

      {claimableOrgs.length > 0 && <ClaimForm issueId={issueId} orgs={claimableOrgs} />}
    </section>
  );
}

function AssignmentRow({ assignment }: { assignment: IssueAssignment }) {
  const { t } = useTranslation();
  return (
    <li className="flex flex-wrap items-center gap-2 text-sm">
      <span className="font-semibold text-slate-800">{assignment.assignedByName}</span>
      <span
        className={
          'rounded-full px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ' +
          TONE[assignment.status]
        }
      >
        {t(`orgDashboard.assignmentStatus.${assignment.status}`)}
      </span>
      {assignment.note && (
        <span className="italic text-slate-600">&ldquo;{assignment.note}&rdquo;</span>
      )}
    </li>
  );
}

function ClaimForm({ issueId, orgs }: { issueId: string; orgs: MyOrgMembership[] }) {
  const { t } = useTranslation();
  const [orgId, setOrgId] = useState<string>(orgs[0]?.organization.id ?? '');
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const createMutation = useCreateAssignment(orgId);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await createMutation.mutateAsync({ issueId, note: note.trim() || undefined });
      setNote('');
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('issueClaim.errors.generic'));
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-4 space-y-3 border-t border-slate-100 pt-4">
      <p className="text-sm font-semibold text-slate-800">{t('issueClaim.formHeading')}</p>
      <p className="text-xs text-slate-500">{t('issueClaim.formIntro')}</p>

      {orgs.length > 1 && (
        <div>
          <label className="block text-xs font-semibold text-slate-700">
            {t('issueClaim.actingAs')}
          </label>
          <select
            value={orgId}
            onChange={(e) => setOrgId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          >
            {orgs.map((m) => (
              <option key={m.organization.id} value={m.organization.id}>
                {m.organization.name}
              </option>
            ))}
          </select>
        </div>
      )}

      <div>
        <label className="block text-xs font-semibold text-slate-700">
          {t('issueClaim.noteLabel')}
        </label>
        <Input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={t('issueClaim.notePlaceholder')}
        />
      </div>

      {error && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-700">
          {error}
        </p>
      )}

      <div className="flex justify-end">
        <Button size="sm" type="submit" disabled={createMutation.isPending || !orgId}>
          {createMutation.isPending ? t('issueClaim.claiming') : t('issueClaim.claim')}
        </Button>
      </div>
    </form>
  );
}
