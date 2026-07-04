import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { EyeOff, Undo2 } from 'lucide-react';
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

export function FlagsPage() {
  const [status, setStatus] = useState('PENDING');
  const queryClient = useQueryClient();

  const flagsQuery = useQuery({
    queryKey: ['admin-flags', status],
    queryFn: () => {
      const params = new URLSearchParams();
      if (status) params.set('status', status);
      params.set('limit', '50');
      return apiGet<ListResponse>(`/api/v1/flags?${params.toString()}`);
    },
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, newStatus, note }: { id: string; newStatus: string; note?: string }) =>
      apiPatch(`/api/v1/flags/${id}`, {
        status: newStatus,
        resolutionNote: note,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-flags'] }),
  });

  const rows = flagsQuery.data?.flags ?? [];

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

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
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
                <tr key={f.id}>
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
                          className="admin-btn admin-btn-danger admin-btn-sm"
                          onClick={() => {
                            const note =
                              prompt('Note on hiding this content? (optional)') ?? undefined;
                            resolveMutation.mutate({
                              id: f.id,
                              newStatus: 'HIDDEN',
                              note: note || undefined,
                            });
                          }}
                          disabled={resolveMutation.isPending}
                        >
                          <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />
                          Hide
                        </button>
                        <button
                          type="button"
                          className="admin-btn admin-btn-secondary admin-btn-sm"
                          onClick={() => {
                            const note =
                              prompt('Note on dismissing this flag? (optional)') ?? undefined;
                            resolveMutation.mutate({
                              id: f.id,
                              newStatus: 'DISMISSED',
                              note: note || undefined,
                            });
                          }}
                          disabled={resolveMutation.isPending}
                        >
                          <Undo2 className="h-3.5 w-3.5" aria-hidden="true" />
                          Dismiss
                        </button>
                      </div>
                    ) : (
                      <span className="text-xs text-slate-400 mono">
                        {f.resolvedAt ? new Date(f.resolvedAt).toLocaleDateString() : '—'}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}
