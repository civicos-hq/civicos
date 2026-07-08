import { Fragment, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { EyeOff, Shield, Undo2 } from 'lucide-react';
import { apiGet, apiPatch } from '../lib/api';

interface AdminFlag {
  id: string;
  contentType: string;
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
  flags: AdminFlag[];
}

interface CountsResponse {
  counts: Record<string, number>;
}

export function FlagsPage() {
  const [status, setStatus] = useState('PENDING');
  const [contentType, setContentType] = useState('');
  const [reason, setReason] = useState('');
  const [q, setQ] = useState('');
  const [actingOn, setActingOn] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const queryClient = useQueryClient();

  const flagsQuery = useQuery({
    queryKey: ['admin-flags', status, contentType, reason, q],
    queryFn: () => {
      const params = new URLSearchParams();
      if (status) params.set('status', status);
      if (contentType) params.set('contentType', contentType);
      if (reason) params.set('reason', reason);
      if (q) params.set('q', q);
      params.set('limit', '50');
      return apiGet<ListResponse>(`/api/v1/flags?${params.toString()}`);
    },
  });

  const countsQuery = useQuery({
    queryKey: ['admin-flag-counts'],
    queryFn: () => apiGet<CountsResponse>('/api/v1/flags/counts'),
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, newStatus, note }: { id: string; newStatus: string; note?: string }) =>
      apiPatch(`/api/v1/flags/${id}`, {
        status: newStatus,
        resolutionNote: note,
      }),
    onSuccess: () => {
      setActingOn(null);
      setNote('');
      setError('');
      queryClient.invalidateQueries({ queryKey: ['admin-flags'] });
      queryClient.invalidateQueries({ queryKey: ['admin-flag-counts'] });
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? 'Could not resolve this flag.');
    },
  });

  const rows = flagsQuery.data?.flags ?? [];
  const counts = countsQuery.data?.counts ?? {};

  function startAction(id: string) {
    setActingOn(id);
    setNote('');
    setError('');
  }

  function submitAction(id: string, newStatus: string) {
    setError('');
    resolveMutation.mutate({ id, newStatus, note: note.trim() || undefined });
  }

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Moderation</p>
        <h1 className="admin-page-title">Moderation queue</h1>
        <p className="admin-page-sub">
          Content reported by citizens. Hide removes it from public view; dismiss keeps it visible.
          Both decisions are recorded to the audit log.
        </p>
      </header>

      <div className="grid gap-3 md:grid-cols-4 mb-4">
        <MetricCard label="Pending" value={counts.PENDING ?? 0} tone="pending" />
        <MetricCard label="Hidden" value={counts.HIDDEN ?? 0} tone="danger" />
        <MetricCard label="Dismissed" value={counts.DISMISSED ?? 0} tone="neutral" />
        <MetricCard label="Reviewed" value={counts.REVIEWED ?? 0} tone="success" />
      </div>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search reporter, note, or content ID…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">Any status</option>
            <option value="PENDING">Pending</option>
            <option value="HIDDEN">Hidden</option>
            <option value="DISMISSED">Dismissed</option>
            <option value="REVIEWED">Reviewed</option>
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={contentType}
            onChange={(e) => setContentType(e.target.value)}
          >
            <option value="">Any content type</option>
            <option value="ISSUE">Issue</option>
            <option value="ISSUE_COMMENT">Issue comment</option>
            <option value="PETITION">Petition</option>
            <option value="PETITION_COMMENT">Petition comment</option>
            <option value="REPRESENTATIVE_COMMENT">Representative comment</option>
            <option value="ANNOUNCEMENT">Announcement</option>
            <option value="PROGRESS_UPDATE">Progress update</option>
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 160px' }}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          >
            <option value="">Any reason</option>
            <option value="SPAM">Spam</option>
            <option value="ABUSE">Abuse</option>
            <option value="MISINFO">Misinfo</option>
            <option value="HATE">Hate</option>
            <option value="OTHER">Other</option>
          </select>
          <span className="text-xs text-slate-500 mono">
            {rows.length} shown ({status || 'any'})
          </span>
        </div>

        {flagsQuery.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">Nothing in the queue with this status.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Reason</th>
                <th>Content</th>
                <th>Reporter</th>
                <th>Status</th>
                <th>Filed</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((f) => (
                <Fragment key={f.id}>
                  <tr>
                    <td>
                      <div className="font-semibold">{f.reason}</div>
                      {f.description && (
                        <div className="text-xs text-slate-500 mt-1">{f.description}</div>
                      )}
                    </td>
                    <td>
                      <div className="mono text-xs text-slate-600">{f.contentType}</div>
                      <div className="mono text-xs text-slate-400">{f.contentId.slice(0, 8)}…</div>
                    </td>
                    <td>{f.reporterName}</td>
                    <td>
                      <span className={`admin-chip admin-chip-status-${f.status}`}>{f.status}</span>
                      {f.resolvedByName && (
                        <div className="text-xs text-slate-500 mt-1">
                          by {f.resolvedByName}
                          {f.resolutionNote ? ` · ${f.resolutionNote}` : ''}
                        </div>
                      )}
                    </td>
                    <td className="mono text-xs text-slate-500">
                      {new Date(f.createdAt).toLocaleString()}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      {f.status === 'PENDING' ? (
                        <div className="flex items-center gap-2 justify-end">
                          <button
                            type="button"
                            className="admin-btn admin-btn-secondary admin-btn-sm"
                            onClick={() => startAction(f.id)}
                            disabled={resolveMutation.isPending}
                          >
                            <Shield className="h-3.5 w-3.5" aria-hidden="true" />
                            Review
                          </button>
                        </div>
                      ) : (
                        <span className="text-xs text-slate-400 mono">
                          {f.resolvedAt ? new Date(f.resolvedAt).toLocaleDateString() : '—'}
                        </span>
                      )}
                    </td>
                  </tr>
                  {actingOn === f.id && (
                    <tr>
                      <td colSpan={6} style={{ background: '#f8fafc' }}>
                        <div className="space-y-3">
                          <div className="text-sm font-semibold text-slate-900">
                            Resolve flag for {f.contentType}
                          </div>
                          <textarea
                            className="admin-table-search"
                            rows={3}
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            placeholder="Why are you hiding or dismissing this flag? A note is required for hide."
                          />
                          {error && <div className="admin-login-error">{error}</div>}
                          <div className="flex items-center justify-end gap-2">
                            <button
                              type="button"
                              className="admin-btn admin-btn-secondary admin-btn-sm"
                              onClick={() => setActingOn(null)}
                              disabled={resolveMutation.isPending}
                            >
                              Cancel
                            </button>
                            <button
                              type="button"
                              className="admin-btn admin-btn-secondary admin-btn-sm"
                              onClick={() => submitAction(f.id, 'DISMISSED')}
                              disabled={resolveMutation.isPending}
                            >
                              <Undo2 className="h-3.5 w-3.5" aria-hidden="true" />
                              Dismiss
                            </button>
                            <button
                              type="button"
                              className="admin-btn admin-btn-danger admin-btn-sm"
                              onClick={() => submitAction(f.id, 'HIDDEN')}
                              disabled={resolveMutation.isPending}
                            >
                              <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />
                              Hide
                            </button>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        )}
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
