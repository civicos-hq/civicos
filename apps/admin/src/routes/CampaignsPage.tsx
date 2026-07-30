import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '../lib/api';

interface CampaignRow {
  id: string;
  title: string;
  slug: string;
  status: string;
  category: string;
  currency: string;
  goalMinor: number;
  raisedMinor: number;
  organizationId: string;
  isEmergency: boolean;
  submittedAt?: string | null;
  createdByName: string;
  state?: string | null;
  lga?: string | null;
}

interface ListResponse {
  campaigns: CampaignRow[];
}

const SLA_STALE_HOURS = 48;
// Emergency relief cannot wait two days for a reviewer. The spec gives
// platform admins an "approve emergency campaigns" capability; this is the
// visual half of it.
const EMERGENCY_SLA_HOURS = 6;

function ageHours(value: string): number {
  return Math.max(1, Math.floor((Date.now() - new Date(value).getTime()) / (1000 * 60 * 60)));
}

function ageLabel(value: string): string {
  const h = ageHours(value);
  if (h < 24) return `${h}h`;
  const d = Math.floor(h / 24);
  if (d < 7) return `${d}d`;
  return `${Math.floor(d / 7)}w`;
}

function ageChipClass(row: CampaignRow): string {
  if (!row.submittedAt) return 'admin-chip admin-chip-age-stable';
  const awaiting = row.status === 'PENDING_REVIEW';
  const limit = row.isEmergency ? EMERGENCY_SLA_HOURS : SLA_STALE_HOURS;
  if (awaiting && ageHours(row.submittedAt) >= limit) {
    return 'admin-chip admin-chip-status-REJECTED';
  }
  if (awaiting) return 'admin-chip admin-chip-age-pending';
  return 'admin-chip admin-chip-age-stable';
}

/**
 * Formats integer minor units for display.
 *
 * Money is stored as int64 minor units (kobo/pence) and must not be turned
 * into a float on the way to the screen — `Intl.NumberFormat` is given the
 * major-unit value only at the final formatting step, and the division is
 * the only place it happens.
 */
function money(minor: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
  }).format(minor / 100);
}

export function CampaignsPage() {
  const [status, setStatus] = useState('PENDING_REVIEW');
  const [category, setCategory] = useState('');
  const [emergencyOnly, setEmergencyOnly] = useState(false);

  const query = useQuery({
    queryKey: ['admin-campaigns', status, category, emergencyOnly],
    queryFn: () => {
      const params = new URLSearchParams();
      if (status) params.set('status', status);
      if (category) params.set('category', category);
      if (emergencyOnly) params.set('emergency', 'true');
      return apiGet<ListResponse>(`/api/v1/admin/campaigns?${params.toString()}`);
    },
  });

  const rows = query.data?.campaigns ?? [];

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Funding</p>
        <h1 className="admin-page-title">Campaign review queue</h1>
        <p className="admin-page-sub">
          Verify community funding campaigns before they can be published. Nothing here can accept
          money yet — approval grants an organization permission to publish its goal and spend plan.
        </p>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <select
            className="admin-table-search"
            style={{ flex: '0 0 200px' }}
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">Any status</option>
            <option value="PENDING_REVIEW">Pending review</option>
            <option value="NEEDS_CHANGES">Needs changes</option>
            <option value="APPROVED">Approved</option>
            <option value="PUBLISHED">Published</option>
            <option value="PAUSED">Paused</option>
            <option value="REJECTED">Rejected</option>
            <option value="COMPLETED">Completed</option>
            <option value="REPORTED">Reported</option>
            <option value="ARCHIVED">Archived</option>
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 220px' }}
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="">Any category</option>
            <option value="EMERGENCY_RELIEF">Emergency relief</option>
            <option value="COMMUNITY_DEVELOPMENT">Community development</option>
            <option value="EDUCATION">Education</option>
            <option value="HEALTHCARE">Healthcare</option>
            <option value="ENVIRONMENT">Environment</option>
            <option value="AGRICULTURE">Agriculture</option>
            <option value="OTHER">Other</option>
          </select>
          <label className="text-xs text-slate-600 flex items-center gap-2">
            <input
              type="checkbox"
              checked={emergencyOnly}
              onChange={(e) => setEmergencyOnly(e.target.checked)}
            />
            Emergency only
          </label>
          <span className="text-xs text-slate-500 mono">{rows.length} shown</span>
        </div>

        {query.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No campaigns match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Campaign</th>
                <th>Category</th>
                <th>Goal</th>
                <th>Status</th>
                <th>Submitted</th>
                <th>Waiting</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>
                    <div className="font-semibold text-slate-900">
                      {row.isEmergency && (
                        <span
                          className="admin-chip admin-chip-status-REJECTED"
                          style={{ marginRight: 6 }}
                          title="Emergency campaign — review target is 6 hours"
                        >
                          EMERGENCY
                        </span>
                      )}
                      {row.title}
                    </div>
                    <div className="text-xs text-slate-500">
                      {[row.lga, row.state].filter(Boolean).join(', ') || 'No location set'} ·{' '}
                      {row.createdByName}
                    </div>
                  </td>
                  <td>
                    <span className="admin-chip">{row.category.replace(/_/g, ' ')}</span>
                  </td>
                  <td className="mono text-xs whitespace-nowrap">
                    {money(row.goalMinor, row.currency)}
                  </td>
                  <td>
                    <span className={`admin-chip admin-chip-status-${row.status}`}>
                      {row.status.replace(/_/g, ' ')}
                    </span>
                  </td>
                  <td className="mono text-xs text-slate-500 whitespace-nowrap">
                    {row.submittedAt ? new Date(row.submittedAt).toLocaleString() : '—'}
                  </td>
                  <td>
                    {row.submittedAt ? (
                      <span
                        className={ageChipClass(row)}
                        title={
                          row.isEmergency
                            ? `Emergency review target: ${EMERGENCY_SLA_HOURS}h`
                            : `Review target: ${SLA_STALE_HOURS}h`
                        }
                      >
                        {ageLabel(row.submittedAt)}
                      </span>
                    ) : (
                      <span className="text-xs text-slate-400">—</span>
                    )}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <Link
                      to={`/campaigns/${row.id}`}
                      className="admin-btn admin-btn-secondary admin-btn-sm"
                    >
                      Review
                    </Link>
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
