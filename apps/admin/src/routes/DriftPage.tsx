import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2 } from 'lucide-react';
import { apiGet, apiPost } from '../lib/api';

interface Finding {
  id: string;
  kind: string;
  donationId: string;
  campaignId?: string | null;
  reference?: string | null;
  amountMinor: number;
  detail: string;
  runId?: string | null;
  firstSeenAt: string;
  lastSeenAt: string;
  timesSeen: number;
  resolvedAt?: string | null;
  resolvedByName?: string | null;
  resolutionNote?: string | null;
}

interface ListResponse {
  findings: Finding[];
  openCount: number;
}

/**
 * How serious each kind of disagreement is.
 *
 * Ordering the list by severity matters more here than recency: a payment
 * credited to the wrong organization and a row we simply could not check are
 * both "drift", and burying the first under a page of the second is how the
 * one that matters gets missed.
 */
const SEVERITY: Record<string, { rank: number; tone: string; plain: string }> = {
  SUBACCOUNT_MISMATCH: {
    rank: 0,
    tone: 'danger',
    plain: 'Money reached a different organization than the campaign belongs to.',
  },
  AMOUNT_MISMATCH: {
    rank: 1,
    tone: 'danger',
    plain: 'We banked a different figure than Paystack holds. Public totals are wrong.',
  },
  SETTLED_HERE_BUT_NOT_AT_PROVIDER: {
    rank: 2,
    tone: 'danger',
    plain: 'We show this as settled; Paystack does not. A missed reversal, or a ledger bug.',
  },
  CURRENCY_MISMATCH: { rank: 3, tone: 'pending', plain: 'The currencies disagree.' },
  PENDING_WITH_MISMATCHED_DETAILS: {
    rank: 4,
    tone: 'pending',
    plain:
      'Paystack reports success but the details do not match what was opened, so it was NOT settled.',
  },
  PROVIDER_UNREACHABLE: {
    rank: 5,
    tone: 'neutral',
    plain:
      'This row could not be checked at all. Not evidence of a problem — but not evidence of health either.',
  },
};

function severityOf(kind: string) {
  return SEVERITY[kind] ?? { rank: 9, tone: 'neutral', plain: '' };
}

function naira(minor: number) {
  return new Intl.NumberFormat('en-NG', {
    style: 'currency',
    currency: 'NGN',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2,
  }).format(minor / 100);
}

export function DriftPage() {
  const [includeResolved, setIncludeResolved] = useState(false);
  const [actingOn, setActingOn] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['admin-drift', includeResolved],
    queryFn: () =>
      apiGet<ListResponse>(`/api/v1/admin/donations/drift?includeResolved=${includeResolved}`),
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      apiPost(`/api/v1/admin/donations/drift/${id}/resolve`, { note }),
    onSuccess: () => {
      setActingOn(null);
      setNote('');
      setError('');
      queryClient.invalidateQueries({ queryKey: ['admin-drift'] });
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? 'Could not resolve this finding.');
    },
  });

  const findings = [...(query.data?.findings ?? [])].sort(
    (a, b) => severityOf(a.kind).rank - severityOf(b.kind).rank,
  );
  const open = query.data?.openCount ?? 0;
  const worst = findings.filter(
    (f) => !f.resolvedAt && severityOf(f.kind).tone === 'danger',
  ).length;
  const unchecked = findings.filter(
    (f) => !f.resolvedAt && f.kind === 'PROVIDER_UNREACHABLE',
  ).length;

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Money</p>
        <h1 className="admin-page-title">Reconciliation drift</h1>
        <p className="admin-page-sub">
          Donations where our ledger and Paystack disagree. Found by the reconciliation sweep, which
          runs on a timer and can also be run on demand. Nothing here clears itself — drift that
          stops being detected may have been fixed, or may have become invisible, and only a person
          can say which.
        </p>
      </header>

      <div className="grid gap-3 md:grid-cols-3 mb-4">
        <MetricCard label="Open" value={open} tone={open > 0 ? 'pending' : 'success'} />
        <MetricCard
          label="Needs attention now"
          value={worst}
          tone={worst > 0 ? 'danger' : 'success'}
        />
        <MetricCard label="Could not be checked" value={unchecked} tone="neutral" />
      </div>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={includeResolved}
              onChange={(e) => setIncludeResolved(e.target.checked)}
            />
            Show resolved
          </label>
        </div>

        {query.isLoading && <p className="admin-empty">Loading…</p>}

        {!query.isLoading && findings.length === 0 && (
          <p className="admin-empty">
            <CheckCircle2 className="mr-2 inline h-4 w-4" aria-hidden="true" />
            Nothing outstanding. Every settled donation agrees with Paystack.
          </p>
        )}

        {findings.map((f) => {
          const sev = severityOf(f.kind);
          return (
            <article key={f.id} className={`admin-drift admin-drift-${sev.tone}`}>
              <div className="admin-drift-head">
                <span className="admin-drift-kind">
                  {sev.tone === 'danger' && (
                    <AlertTriangle className="mr-1 inline h-4 w-4" aria-hidden="true" />
                  )}
                  {f.kind.replaceAll('_', ' ').toLowerCase()}
                </span>
                <span className="admin-drift-amount">{naira(f.amountMinor)}</span>
              </div>

              {/* What it means, in words. The enum is precise but an operator
                  under pressure should not have to translate it. */}
              {sev.plain && <p className="admin-drift-plain">{sev.plain}</p>}
              <p className="admin-drift-detail">{f.detail}</p>

              <p className="admin-drift-meta">
                donation <code>{f.donationId}</code>
                {f.reference && (
                  <>
                    {' · '}ref <code>{f.reference}</code>
                  </>
                )}
                {' · '}first seen {new Date(f.firstSeenAt).toLocaleString()}
                {f.timesSeen > 1 && ` · seen ${f.timesSeen}×`}
              </p>

              {f.resolvedAt ? (
                <p className="admin-drift-resolved">
                  Resolved by {f.resolvedByName ?? 'unknown'} on{' '}
                  {new Date(f.resolvedAt).toLocaleDateString()}
                  {f.resolutionNote && ` — ${f.resolutionNote}`}
                </p>
              ) : actingOn === f.id ? (
                <div className="admin-drift-resolve">
                  <input
                    className="admin-input"
                    placeholder="What did you find, and what did you do about it?"
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                  />
                  <button
                    type="button"
                    className="admin-btn admin-btn-primary"
                    disabled={note.trim().length < 4 || resolveMutation.isPending}
                    onClick={() => resolveMutation.mutate({ id: f.id, note: note.trim() })}
                  >
                    {resolveMutation.isPending ? 'Saving…' : 'Mark resolved'}
                  </button>
                  <button type="button" className="admin-btn" onClick={() => setActingOn(null)}>
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  className="admin-btn"
                  onClick={() => {
                    setActingOn(f.id);
                    setNote('');
                    setError('');
                  }}
                >
                  Resolve
                </button>
              )}

              {actingOn === f.id && error && <p className="admin-error">{error}</p>}
            </article>
          );
        })}
      </div>
    </>
  );
}

function MetricCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: 'pending' | 'danger' | 'neutral' | 'success';
}) {
  return (
    <div className={`admin-metric-card admin-metric-card-${tone}`}>
      <div className="admin-metric-label">{label}</div>
      <div className="admin-metric-value">{value.toLocaleString()}</div>
    </div>
  );
}
