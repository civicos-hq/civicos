import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { apiGet } from '../lib/api';

interface AdminCommunity {
  id: string;
  name: string;
  slug: string;
  state: string;
  lga: string;
  country: string;
  latitude?: number | null;
  longitude?: number | null;
  createdAt: string;
}

const isLocated = (c: AdminCommunity) =>
  typeof c.latitude === 'number' && typeof c.longitude === 'number';

interface ListResponse {
  communities: AdminCommunity[];
}

export function CommunitiesPage() {
  const [q, setQ] = useState('');
  const [state, setState] = useState('');
  // "Which communities are silently receiving no flood forecasts?" is the
  // question an operator needs to be able to ask, and there was no way to
  // ask it — an unlocated community fails quiet by design.
  const [onlyUnlocated, setOnlyUnlocated] = useState(false);

  const query = useQuery({
    queryKey: ['admin-communities', state],
    queryFn: () => {
      const params = new URLSearchParams();
      if (state) params.set('state', state);
      return apiGet<ListResponse>(`/api/v1/communities?${params.toString()}`);
    },
  });

  const all = query.data?.communities ?? [];
  const rows = all
    .filter((c) =>
      q ? c.name.toLowerCase().includes(q.toLowerCase()) || c.slug.includes(q.toLowerCase()) : true,
    )
    .filter((c) => (onlyUnlocated ? !isLocated(c) : true));

  const unlocatedCount = all.filter((c) => !isLocated(c)).length;

  return (
    <>
      <header className="admin-page-header">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="admin-page-eyebrow">Section — Communities</p>
            <h1 className="admin-page-title">Communities</h1>
            <p className="admin-page-sub">
              Every community on the platform. Click a row to see its per-community breakdown —
              issue volume, petitions, representatives, citizen count.
            </p>
          </div>
          <Link to="/communities/new" className="admin-btn admin-btn-primary">
            <Plus className="h-4 w-4" aria-hidden="true" />
            New community
          </Link>
        </div>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search name or slug…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <input
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            placeholder="Filter by state…"
            value={state}
            onChange={(e) => setState(e.target.value)}
          />
          <label className="flex items-center gap-1.5 text-xs text-slate-600">
            <input
              type="checkbox"
              checked={onlyUnlocated}
              onChange={(e) => setOnlyUnlocated(e.target.checked)}
            />
            No coordinates
          </label>
          <span className="text-xs text-slate-500 mono">{rows.length} shown</span>
        </div>

        {unlocatedCount > 0 && !onlyUnlocated && (
          <div className="border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-900">
            <strong>{unlocatedCount}</strong> of {all.length} communities have no coordinates and
            are excluded from flood forecasts.{' '}
            <button
              type="button"
              className="font-semibold underline"
              onClick={() => setOnlyUnlocated(true)}
            >
              Show them
            </button>
          </div>
        )}

        {query.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No communities match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>State / LGA</th>
                <th>Location</th>
                <th>Slug</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr key={c.id}>
                  <td>
                    <Link
                      to={`/communities/${c.id}`}
                      className="font-medium text-civic-700 hover:underline"
                    >
                      {c.name}
                    </Link>
                  </td>
                  <td>
                    <div>{c.state}</div>
                    <div className="text-xs text-slate-500">{c.lga}</div>
                  </td>
                  <td>
                    {isLocated(c) ? (
                      <span className="mono text-xs text-slate-600">
                        {c.latitude!.toFixed(3)}, {c.longitude!.toFixed(3)}
                      </span>
                    ) : (
                      <Link
                        to={`/communities/${c.id}`}
                        className="text-xs font-medium text-amber-700 hover:underline"
                        title="No coordinates — excluded from flood forecasts"
                      >
                        Not set
                      </Link>
                    )}
                  </td>
                  <td className="mono text-xs text-slate-600">{c.slug}</td>
                  <td className="mono text-xs text-slate-500">
                    {new Date(c.createdAt).toLocaleDateString()}
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
