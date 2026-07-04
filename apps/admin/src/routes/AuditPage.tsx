import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '../lib/api';

interface AuditEntry {
  id: string;
  actorId: string;
  actorName: string;
  actorRole: string;
  action: string;
  targetType: string;
  targetId: string;
  metadata: string; // JSON string
  ipAddress?: string | null;
  userAgent?: string | null;
  createdAt: string;
}

interface ListResponse {
  auditLogs: AuditEntry[];
  total: number;
}

export function AuditPage() {
  const [action, setAction] = useState('');
  const [targetType, setTargetType] = useState('');

  const query = useQuery({
    queryKey: ['admin-audit', action, targetType],
    queryFn: () => {
      const params = new URLSearchParams();
      if (action) params.set('action', action);
      if (targetType) params.set('targetType', targetType);
      params.set('limit', '100');
      return apiGet<ListResponse>(`/api/v1/audit-logs?${params.toString()}`);
    },
  });

  const rows = query.data?.auditLogs ?? [];
  const total = query.data?.total ?? 0;

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Governance</p>
        <h1 className="admin-page-title">Audit log</h1>
        <p className="admin-page-sub">
          Every administrative action, in chronological order. Filter by action prefix or target
          type. This log is append-only.
        </p>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Filter by action prefix (e.g. flag., user., org.)…"
            value={action}
            onChange={(e) => setAction(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 200px' }}
            value={targetType}
            onChange={(e) => setTargetType(e.target.value)}
          >
            <option value="">Any target type</option>
            <option value="USER">User</option>
            <option value="CONTENT_FLAG">Content flag</option>
            <option value="ORGANIZATION">Organization</option>
            <option value="ISSUE">Issue</option>
          </select>
          <span className="text-xs text-slate-500 mono">
            {rows.length} of {total.toLocaleString()}
          </span>
        </div>

        {query.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No audit entries match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>When</th>
                <th>Actor</th>
                <th>Action</th>
                <th>Target</th>
                <th>Details</th>
                <th>Source</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className="mono text-xs text-slate-500 whitespace-nowrap">
                    {new Date(r.createdAt).toLocaleString()}
                  </td>
                  <td>
                    <div>{r.actorName}</div>
                    <div className="text-xs text-slate-500">{r.actorRole}</div>
                  </td>
                  <td>
                    <span className="mono text-xs text-civic-700">{r.action}</span>
                  </td>
                  <td>
                    <div className="mono text-xs text-slate-600">{r.targetType}</div>
                    <div className="mono text-xs text-slate-400">{r.targetId.slice(0, 8)}…</div>
                  </td>
                  <td>
                    <MetadataCell raw={r.metadata} />
                  </td>
                  <td className="mono text-xs text-slate-500">{r.ipAddress ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}

function MetadataCell({ raw }: { raw: string }) {
  try {
    const obj = JSON.parse(raw) as Record<string, unknown>;
    const keys = Object.keys(obj);
    if (keys.length === 0) return <span className="text-xs text-slate-400">—</span>;
    return (
      <div className="text-xs text-slate-600 space-y-0.5">
        {keys.map((k) => (
          <div key={k} className="mono">
            <span className="text-slate-500">{k}:</span> {String(obj[k] ?? '')}
          </div>
        ))}
      </div>
    );
  } catch {
    return <span className="mono text-xs text-slate-400">{raw}</span>;
  }
}
