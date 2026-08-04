import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '../lib/api';
import { RiskPanel } from './RiskPanel';

interface Campaign {
  id: string;
  title: string;
  slug: string;
  summary: string;
  description: string;
  status: string;
  category: string;
  currency: string;
  goalMinor: number;
  raisedMinor: number;
  donorCount: number;
  organizationId: string;
  isEmergency: boolean;
  approvalStatus: string;
  reviewNote?: string | null;
  pauseReasonCode?: string | null;
  pauseNote?: string | null;
  submittedAt?: string | null;
  publishedAt?: string | null;
  createdByName: string;
  state?: string | null;
  lga?: string | null;
}

interface Milestone {
  id: string;
  title: string;
  description?: string | null;
  targetMinor: number;
  status: string;
  position: number;
}

interface Organization {
  id: string;
  name: string;
  verified: boolean;
  registrationNumber?: string | null;
  country?: string | null;
  officialEmail?: string | null;
  representativeName?: string | null;
  bankAccountVerified: boolean;
  supportingDocumentUrl?: string | null;
}

interface Eligibility {
  eligible: boolean;
  missing: string[];
}

const PAUSE_REASONS = [
  ['FRAUD_DETECTED', 'Fraud detected'],
  ['VERIFICATION_EXPIRED', 'Verification expired'],
  ['MISUSE_REPORTED', 'Misuse reported'],
  ['ORGANIZATION_SUSPENDED', 'Organization suspended'],
  ['FALSE_INFORMATION', 'False information identified'],
  ['OTHER', 'Other'],
] as const;

function money(minor: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    maximumFractionDigits: 2,
  }).format(minor / 100);
}

export function CampaignDetailPage() {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const [note, setNote] = useState('');
  const [pauseReason, setPauseReason] = useState<string>('MISUSE_REPORTED');
  const [error, setError] = useState<string | null>(null);

  const campaignQ = useQuery({
    queryKey: ['admin-campaign', id],
    queryFn: () => apiGet<{ campaign: Campaign }>(`/api/v1/campaigns/${id}`),
    enabled: !!id,
  });
  const campaign = campaignQ.data?.campaign;

  const milestonesQ = useQuery({
    queryKey: ['admin-campaign-milestones', id],
    queryFn: () => apiGet<{ milestones: Milestone[] }>(`/api/v1/campaigns/${id}/milestones`),
    enabled: !!id,
  });

  const orgQ = useQuery({
    queryKey: ['admin-campaign-org', campaign?.organizationId],
    queryFn: () =>
      apiGet<{ organization: Organization }>(`/api/v1/organizations/${campaign!.organizationId}`),
    enabled: !!campaign?.organizationId,
  });

  // The eligibility checklist is computed server-side so the console and the
  // submit gate can never disagree about what "eligible" means.
  const eligibilityQ = useQuery({
    queryKey: ['admin-campaign-eligibility', campaign?.organizationId],
    queryFn: () =>
      apiGet<Eligibility>(`/api/v1/organizations/${campaign!.organizationId}/funding-eligibility`),
    enabled: !!campaign?.organizationId,
  });

  function refresh() {
    queryClient.invalidateQueries({ queryKey: ['admin-campaign', id] });
    queryClient.invalidateQueries({ queryKey: ['admin-campaigns'] });
    queryClient.invalidateQueries({ queryKey: ['admin-audit'] });
  }

  const act = useMutation({
    mutationFn: (input: { path: string; body?: unknown }) =>
      apiPost(`/api/v1/campaigns/${id}/${input.path}`, input.body ?? {}),
    onSuccess: () => {
      setError(null);
      setNote('');
      refresh();
    },
    onError: (e: unknown) => {
      const res = (e as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? 'Action failed.');
    },
  });

  if (campaignQ.isLoading) return <div className="admin-empty">Loading…</div>;
  if (!campaign) return <div className="admin-empty">Campaign not found.</div>;

  const milestones = milestonesQ.data?.milestones ?? [];
  const org = orgQ.data?.organization;
  const eligibility = eligibilityQ.data;
  const plannedTotal = milestones.reduce((sum, m) => sum + m.targetMinor, 0);
  const underReview = campaign.status === 'PENDING_REVIEW';
  const canPause = campaign.status === 'PUBLISHED' || campaign.status === 'FUNDED';
  const noteRequired = !note.trim();

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">
          <Link to="/campaigns">← Campaign review queue</Link>
        </p>
        <h1 className="admin-page-title">{campaign.title}</h1>
        <p className="admin-page-sub">
          {campaign.summary} · {[campaign.lga, campaign.state].filter(Boolean).join(', ')}
        </p>
        <div className="mt-2 flex gap-2">
          <span className={`admin-chip admin-chip-status-${campaign.status}`}>
            {campaign.status.replace(/_/g, ' ')}
          </span>
          <span className="admin-chip">{campaign.category.replace(/_/g, ' ')}</span>
          {campaign.isEmergency && (
            <span className="admin-chip admin-chip-status-REJECTED">EMERGENCY</span>
          )}
        </div>
      </header>

      {error && (
        <div className="admin-empty" style={{ color: '#b91c1c', marginBottom: 16 }}>
          {error}
        </div>
      )}

      <section className="admin-table-shell" style={{ marginBottom: 16 }}>
        <div className="admin-table-toolbar">
          <strong className="text-sm">Ask</strong>
        </div>
        <div className="p-4 text-sm">
          <div className="mono">
            Goal {money(campaign.goalMinor, campaign.currency)} · raised{' '}
            {money(campaign.raisedMinor, campaign.currency)} from {campaign.donorCount} donors
          </div>
          <p className="mt-3 whitespace-pre-wrap text-slate-700">{campaign.description}</p>
        </div>
      </section>

      {/* After the Ask, never before it: a reviewer should read what the
          organization actually wrote before reading a machine's opinion of
          it. */}
      <RiskPanel campaignId={campaign.id} />

      {/* Verification evidence. A reviewer approving a fundraiser needs the
          organization's paperwork in front of them, not one tab away. */}
      <section className="admin-table-shell" style={{ marginBottom: 16 }}>
        <div className="admin-table-toolbar">
          <strong className="text-sm">Organization verification</strong>
          {eligibility && (
            <span
              className={`admin-chip ${
                eligibility.eligible
                  ? 'admin-chip-status-APPROVED'
                  : 'admin-chip-status-NEEDS_CHANGES'
              }`}
            >
              {eligibility.eligible ? 'Funding eligible' : 'Not eligible'}
            </span>
          )}
        </div>
        <div className="p-4 text-sm">
          {!org ? (
            <div className="text-slate-500">Loading organization…</div>
          ) : (
            <>
              <div className="font-semibold">{org.name}</div>
              <dl className="mt-3 grid grid-cols-2 gap-y-2 text-xs">
                <dt className="text-slate-500">Verified badge</dt>
                <dd>{org.verified ? 'Yes' : 'No'}</dd>
                <dt className="text-slate-500">Registration number</dt>
                <dd className="mono">{org.registrationNumber || '—'}</dd>
                <dt className="text-slate-500">Country</dt>
                <dd>{org.country || '—'}</dd>
                <dt className="text-slate-500">Official email</dt>
                <dd className="mono">{org.officialEmail || '—'}</dd>
                <dt className="text-slate-500">Representative</dt>
                <dd>{org.representativeName || '—'}</dd>
                <dt className="text-slate-500">Bank account verified</dt>
                <dd>{org.bankAccountVerified ? 'Yes' : 'No'}</dd>
                <dt className="text-slate-500">Supporting document</dt>
                <dd>
                  {org.supportingDocumentUrl ? (
                    <a href={org.supportingDocumentUrl} target="_blank" rel="noopener noreferrer">
                      View
                    </a>
                  ) : (
                    '—'
                  )}
                </dd>
              </dl>
              {eligibility && !eligibility.eligible && (
                <p className="mt-3 text-xs" style={{ color: '#b91c1c' }}>
                  Still outstanding: {eligibility.missing.join(', ')}. The organization cannot
                  submit a campaign until these are supplied.
                </p>
              )}
            </>
          )}
        </div>
      </section>

      <section className="admin-table-shell" style={{ marginBottom: 16 }}>
        <div className="admin-table-toolbar">
          <strong className="text-sm">Spend plan</strong>
          <span className="text-xs text-slate-500 mono">
            {money(plannedTotal, campaign.currency)} of{' '}
            {money(campaign.goalMinor, campaign.currency)} allocated
          </span>
        </div>
        {milestones.length === 0 ? (
          <div className="admin-empty">No milestones.</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Milestone</th>
                <th>Target</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {milestones.map((m) => (
                <tr key={m.id}>
                  <td className="mono text-xs">{m.position}</td>
                  <td>
                    <div>{m.title}</div>
                    {m.description && <div className="text-xs text-slate-500">{m.description}</div>}
                  </td>
                  <td className="mono text-xs">{money(m.targetMinor, campaign.currency)}</td>
                  <td>
                    <span className="admin-chip">{m.status.replace(/_/g, ' ')}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {campaign.reviewNote && (
        <section className="admin-table-shell" style={{ marginBottom: 16 }}>
          <div className="admin-table-toolbar">
            <strong className="text-sm">Last review note</strong>
          </div>
          <div className="p-4 text-sm text-slate-700">{campaign.reviewNote}</div>
        </section>
      )}

      {campaign.status === 'PAUSED' && (
        <section className="admin-table-shell" style={{ marginBottom: 16 }}>
          <div className="admin-table-toolbar">
            <strong className="text-sm">Paused</strong>
            <span className="admin-chip admin-chip-status-REJECTED">
              {campaign.pauseReasonCode?.replace(/_/g, ' ')}
            </span>
          </div>
          <div className="p-4 text-sm text-slate-700">{campaign.pauseNote}</div>
        </section>
      )}

      <section className="admin-table-shell">
        <div className="admin-table-toolbar">
          <strong className="text-sm">Decision</strong>
        </div>
        <div className="p-4">
          <p className="text-xs text-slate-500">
            A note is required for needs-changes, rejection and pause, and is shown back to the
            organization. Approval does not publish the campaign — the organization chooses when to
            publish.
          </p>
          <textarea
            className="admin-table-search mt-4"
            rows={4}
            style={{ width: '100%' }}
            placeholder="What needs to change, or why is this being rejected or paused?"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />

          <div className="mt-4 flex flex-wrap gap-2">
            {underReview ? (
              <>
                <button
                  className="admin-btn admin-btn-primary"
                  disabled={act.isPending}
                  onClick={() => act.mutate({ path: 'review', body: { decision: 'APPROVED' } })}
                >
                  Approve
                </button>
                <button
                  className="admin-btn admin-btn-secondary"
                  disabled={act.isPending || noteRequired}
                  title={noteRequired ? 'A note is required' : undefined}
                  onClick={() =>
                    act.mutate({ path: 'review', body: { decision: 'NEEDS_CHANGES', note } })
                  }
                >
                  Request changes
                </button>
                <button
                  className="admin-btn admin-btn-danger"
                  disabled={act.isPending || noteRequired}
                  title={noteRequired ? 'A note is required' : undefined}
                  onClick={() =>
                    act.mutate({ path: 'review', body: { decision: 'REJECTED', note } })
                  }
                >
                  Reject
                </button>
              </>
            ) : (
              <span className="text-xs text-slate-500">
                This campaign is {campaign.status.replace(/_/g, ' ').toLowerCase()} — no review
                decision is pending.
              </span>
            )}
          </div>

          {canPause && (
            <div className="mt-6 border-t pt-4">
              <strong className="text-sm">Governance</strong>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <select
                  className="admin-table-search"
                  style={{ flex: '0 0 240px' }}
                  value={pauseReason}
                  onChange={(e) => setPauseReason(e.target.value)}
                >
                  {PAUSE_REASONS.map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
                <button
                  className="admin-btn admin-btn-danger"
                  disabled={act.isPending || noteRequired}
                  title={noteRequired ? 'A note is required' : undefined}
                  onClick={() =>
                    act.mutate({ path: 'pause', body: { reasonCode: pauseReason, note } })
                  }
                >
                  Pause campaign
                </button>
              </div>
            </div>
          )}

          {campaign.status === 'PAUSED' && (
            <div className="mt-6 border-t pt-4">
              <button
                className="admin-btn admin-btn-secondary"
                disabled={act.isPending}
                onClick={() => act.mutate({ path: 'resume' })}
              >
                Resume campaign
              </button>
            </div>
          )}

          {campaign.status === 'REPORTED' && (
            <div className="mt-6 border-t pt-4">
              <button
                className="admin-btn admin-btn-secondary"
                disabled={act.isPending}
                onClick={() => act.mutate({ path: 'archive' })}
              >
                Archive campaign
              </button>
            </div>
          )}
        </div>
      </section>
    </>
  );
}
