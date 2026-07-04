import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft } from 'lucide-react';
import { apiGet } from '../lib/api';

interface Community {
  id: string;
  name: string;
  slug: string;
  state: string;
  lga: string;
  country: string;
  description?: string | null;
  createdAt: string;
}

interface CommunityStats {
  citizenCount: number;
  issueTotal: number;
  issuesByStatus: Record<string, number>;
  petitionTotal: number;
  representativeTotal: number;
}

export function CommunityDetailPage() {
  const { id = '' } = useParams<{ id: string }>();

  const communityQuery = useQuery({
    queryKey: ['admin-community', id],
    queryFn: () => apiGet<{ community: Community }>(`/api/v1/communities/${id}`),
    enabled: Boolean(id),
  });

  const statsQuery = useQuery({
    queryKey: ['admin-community-stats', id],
    queryFn: () => apiGet<{ stats: CommunityStats }>(`/api/v1/admin/communities/${id}/stats`),
    enabled: Boolean(id),
  });

  if (communityQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (communityQuery.isError || !communityQuery.data) {
    return (
      <>
        <BackLink />
        <p className="text-sm text-red-700">Couldn't load this community.</p>
      </>
    );
  }

  const c = communityQuery.data.community;
  const s = statsQuery.data?.stats;

  return (
    <>
      <BackLink />
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Community detail</p>
        <h1 className="admin-page-title">{c.name}</h1>
        <p className="admin-page-sub">
          <span className="mono">{c.slug}</span> · {c.state} / {c.lga} · {c.country}
        </p>
        {c.description && <p className="mt-2 text-sm text-slate-600">{c.description}</p>}
      </header>

      <h2
        className="text-xs font-semibold text-slate-500 mono mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        PARTICIPATION
      </h2>
      <section className="admin-stat-grid" aria-label="Community counters">
        <StatCard label="Citizens" value={s?.citizenCount ?? 0} sub="Joined this community" />
        <StatCard label="Issues" value={s?.issueTotal ?? 0} sub="All time" />
        <StatCard label="Petitions" value={s?.petitionTotal ?? 0} sub="All time" />
        <StatCard
          label="Representatives"
          value={s?.representativeTotal ?? 0}
          sub="On record here"
        />
      </section>

      {s?.issuesByStatus && Object.keys(s.issuesByStatus).length > 0 && (
        <>
          <h2
            className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
            style={{ letterSpacing: '0.16em' }}
          >
            ISSUES BY STATUS
          </h2>
          <section className="admin-stat-grid">
            {(['OPEN', 'UNDER_REVIEW', 'IN_PROGRESS', 'RESOLVED', 'CLOSED'] as const).map((k) => (
              <StatCard
                key={k}
                label={k.replace(/_/g, ' ')}
                value={s.issuesByStatus[k] ?? 0}
                sub={pctOf(s.issuesByStatus[k] ?? 0, s.issueTotal)}
              />
            ))}
          </section>
        </>
      )}

      <p className="mt-6 text-xs text-slate-500 mono" style={{ letterSpacing: '0.14em' }}>
        Created {new Date(c.createdAt).toLocaleString()}
      </p>
    </>
  );
}

function pctOf(n: number, total: number): string {
  if (total <= 0) return '—';
  return `${Math.round((n * 100) / total)}%`;
}

function BackLink() {
  return (
    <Link
      to="/communities"
      className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden="true" />
      Back to Communities
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
