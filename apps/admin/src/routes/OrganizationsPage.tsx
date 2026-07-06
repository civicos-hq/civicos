import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, XCircle } from 'lucide-react';
import { apiGet, apiPatch } from '../lib/api';

interface AdminOrganization {
  id: string;
  name: string;
  slug: string;
  kind: string;
  jurisdiction: string;
  state?: string;
  lga?: string;
  verified: boolean;
  memberCount: number;
  announcementCount: number;
  projectCount: number;
  createdAt: string;
}

interface ListResponse {
  organizations: AdminOrganization[];
}

export function OrganizationsPage() {
  const [q, setQ] = useState('');
  const [showVerifiedOnly, setShowVerifiedOnly] = useState('');
  const queryClient = useQueryClient();

  const orgsQuery = useQuery({
    queryKey: ['admin-orgs', q],
    queryFn: () => {
      const params = new URLSearchParams();
      if (q) params.set('q', q);
      return apiGet<ListResponse>(`/api/v1/organizations?${params.toString()}`);
    },
  });

  const verifyMutation = useMutation({
    mutationFn: ({ id, verified }: { id: string; verified: boolean }) =>
      apiPatch(`/api/v1/organizations/${id}`, { verified }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-orgs'] }),
  });

  const rows = (orgsQuery.data?.organizations ?? []).filter((o) => {
    if (showVerifiedOnly === 'verified') return o.verified;
    if (showVerifiedOnly === 'unverified') return !o.verified;
    return true;
  });

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Organizations</p>
        <h1 className="admin-page-title">Organization management</h1>
        <p className="admin-page-sub">
          Organizations now enter through self-serve applications. Admins review and verify them
          here after approval.
        </p>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search organizations by name…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={showVerifiedOnly}
            onChange={(e) => setShowVerifiedOnly(e.target.value)}
          >
            <option value="">All</option>
            <option value="verified">Verified only</option>
            <option value="unverified">Unverified only</option>
          </select>
          <span className="text-xs text-slate-500 mono">{rows.length} shown</span>
        </div>

        {orgsQuery.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No organizations match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Kind</th>
                <th>Jurisdiction</th>
                <th>Members</th>
                <th>Content</th>
                <th style={{ textAlign: 'right' }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((o) => (
                <tr key={o.id}>
                  <td>
                    <Link
                      to={`/organizations/${o.id}`}
                      className="font-medium text-civic-700 hover:underline"
                    >
                      {o.name}
                    </Link>
                    <div className="mono text-xs text-slate-500">{o.slug}</div>
                  </td>
                  <td>
                    <span className="admin-chip admin-chip-role-CITIZEN">{o.kind}</span>
                  </td>
                  <td>
                    {o.jurisdiction}
                    {o.state ? ` · ${o.state}` : ''}
                    {o.lga ? ` / ${o.lga}` : ''}
                  </td>
                  <td>{o.memberCount}</td>
                  <td className="text-xs text-slate-500">
                    {o.announcementCount} announcements · {o.projectCount} projects
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    {o.verified ? (
                      <button
                        type="button"
                        className="admin-btn admin-btn-secondary admin-btn-sm"
                        onClick={() => {
                          if (confirm(`Revoke verified badge from ${o.name}?`)) {
                            verifyMutation.mutate({ id: o.id, verified: false });
                          }
                        }}
                        disabled={verifyMutation.isPending}
                      >
                        <XCircle className="h-3.5 w-3.5" aria-hidden="true" />
                        Revoke verify
                      </button>
                    ) : (
                      <button
                        type="button"
                        className="admin-btn admin-btn-primary admin-btn-sm"
                        onClick={() => {
                          if (confirm(`Grant verified badge to ${o.name}?`)) {
                            verifyMutation.mutate({ id: o.id, verified: true });
                          }
                        }}
                        disabled={verifyMutation.isPending}
                      >
                        <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
                        Verify
                      </button>
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
