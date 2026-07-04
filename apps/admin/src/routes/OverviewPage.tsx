import { useQuery } from '@tanstack/react-query';
import { apiGet } from '../lib/api';

const SERVICES = [
  { name: 'gateway', url: 'http://localhost:3000/health' },
  { name: 'identity', url: 'http://localhost:3001/health' },
  { name: 'community', url: 'http://localhost:3002/health' },
  { name: 'organization', url: 'http://localhost:3003/health' },
];

interface HealthResult {
  name: string;
  status: 'ok' | 'down' | 'unknown';
}

async function pingServices(): Promise<HealthResult[]> {
  const results = await Promise.all(
    SERVICES.map(async (svc) => {
      try {
        const res = await fetch(svc.url);
        if (!res.ok) return { name: svc.name, status: 'down' as const };
        const body = (await res.json()) as { status?: string };
        return {
          name: svc.name,
          status: body.status === 'ok' ? ('ok' as const) : ('down' as const),
        };
      } catch {
        return { name: svc.name, status: 'down' as const };
      }
    }),
  );
  return results;
}

function useHealth() {
  return useQuery({
    queryKey: ['admin-health'],
    queryFn: pingServices,
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

interface PlatformMetrics {
  users: {
    total: number;
    newToday: number;
    newThisWeek: number;
    verifiedRate: number;
    bannedTotal: number;
  };
  communities: { total: number };
  issues: { total: number; byStatus: Record<string, number>; responseRate: number };
  petitions: { total: number; signaturesTotal: number; signaturesThisWeek: number };
  representatives: { total: number };
  organizations: { total: number; verified: number };
  moderation: { pendingFlags: number; hiddenAllTime: number; auditLogEntries: number };
}

function useMetrics() {
  return useQuery({
    queryKey: ['admin-metrics'],
    queryFn: () => apiGet<{ metrics: PlatformMetrics }>('/api/v1/admin/metrics'),
    refetchInterval: 30_000,
  });
}

export function OverviewPage() {
  const health = useHealth();
  const metrics = useMetrics();
  const m = metrics.data?.metrics;

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Overview</p>
        <h1 className="admin-page-title">Platform overview</h1>
        <p className="admin-page-sub">
          Live health of the four backend services + a snapshot of platform activity, moderation
          queue, and admin actions.
        </p>
      </header>

      <section className="admin-health-strip" aria-label="Service health">
        {health.isLoading ? (
          <p className="text-sm text-slate-500">Pinging services…</p>
        ) : (
          (health.data ?? []).map((svc) => (
            <span key={svc.name} className="admin-health-item">
              <span
                className={`admin-health-dot admin-health-dot-${svc.status}`}
                aria-hidden="true"
              />
              <span className="mono">{svc.name}</span>
              <span className="text-slate-500">· {svc.status}</span>
            </span>
          ))
        )}
      </section>

      <h2
        className="text-xs font-semibold text-slate-500 mono mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        PLATFORM
      </h2>
      <section className="admin-stat-grid" aria-label="Platform counters">
        <StatCard
          label="Citizens"
          value={m?.users.total ?? 0}
          sub={`${m?.users.newThisWeek ?? 0} joined this week · ${m?.users.verifiedRate ?? 0}% verified`}
        />
        <StatCard
          label="Communities"
          value={m?.communities.total ?? 0}
          sub="Every LGA/community on the platform"
        />
        <StatCard
          label="Issues"
          value={m?.issues.total ?? 0}
          sub={`${m?.issues.responseRate ?? 0}% received an official response`}
        />
        <StatCard
          label="Petitions"
          value={m?.petitions.total ?? 0}
          sub={`${m?.petitions.signaturesTotal ?? 0} signatures collected`}
        />
        <StatCard
          label="Representatives"
          value={m?.representatives.total ?? 0}
          sub="Elected officials on record"
        />
        <StatCard
          label="Organizations"
          value={m?.organizations.total ?? 0}
          sub={`${m?.organizations.verified ?? 0} verified`}
        />
      </section>

      <h2
        className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        MODERATION
      </h2>
      <section className="admin-stat-grid" aria-label="Moderation counters">
        <StatCard
          label="Pending flags"
          value={m?.moderation.pendingFlags ?? 0}
          sub="Content awaiting moderator review"
        />
        <StatCard
          label="Hidden (all time)"
          value={m?.moderation.hiddenAllTime ?? 0}
          sub="Flags where content was removed"
        />
        <StatCard
          label="Banned users"
          value={m?.users.bannedTotal ?? 0}
          sub="Accounts currently suspended"
        />
        <StatCard
          label="Audit log entries"
          value={m?.moderation.auditLogEntries ?? 0}
          sub="Every admin action recorded"
        />
      </section>

      {m?.issues.byStatus && Object.keys(m.issues.byStatus).length > 0 && (
        <>
          <h2
            className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
            style={{ letterSpacing: '0.16em' }}
          >
            ISSUES BY STATUS
          </h2>
          <section className="admin-stat-grid" aria-label="Issue status breakdown">
            {(['OPEN', 'UNDER_REVIEW', 'IN_PROGRESS', 'RESOLVED', 'CLOSED'] as const).map((s) => (
              <StatCard
                key={s}
                label={s.replace(/_/g, ' ')}
                value={m.issues.byStatus[s] ?? 0}
                sub={pctOf(m.issues.byStatus[s] ?? 0, m.issues.total)}
              />
            ))}
          </section>
        </>
      )}
    </>
  );
}

function pctOf(n: number, total: number): string {
  if (total <= 0) return '—';
  return `${Math.round((n * 100) / total)}% of ${total.toLocaleString()}`;
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
