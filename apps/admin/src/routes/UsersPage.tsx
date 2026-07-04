import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Ban, Undo2 } from 'lucide-react';
import { apiGet, apiPatch, apiPost } from '../lib/api';

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

interface ListResponse {
  users: AdminUser[];
  total: number;
}

const ROLES = [
  'CITIZEN',
  'REPRESENTATIVE',
  'GOVERNMENT_ADMIN',
  'NGO',
  'MODERATOR',
  'PLATFORM_ADMIN',
];

export function UsersPage() {
  const [q, setQ] = useState('');
  const [role, setRole] = useState('');
  const [banned, setBanned] = useState('');
  const queryClient = useQueryClient();

  const usersQuery = useQuery({
    queryKey: ['admin-users', q, role, banned],
    queryFn: () => {
      const params = new URLSearchParams();
      if (q) params.set('q', q);
      if (role) params.set('role', role);
      if (banned) params.set('banned', banned);
      params.set('limit', '50');
      return apiGet<ListResponse>(`/api/v1/users?${params.toString()}`);
    },
  });

  const changeRoleMutation = useMutation({
    mutationFn: ({ id, newRole }: { id: string; newRole: string }) =>
      apiPatch(`/api/v1/users/${id}/role`, { role: newRole }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-users'] }),
  });

  const banMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      apiPost(`/api/v1/users/${id}/ban`, { reason }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-users'] }),
  });

  const unbanMutation = useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/users/${id}/unban`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-users'] }),
  });

  const rows = usersQuery.data?.users ?? [];
  const total = usersQuery.data?.total ?? 0;

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Users</p>
        <h1 className="admin-page-title">User management</h1>
        <p className="admin-page-sub">
          Search accounts, change roles, and ban/unban users. Every action is written to the audit
          log.
        </p>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search email or name…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={role}
            onChange={(e) => setRole(e.target.value)}
          >
            <option value="">Any role</option>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 140px' }}
            value={banned}
            onChange={(e) => setBanned(e.target.value)}
          >
            <option value="">Any status</option>
            <option value="false">Active</option>
            <option value="true">Banned</option>
          </select>
          <span className="text-xs text-slate-500 mono">{total.toLocaleString()} matches</span>
        </div>

        {usersQuery.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No users match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Email</th>
                <th>Name</th>
                <th>Role</th>
                <th>Status</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((u) => (
                <tr key={u.id}>
                  <td>
                    <Link
                      to={`/users/${u.id}`}
                      className="mono text-slate-700 hover:text-civic-700 hover:underline"
                    >
                      {u.email}
                    </Link>
                    {u.emailVerified && (
                      <span className="admin-chip admin-chip-verified ml-2">verified</span>
                    )}
                  </td>
                  <td>{u.name}</td>
                  <td>
                    <span className={`admin-chip admin-chip-role-${u.role}`}>{u.role}</span>
                  </td>
                  <td>
                    {u.bannedAt ? (
                      <span className="admin-chip admin-chip-banned">
                        Banned {u.banReason ? `· ${u.banReason}` : ''}
                      </span>
                    ) : (
                      <span className="text-xs text-slate-500">active</span>
                    )}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <RowActions
                      user={u}
                      onRoleChange={(newRole) => changeRoleMutation.mutate({ id: u.id, newRole })}
                      onBan={(reason) => banMutation.mutate({ id: u.id, reason })}
                      onUnban={() => unbanMutation.mutate(u.id)}
                      isPending={
                        changeRoleMutation.isPending ||
                        banMutation.isPending ||
                        unbanMutation.isPending
                      }
                    />
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

function RowActions({
  user,
  onRoleChange,
  onBan,
  onUnban,
  isPending,
}: {
  user: AdminUser;
  onRoleChange: (role: string) => void;
  onBan: (reason?: string) => void;
  onUnban: () => void;
  isPending: boolean;
}) {
  return (
    <div className="flex items-center gap-2 justify-end">
      <select
        aria-label={`Change role for ${user.email}`}
        className="admin-table-search"
        style={{ padding: '0.3rem 0.6rem', fontSize: '0.78rem' }}
        value={user.role}
        onChange={(e) => {
          if (e.target.value !== user.role) {
            if (confirm(`Change role from ${user.role} to ${e.target.value}?`)) {
              onRoleChange(e.target.value);
            }
          }
        }}
        disabled={isPending}
      >
        {ROLES.map((r) => (
          <option key={r} value={r}>
            {r}
          </option>
        ))}
      </select>
      {user.bannedAt ? (
        <button
          type="button"
          className="admin-btn admin-btn-secondary admin-btn-sm"
          onClick={onUnban}
          disabled={isPending}
        >
          <Undo2 className="h-3.5 w-3.5" aria-hidden="true" />
          Unban
        </button>
      ) : (
        <button
          type="button"
          className="admin-btn admin-btn-danger admin-btn-sm"
          onClick={() => {
            const reason = prompt('Reason for ban? (optional)') ?? undefined;
            if (confirm(`Ban ${user.email}?`)) onBan(reason || undefined);
          }}
          disabled={isPending}
        >
          <Ban className="h-3.5 w-3.5" aria-hidden="true" />
          Ban
        </button>
      )}
    </div>
  );
}
