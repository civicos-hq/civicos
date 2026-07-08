import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '../lib/api';

interface Applicant {
  id: string;
  email: string;
  name: string;
  role: string;
  requestedAccountType: string;
  approvalStatus: string;
  emailVerified: boolean;
}

interface ApplicationSummary {
  kind: string;
  id: string;
  status: string;
  submittedAt: string;
  reviewedAt?: string | null;
  headline: string;
  subhead: string;
  applicant: Applicant;
}

interface ListResponse {
  applications: ApplicationSummary[];
  total: number;
}

const SLA_STALE_HOURS = 48;

function ageHours(value: string): number {
  return Math.max(1, Math.floor((Date.now() - new Date(value).getTime()) / (1000 * 60 * 60)));
}

function ageLabel(value: string): string {
  const diffHours = ageHours(value);
  if (diffHours < 24) return `${diffHours}h`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d`;
  const diffWeeks = Math.floor(diffDays / 7);
  return `${diffWeeks}w`;
}

function ageChipClass(status: string, submittedAt: string): string {
  const awaitingReview = status === 'PENDING' || status === 'NEEDS_CHANGES';
  if (awaitingReview && ageHours(submittedAt) >= SLA_STALE_HOURS) {
    return 'admin-chip admin-chip-status-REJECTED';
  }
  if (awaitingReview) return 'admin-chip admin-chip-age-pending';
  return 'admin-chip admin-chip-age-stable';
}

export function ApplicationsPage() {
  const [kind, setKind] = useState('');
  const [status, setStatus] = useState('PENDING');
  const [q, setQ] = useState('');
  const [staleAfterHours, setStaleAfterHours] = useState('');

  const query = useQuery({
    queryKey: ['admin-applications', kind, status, q, staleAfterHours],
    queryFn: () => {
      const params = new URLSearchParams();
      if (kind) params.set('kind', kind);
      if (status) params.set('status', status);
      if (q) params.set('q', q);
      if (staleAfterHours) params.set('staleAfterHours', staleAfterHours);
      params.set('limit', '50');
      return apiGet<ListResponse>(`/api/v1/admin/applications?${params.toString()}`);
    },
  });

  const rows = query.data?.applications ?? [];
  const total = query.data?.total ?? 0;

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Approval</p>
        <h1 className="admin-page-title">Application review queue</h1>
        <p className="admin-page-sub">
          Review representative and organization signup requests before elevated access is granted.
        </p>
      </header>

      <div className="admin-table-shell">
        <div className="admin-table-toolbar">
          <input
            className="admin-table-search"
            placeholder="Search applicant, email, title, organization…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={kind}
            onChange={(e) => setKind(e.target.value)}
          >
            <option value="">Any type</option>
            <option value="REPRESENTATIVE">Representative</option>
            <option value="ORGANIZATION">Organization</option>
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 180px' }}
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">Any status</option>
            <option value="PENDING">Pending</option>
            <option value="NEEDS_CHANGES">Needs changes</option>
            <option value="APPROVED">Approved</option>
            <option value="REJECTED">Rejected</option>
          </select>
          <select
            className="admin-table-search"
            style={{ flex: '0 0 160px' }}
            value={staleAfterHours}
            onChange={(e) => setStaleAfterHours(e.target.value)}
            title="Filter by how long the application has been waiting"
          >
            <option value="">Any age</option>
            <option value="24">Waiting &gt; 24h</option>
            <option value="48">Waiting &gt; 48h</option>
            <option value="168">Waiting &gt; 7d</option>
          </select>
          <span className="text-xs text-slate-500 mono">
            {rows.length} of {total.toLocaleString()}
          </span>
        </div>

        {query.isLoading ? (
          <div className="admin-empty">Loading…</div>
        ) : rows.length === 0 ? (
          <div className="admin-empty">No applications match this filter.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Request</th>
                <th>Applicant</th>
                <th>Status</th>
                <th>Submitted</th>
                <th>Age</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>
                    <span className={`admin-chip admin-chip-role-${row.kind}`}>{row.kind}</span>
                  </td>
                  <td>
                    <div className="font-semibold text-slate-900">{row.headline}</div>
                    <div className="text-xs text-slate-500">{row.subhead}</div>
                  </td>
                  <td>
                    <div>{row.applicant.name}</div>
                    <div className="mono text-xs text-slate-500">{row.applicant.email}</div>
                  </td>
                  <td>
                    <span className={`admin-chip admin-chip-status-${row.status}`}>
                      {row.status}
                    </span>
                  </td>
                  <td className="mono text-xs text-slate-500 whitespace-nowrap">
                    {new Date(row.submittedAt).toLocaleString()}
                  </td>
                  <td>
                    <span
                      className={ageChipClass(row.status, row.submittedAt)}
                      title={
                        row.status === 'PENDING' && ageHours(row.submittedAt) >= SLA_STALE_HOURS
                          ? `Awaiting review for over ${SLA_STALE_HOURS}h`
                          : undefined
                      }
                    >
                      {ageLabel(row.submittedAt)}
                    </span>
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <Link
                      to={`/applications/${row.kind}/${row.id}`}
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
