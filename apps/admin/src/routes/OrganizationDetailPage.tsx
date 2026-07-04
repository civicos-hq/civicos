import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, ShieldCheck } from 'lucide-react';
import { apiGet } from '../lib/api';

interface Organization {
  id: string;
  name: string;
  slug: string;
  kind: string;
  jurisdiction: string;
  state?: string;
  lga?: string;
  description?: string;
  email?: string;
  phone?: string;
  website?: string;
  verified: boolean;
  memberCount: number;
  announcementCount: number;
  projectCount: number;
  assignmentCount: number;
  createdAt: string;
  updatedAt: string;
}

interface OrgMember {
  id: string;
  userId: string;
  userName: string;
  userRole: string;
  role: string;
  joinedAt: string;
}

export function OrganizationDetailPage() {
  const { id = '' } = useParams<{ id: string }>();

  const orgQuery = useQuery({
    queryKey: ['admin-org', id],
    queryFn: () => apiGet<{ organization: Organization }>(`/api/v1/organizations/${id}`),
    enabled: Boolean(id),
  });

  const membersQuery = useQuery({
    queryKey: ['admin-org-members', id],
    queryFn: () => apiGet<{ members: OrgMember[] }>(`/api/v1/organizations/${id}/members`),
    enabled: Boolean(id),
  });

  if (orgQuery.isLoading) return <p className="text-sm text-slate-500">Loading…</p>;
  if (orgQuery.isError || !orgQuery.data) {
    return (
      <>
        <BackLink />
        <p className="text-sm text-red-700">Couldn't load this organization.</p>
      </>
    );
  }

  const o = orgQuery.data.organization;
  const members = membersQuery.data?.members ?? [];

  return (
    <>
      <BackLink />
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Organization detail</p>
        <h1 className="admin-page-title inline-flex items-center gap-2">
          {o.name}
          {o.verified && (
            <ShieldCheck className="h-5 w-5 text-emerald-600" aria-label="Verified organization" />
          )}
        </h1>
        <p className="admin-page-sub">
          <span className="mono">{o.slug}</span> · {o.kind} · {o.jurisdiction}
          {o.state ? ` · ${o.state}` : ''}
          {o.lga ? ` / ${o.lga}` : ''}
        </p>
        {o.description && <p className="mt-2 text-sm text-slate-600">{o.description}</p>}
      </header>

      <h2
        className="text-xs font-semibold text-slate-500 mono mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        ACTIVITY
      </h2>
      <section className="admin-stat-grid" aria-label="Organization activity">
        <StatCard
          label="Members"
          value={o.memberCount}
          sub={o.verified ? 'Verified organization' : 'Unverified'}
        />
        <StatCard label="Announcements" value={o.announcementCount} sub="Published to citizens" />
        <StatCard label="Projects" value={o.projectCount} sub="Managed by this org" />
        <StatCard
          label="Reports received"
          value={o.assignmentCount}
          sub="Citizen issues assigned here"
        />
      </section>

      {(o.email || o.phone || o.website) && (
        <>
          <h2
            className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
            style={{ letterSpacing: '0.16em' }}
          >
            CONTACT
          </h2>
          <div className="flex flex-wrap gap-3 text-sm">
            {o.email && (
              <a href={`mailto:${o.email}`} className="text-civic-700 hover:underline mono">
                {o.email}
              </a>
            )}
            {o.phone && <span className="mono text-slate-700">{o.phone}</span>}
            {o.website && (
              <a
                href={o.website}
                target="_blank"
                rel="noopener noreferrer"
                className="text-civic-700 hover:underline mono"
              >
                {o.website.replace(/^https?:\/\//, '')}
              </a>
            )}
          </div>
        </>
      )}

      <h2
        className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        MEMBERS ({members.length})
      </h2>
      <div className="admin-table-shell">
        {membersQuery.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : members.length === 0 ? (
          <div className="admin-empty">No members yet.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>User</th>
                <th>Role in org</th>
                <th>Platform role</th>
                <th>Joined</th>
              </tr>
            </thead>
            <tbody>
              {members.map((m) => (
                <tr key={m.id}>
                  <td>
                    <Link to={`/users/${m.userId}`} className="text-civic-700 hover:underline">
                      {m.userName}
                    </Link>
                  </td>
                  <td>
                    <span className="admin-chip admin-chip-role-CITIZEN">{m.role}</span>
                  </td>
                  <td>
                    <span className={`admin-chip admin-chip-role-${m.userRole}`}>{m.userRole}</span>
                  </td>
                  <td className="mono text-xs text-slate-500">
                    {new Date(m.joinedAt).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <p className="mt-6 text-xs text-slate-500 mono" style={{ letterSpacing: '0.14em' }}>
        Created {new Date(o.createdAt).toLocaleString()} · Last updated{' '}
        {new Date(o.updatedAt).toLocaleString()}
      </p>
    </>
  );
}

function BackLink() {
  return (
    <Link
      to="/organizations"
      className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden="true" />
      Back to Organizations
    </Link>
  );
}

function StatCard({ label, value, sub }: { label: string; value: number; sub: string }) {
  return (
    <article className="admin-stat-card">
      <p className="admin-stat-label">{label}</p>
      <p className="admin-stat-value">{value.toLocaleString()}</p>
      <p className="admin-stat-sub">{sub}</p>
    </article>
  );
}
