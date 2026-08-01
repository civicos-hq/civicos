import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Users } from 'lucide-react';
import { apiGet, apiPatch } from '../lib/api';

/**
 * Concerns citizens have raised about funded campaigns.
 *
 * Separate from the moderation queue on purpose. A flag on a comment asks
 * "should this text be visible"; a concern about a campaign asks "is money
 * being taken under false pretences", and the two do not belong in one list
 * competing for the same attention.
 *
 * Grouped by campaign rather than listed by recency, because the number of
 * separate people raising the same concern is the strongest signal available
 * here — CivicOS cannot verify spending, so corroboration between independent
 * observers is most of what there is to go on. Five reports against one
 * campaign and one report each against five are very different situations,
 * and a flat list hides the difference.
 *
 * Nothing on this page can pause a campaign. That is deliberate: pausing
 * stops money moving and must be a separate, deliberate act with its own
 * reason code, taken on the campaign itself. If clearing this queue could
 * pause a fundraiser, a coordinated set of reports would become a way to
 * shut down a rival.
 */

interface Concern {
  id: string;
  contentId: string;
  reporterId: string;
  reporterName: string;
  reason: string;
  description?: string | null;
  status: string;
  resolvedByName?: string | null;
  resolutionNote?: string | null;
  resolvedAt?: string | null;
  createdAt: string;
}

interface ListResponse {
  flags: Concern[];
}

const REASON_TEXT: Record<string, string> = {
  FUNDS_MISUSE: 'Money may be misused',
  WORK_NOT_DONE: 'The work has not been done',
  MISREPRESENTED: 'The campaign misrepresents the situation',
  NO_UPDATES: 'The organization has gone silent',
  OTHER: 'Something else',
};

export function ConcernsPage() {
  const [status, setStatus] = useState('PENDING');
  const [actingOn, setActingOn] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['admin-concerns', status],
    queryFn: () => {
      const params = new URLSearchParams({ contentType: 'CAMPAIGN', limit: '100' });
      if (status) params.set('status', status);
      return apiGet<ListResponse>(`/api/v1/flags?${params.toString()}`);
    },
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, newStatus, note }: { id: string; newStatus: string; note?: string }) =>
      apiPatch(`/api/v1/flags/${id}`, { status: newStatus, resolutionNote: note || undefined }),
    onSuccess: () => {
      setActingOn(null);
      setNote('');
      setError('');
      queryClient.invalidateQueries({ queryKey: ['admin-concerns'] });
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? 'Could not update this concern.');
    },
  });

  const concerns = query.data?.flags ?? [];

  // Group by campaign, then order groups by how many distinct people raised
  // a concern — not by recency.
  const groups = Object.values(
    concerns.reduce<Record<string, { campaignId: string; items: Concern[] }>>((acc, c) => {
      (acc[c.contentId] ??= { campaignId: c.contentId, items: [] }).items.push(c);
      return acc;
    }, {}),
  ).sort((a, b) => distinctReporters(b.items) - distinctReporters(a.items));

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Money</p>
        <h1 className="admin-page-title">Campaign concerns</h1>
        <p className="admin-page-sub">
          Raised by people who donated to a campaign, or who live in the area it serves — nobody
          else can file one. Campaigns are reviewed before they publish and locked afterwards, so
          everything here is about conduct <em>after</em> approval, which CivicOS cannot verify on
          its own. Nothing here pauses a campaign; open the campaign to do that.
        </p>
      </header>

      <div className="grid gap-3 md:grid-cols-3 mb-4">
        <MetricCard
          label="Campaigns with open concerns"
          value={groups.length}
          tone={groups.length > 0 ? 'pending' : 'success'}
        />
        <MetricCard
          label="Raised by 2+ people"
          value={groups.filter((g) => distinctReporters(g.items) > 1).length}
          tone={groups.some((g) => distinctReporters(g.items) > 1) ? 'danger' : 'success'}
        />
        <MetricCard label="Concerns in total" value={concerns.length} tone="neutral" />
      </div>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <select
            className="admin-table-search"
            style={{ flex: '0 0 200px' }}
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="PENDING">Pending</option>
            <option value="REVIEWED">Reviewed</option>
            <option value="DISMISSED">Dismissed</option>
            <option value="">Any status</option>
          </select>
        </div>

        {query.isLoading && <p className="admin-empty">Loading…</p>}

        {!query.isLoading && groups.length === 0 && (
          <p className="admin-empty">
            <CheckCircle2 className="mr-2 inline h-4 w-4" aria-hidden="true" />
            Nothing outstanding.
          </p>
        )}

        {groups.map((g) => {
          const people = distinctReporters(g.items);
          return (
            <article key={g.campaignId} className="admin-drift admin-drift-pending">
              <div className="admin-drift-head">
                <span className="admin-drift-kind">
                  {people > 1 && (
                    <AlertTriangle className="mr-1 inline h-4 w-4" aria-hidden="true" />
                  )}
                  <Users className="mr-1 inline h-4 w-4" aria-hidden="true" />
                  {people === 1 ? '1 person' : `${people} separate people`}
                </span>
                <Link to={`/campaigns/${g.campaignId}`} className="admin-btn admin-btn-sm">
                  Open campaign
                </Link>
              </div>

              {people > 1 && (
                <p className="admin-drift-plain">
                  Independent reports agreeing with each other. CivicOS cannot verify spending, so
                  corroboration between people who are not connected is the strongest signal
                  available.
                </p>
              )}

              {g.items.map((c) => (
                <div key={c.id} className="admin-concern-item">
                  <p className="admin-drift-meta">
                    <strong>{REASON_TEXT[c.reason] ?? c.reason}</strong>
                    {' · '}
                    {c.reporterName}
                    {' · '}
                    {new Date(c.createdAt).toLocaleString()}
                  </p>
                  {/* The citizen's own account, in their words. The reason code
                      is a bucket; this is the part with the detail an
                      investigation can actually start from. */}
                  {c.description && <p className="admin-drift-detail">“{c.description}”</p>}

                  {c.resolvedAt ? (
                    <p className="admin-drift-resolved">
                      {c.status} by {c.resolvedByName ?? 'unknown'} on{' '}
                      {new Date(c.resolvedAt).toLocaleDateString()}
                      {c.resolutionNote && ` — ${c.resolutionNote}`}
                    </p>
                  ) : actingOn === c.id ? (
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
                        onClick={() =>
                          resolveMutation.mutate({
                            id: c.id,
                            newStatus: 'REVIEWED',
                            note: note.trim(),
                          })
                        }
                      >
                        Mark reviewed
                      </button>
                      <button
                        type="button"
                        className="admin-btn"
                        disabled={note.trim().length < 4 || resolveMutation.isPending}
                        onClick={() =>
                          resolveMutation.mutate({
                            id: c.id,
                            newStatus: 'DISMISSED',
                            note: note.trim(),
                          })
                        }
                      >
                        Dismiss
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
                        setActingOn(c.id);
                        setNote('');
                        setError('');
                      }}
                    >
                      Resolve
                    </button>
                  )}

                  {actingOn === c.id && error && <p className="admin-error">{error}</p>}
                </div>
              ))}
            </article>
          );
        })}
      </div>
    </>
  );
}

// One person filing four times is not four people. The dedupe index already
// prevents that per campaign, but counting distinctly keeps the headline
// honest if that ever changes.
function distinctReporters(items: Concern[]): number {
  return new Set(items.map((i) => i.reporterId)).size;
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
