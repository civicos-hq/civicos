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

interface FlagCountsResponse {
  counts: Record<string, number>;
}

function useFlagCounts() {
  return useQuery({
    queryKey: ['admin-flag-counts'],
    queryFn: () => apiGet<FlagCountsResponse>('/api/v1/flags/counts'),
    refetchInterval: 30_000,
  });
}

interface AuditListResponse {
  auditLogs: unknown[];
  total: number;
}

function useAuditCount() {
  return useQuery({
    queryKey: ['admin-audit-count'],
    queryFn: () => apiGet<AuditListResponse>('/api/v1/audit-logs?limit=1'),
  });
}

export function OverviewPage() {
  const health = useHealth();
  const flags = useFlagCounts();
  const audit = useAuditCount();

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Overview</p>
        <h1 className="admin-page-title">Platform overview</h1>
        <p className="admin-page-sub">
          Live health of the four backend services + a snapshot of the moderation queue and admin
          activity.
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

      <section className="admin-stat-grid" aria-label="Moderation counters">
        <StatCard
          label="Pending flags"
          value={flags.data?.counts.PENDING ?? 0}
          sub="Content awaiting moderator review"
        />
        <StatCard
          label="Hidden (all time)"
          value={flags.data?.counts.HIDDEN ?? 0}
          sub="Flags where content was removed"
        />
        <StatCard
          label="Dismissed"
          value={flags.data?.counts.DISMISSED ?? 0}
          sub="Flags where content was kept"
        />
        <StatCard
          label="Audit log entries"
          value={audit.data?.total ?? 0}
          sub="Every admin action recorded"
        />
      </section>
    </>
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
