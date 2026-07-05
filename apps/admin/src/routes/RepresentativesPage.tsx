import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { apiGet } from '../lib/api';

interface AdminRepresentative {
  id: string;
  name: string;
  title: string;
  position: string;
  constituency: string;
  party?: string;
  communityId: string;
  followerCount: number;
  commentCount: number;
  responseRate: number;
  createdAt: string;
}

interface CommunityLite {
  id: string;
  name: string;
  state: string;
  lga: string;
}

export function RepresentativesPage() {
  const [q, setQ] = useState('');
  const [communityId, setCommunityId] = useState('');

  const communitiesQuery = useQuery({
    queryKey: ['admin-communities-lite'],
    queryFn: () => apiGet<{ communities: CommunityLite[] }>('/api/v1/communities'),
  });

  const repsQuery = useQuery({
    queryKey: ['admin-reps', communityId],
    queryFn: () => {
      const params = new URLSearchParams();
      if (communityId) params.set('communityId', communityId);
      return apiGet<{ representatives: AdminRepresentative[] }>(
        `/api/v1/representatives?${params.toString()}`,
      );
    },
  });

  const communities = communitiesQuery.data?.communities ?? [];
  const communityById = new Map(communities.map((c) => [c.id, c]));

  const rows = (repsQuery.data?.representatives ?? []).filter((r) =>
    q
      ? r.name.toLowerCase().includes(q.toLowerCase()) ||
        r.constituency.toLowerCase().includes(q.toLowerCase()) ||
        r.position.toLowerCase().includes(q.toLowerCase())
      : true,
  );

  return (
    <>
      <header className="admin-page-header">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="admin-page-eyebrow">Section — Representatives</p>
            <h1 className="admin-page-title">Elected officials on record</h1>
            <p className="admin-page-sub">
              Every representative is tied to a community and appears on the citizen app under{' '}
              <span className="mono">/representatives</span>. Citizens can follow them, comment
              publicly, and see their response rate.
            </p>
          </div>
          <Link to="/representatives/new" className="admin-btn admin-btn-primary">
            <Plus className="h-4 w-4" aria-hidden="true" />
            New representative
          </Link>
        </div>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search by name, position, or constituency…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 240px' }}
            value={communityId}
            onChange={(e) => setCommunityId(e.target.value)}
          >
            <option value="">All communities</option>
            {communities.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.state})
              </option>
            ))}
          </select>
          <span className="text-xs text-slate-500 mono">{rows.length} shown</span>
        </div>

        {repsQuery.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">
            No representatives{communityId ? ' in this community' : ' on the platform yet'}.
          </div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Position</th>
                <th>Constituency</th>
                <th>Community</th>
                <th>Followers</th>
                <th>Response rate</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const community = communityById.get(r.communityId);
                return (
                  <tr key={r.id}>
                    <td>
                      <div className="font-medium text-slate-900">{r.name}</div>
                      <div className="text-xs text-slate-500">{r.title}</div>
                    </td>
                    <td>
                      {r.position}
                      {r.party ? (
                        <div className="text-xs text-slate-500 mono">{r.party}</div>
                      ) : null}
                    </td>
                    <td>{r.constituency}</td>
                    <td>
                      {community ? (
                        <Link
                          to={`/communities/${community.id}`}
                          className="text-civic-700 hover:underline"
                        >
                          {community.name}
                        </Link>
                      ) : (
                        <span className="text-xs text-slate-400 mono">unknown</span>
                      )}
                    </td>
                    <td>{r.followerCount}</td>
                    <td>{r.responseRate}%</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}
