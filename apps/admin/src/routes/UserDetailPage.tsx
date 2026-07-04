import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Ban, CheckCircle2, ShieldCheck } from 'lucide-react';
import { apiGet } from '../lib/api';

interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  emailVerified: boolean;
  communityId?: string | null;
  bannedAt?: string | null;
  banReason?: string | null;
  bannedById?: string | null;
  createdAt: string;
}

interface UserResp {
  user: AdminUser;
}

interface AuditEntry {
  id: string;
  actorName: string;
  actorRole: string;
  action: string;
  targetType: string;
  metadata: string;
  createdAt: string;
}

interface AuditResp {
  auditLogs: AuditEntry[];
  total: number;
}

interface AdminFlag {
  id: string;
  contentType: string;
  contentId: string;
  reason: string;
  description?: string | null;
  status: string;
  resolvedByName?: string | null;
  createdAt: string;
}

interface FlagsResp {
  flags: AdminFlag[];
}

export function UserDetailPage() {
  const { id = '' } = useParams<{ id: string }>();

  const userQuery = useQuery({
    queryKey: ['admin-user', id],
    queryFn: () => apiGet<UserResp>(`/api/v1/users/${id}`),
    enabled: Boolean(id),
  });

  // Audit rows targeting this user — bans, role changes, etc.
  const auditQuery = useQuery({
    queryKey: ['admin-user-audit', id],
    queryFn: () => apiGet<AuditResp>(`/api/v1/audit-logs?targetId=${id}&limit=50`),
    enabled: Boolean(id),
  });

  // Flags this user has filed as reporter.
  const flagsQuery = useQuery({
    queryKey: ['admin-user-flags', id],
    queryFn: () => apiGet<FlagsResp>(`/api/v1/flags?reporterId=${id}&limit=50`),
    enabled: Boolean(id),
  });

  if (userQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (userQuery.isError || !userQuery.data) {
    return (
      <>
        <BackLink />
        <p className="text-sm text-red-700">Couldn't load this user.</p>
      </>
    );
  }

  const u = userQuery.data.user;
  const audit = auditQuery.data?.auditLogs ?? [];
  const flags = flagsQuery.data?.flags ?? [];

  return (
    <>
      <BackLink />
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — User detail</p>
        <h1 className="admin-page-title">{u.name}</h1>
        <p className="admin-page-sub">
          <span className="mono">{u.email}</span>
        </p>
      </header>

      {/* Identity card */}
      <div
        className="admin-stat-grid"
        style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))' }}
      >
        <div className="admin-stat-card">
          <p className="admin-stat-label">Role</p>
          <p className="admin-stat-value" style={{ fontSize: '1rem' }}>
            <span className={`admin-chip admin-chip-role-${u.role}`}>{u.role}</span>
          </p>
        </div>
        <div className="admin-stat-card">
          <p className="admin-stat-label">Email verified</p>
          <p className="admin-stat-value" style={{ fontSize: '1rem' }}>
            {u.emailVerified ? (
              <span className="admin-chip admin-chip-verified">
                <CheckCircle2 className="h-3 w-3" aria-hidden="true" /> verified
              </span>
            ) : (
              <span className="text-sm text-slate-500">not verified</span>
            )}
          </p>
        </div>
        <div className="admin-stat-card">
          <p className="admin-stat-label">Account status</p>
          <p className="admin-stat-value" style={{ fontSize: '1rem' }}>
            {u.bannedAt ? (
              <span className="admin-chip admin-chip-banned">
                <Ban className="h-3 w-3" aria-hidden="true" /> Banned
              </span>
            ) : (
              <span className="admin-chip admin-chip-verified">
                <ShieldCheck className="h-3 w-3" aria-hidden="true" /> Active
              </span>
            )}
          </p>
          {u.banReason && <p className="admin-stat-sub">{u.banReason}</p>}
        </div>
        <div className="admin-stat-card">
          <p className="admin-stat-label">Joined</p>
          <p className="admin-stat-value" style={{ fontSize: '1rem' }}>
            {new Date(u.createdAt).toLocaleDateString()}
          </p>
          <p className="admin-stat-sub">{new Date(u.createdAt).toLocaleString()}</p>
        </div>
      </div>

      {/* Audit trail against this user */}
      <section style={{ marginTop: '2rem' }}>
        <h2 className="text-base font-semibold text-slate-800 mb-2">
          Actions taken against this user{' '}
          <span className="mono text-xs text-slate-500">({auditQuery.data?.total ?? 0})</span>
        </h2>
        <div className="admin-table-shell">
          {audit.length === 0 ? (
            <div className="admin-empty">
              No administrative actions recorded against this user yet.
            </div>
          ) : (
            <table className="admin-table">
              <thead>
                <tr>
                  <th>When</th>
                  <th>By</th>
                  <th>Action</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {audit.map((r) => (
                  <tr key={r.id}>
                    <td className="mono text-xs text-slate-500 whitespace-nowrap">
                      {new Date(r.createdAt).toLocaleString()}
                    </td>
                    <td>
                      {r.actorName}
                      <div className="text-xs text-slate-500">{r.actorRole}</div>
                    </td>
                    <td>
                      <span className="mono text-xs text-civic-700">{r.action}</span>
                    </td>
                    <td>
                      <MetadataCell raw={r.metadata} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>

      {/* Flags this user has filed */}
      <section style={{ marginTop: '2rem' }}>
        <h2 className="text-base font-semibold text-slate-800 mb-2">
          Reports filed by this user{' '}
          <span className="mono text-xs text-slate-500">({flags.length})</span>
        </h2>
        <div className="admin-table-shell">
          {flags.length === 0 ? (
            <div className="admin-empty">This user hasn't filed any moderation reports.</div>
          ) : (
            <table className="admin-table">
              <thead>
                <tr>
                  <th>When</th>
                  <th>Reason</th>
                  <th>Content</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {flags.map((f) => (
                  <tr key={f.id}>
                    <td className="mono text-xs text-slate-500 whitespace-nowrap">
                      {new Date(f.createdAt).toLocaleString()}
                    </td>
                    <td>
                      <div className="font-medium">{f.reason}</div>
                      {f.description && (
                        <div className="text-xs text-slate-500 mt-1">{f.description}</div>
                      )}
                    </td>
                    <td>
                      <div className="mono text-xs text-slate-600">{f.contentType}</div>
                      <div className="mono text-xs text-slate-400">{f.contentId.slice(0, 8)}…</div>
                    </td>
                    <td>
                      <span className={`admin-chip admin-chip-status-${f.status}`}>{f.status}</span>
                      {f.resolvedByName && (
                        <div className="text-xs text-slate-500 mt-1">by {f.resolvedByName}</div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </>
  );
}

function BackLink() {
  return (
    <Link
      to="/users"
      className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden="true" />
      Back to Users
    </Link>
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
