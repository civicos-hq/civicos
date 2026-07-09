import { useMemo, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import type { MyOrgMembership } from '@civicos/types';
import { getApiError } from '../../lib/api';
import { useMyOrganizations } from '../../hooks/useConsultations';
import { useCreateProgressUpdate } from '../../hooks/useProgressUpdates';

/**
 * A shared "Post progress update" form used on both issue and project
 * detail pages. Renders only for users who are OWNER/ADMIN of at least
 * one org. When more than one org qualifies, an "acting as" picker is
 * shown so the update is attributed correctly.
 *
 * The `filterOrgIds` prop lets callers restrict to a specific set of
 * orgs — e.g. on an issue page we only want to show the form for orgs
 * that have actually claimed the issue, not every org the user admins.
 */
export function PostProgressUpdateForm({
  target,
  filterOrgIds,
}: {
  target: { issueId?: string; projectId?: string };
  filterOrgIds?: string[];
}) {
  const { t } = useTranslation();
  const { data: memberships = [] } = useMyOrganizations();

  const eligibleOrgs: MyOrgMembership[] = useMemo(() => {
    const filterSet = filterOrgIds ? new Set(filterOrgIds) : null;
    return memberships.filter((m) => {
      const isAdmin = m.membership.role === 'OWNER' || m.membership.role === 'ADMIN';
      if (!isAdmin) return false;
      if (filterSet && !filterSet.has(m.organization.id)) return false;
      return true;
    });
  }, [memberships, filterOrgIds]);

  const [orgId, setOrgId] = useState<string>('');
  const [body, setBody] = useState('');
  const [isPublic, setIsPublic] = useState(true);
  const [error, setError] = useState('');
  const createMutation = useCreateProgressUpdate(orgId || eligibleOrgs[0]?.organization.id);

  if (eligibleOrgs.length === 0) return null;

  // Default the picker to the first eligible org on first render. This
  // avoids the form submitting with orgId="" when the user has exactly
  // one option and never touches the select.
  const effectiveOrgId = orgId || eligibleOrgs[0].organization.id;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await createMutation.mutateAsync({
        ...target,
        body,
        isPublic,
      });
      setBody('');
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('progressUpdate.errors.generic'));
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="mt-4 space-y-3 rounded-2xl border border-slate-200 bg-slate-50 p-4"
    >
      <p className="text-sm font-semibold text-slate-800">{t('progressUpdate.heading')}</p>
      <p className="text-xs text-slate-500">{t('progressUpdate.intro')}</p>

      {eligibleOrgs.length > 1 && (
        <div>
          <label className="block text-xs font-semibold text-slate-700">
            {t('progressUpdate.actingAs')}
          </label>
          <select
            value={effectiveOrgId}
            onChange={(e) => setOrgId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          >
            {eligibleOrgs.map((m) => (
              <option key={m.organization.id} value={m.organization.id}>
                {m.organization.name}
              </option>
            ))}
          </select>
        </div>
      )}

      <div>
        <label className="block text-xs font-semibold text-slate-700">
          {t('progressUpdate.bodyLabel')}
        </label>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          required
          minLength={2}
          rows={3}
          className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          placeholder={t('progressUpdate.bodyPlaceholder')}
        />
      </div>

      <label className="flex items-center gap-2 text-xs text-slate-700">
        <input
          type="checkbox"
          checked={isPublic}
          onChange={(e) => setIsPublic(e.target.checked)}
          className="rounded text-civic-600 focus:ring-civic-500"
        />
        {t('progressUpdate.publicLabel')}
      </label>

      {error && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-700">
          {error}
        </p>
      )}

      <div className="flex justify-end">
        <Button size="sm" type="submit" disabled={createMutation.isPending}>
          {createMutation.isPending ? t('progressUpdate.posting') : t('progressUpdate.post')}
        </Button>
      </div>
    </form>
  );
}
