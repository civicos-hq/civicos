import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, ArrowLeft, CheckCircle2, XCircle } from 'lucide-react';
import { apiGet, apiPatch } from '../lib/api';

interface Applicant {
  id: string;
  email: string;
  name: string;
  role: string;
  requestedAccountType: string;
  approvalStatus: string;
  emailVerified: boolean;
  approvalNote?: string | null;
}

interface RepresentativeApplication {
  id: string;
  fullName: string;
  title: string;
  position: string;
  constituency: string;
  communityId: string;
  party?: string | null;
  bio?: string | null;
  officialEmail?: string | null;
  officialPhone?: string | null;
  website?: string | null;
  proofUrls?: string[];
  reviewNote?: string | null;
}

interface OrganizationApplication {
  id: string;
  name: string;
  slug: string;
  kind: string;
  jurisdiction: string;
  state?: string | null;
  lga?: string | null;
  description?: string | null;
  officialEmail?: string | null;
  officialPhone?: string | null;
  website?: string | null;
  proofUrls?: string[];
  reviewNote?: string | null;
}

interface ApplicationDetail {
  kind: string;
  id: string;
  status: string;
  submittedAt: string;
  reviewedAt?: string | null;
  applicant: Applicant;
  representativeApplication?: RepresentativeApplication;
  organizationApplication?: OrganizationApplication;
  reviewHistory: ReviewHistoryItem[];
}

interface DetailResponse {
  application: ApplicationDetail;
}

interface ReviewHistoryItem {
  id: string;
  reviewerName: string;
  status: string;
  note?: string | null;
  createdAt: string;
}

function timeSince(value: string): string {
  const diffMs = Date.now() - new Date(value).getTime();
  const diffHours = Math.max(1, Math.floor(diffMs / (1000 * 60 * 60)));
  if (diffHours < 24) return `${diffHours} hours`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays} days`;
  const diffWeeks = Math.floor(diffDays / 7);
  return `${diffWeeks} weeks`;
}

export function ApplicationDetailPage() {
  const { kind = '', id = '' } = useParams<{ kind: string; id: string }>();
  const queryClient = useQueryClient();
  const [note, setNote] = useState('');
  const [error, setError] = useState('');

  const query = useQuery({
    queryKey: ['admin-application', kind, id],
    queryFn: () => apiGet<DetailResponse>(`/api/v1/admin/applications/${kind}/${id}`),
    enabled: Boolean(kind && id),
  });

  const review = useMutation({
    mutationFn: ({ status }: { status: 'APPROVED' | 'NEEDS_CHANGES' | 'REJECTED' }) =>
      apiPatch<DetailResponse>(`/api/v1/admin/applications/${kind}/${id}`, {
        status,
        note: note.trim() || undefined,
      }),
    onSuccess: (data) => {
      setError('');
      queryClient.setQueryData(['admin-application', kind, id], data);
      queryClient.invalidateQueries({ queryKey: ['admin-applications'] });
      queryClient.invalidateQueries({ queryKey: ['admin-audit'] });
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? 'Could not review this application.');
    },
  });

  const item = query.data?.application;

  useEffect(() => {
    if (!item) return;
    const currentNote =
      item.kind === 'REPRESENTATIVE'
        ? item.representativeApplication?.reviewNote
        : item.organizationApplication?.reviewNote;
    setNote(currentNote ?? '');
  }, [item]);

  return (
    <>
      <Link
        to="/applications"
        className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to Applications
      </Link>

      {query.isLoading ? (
        <div className="admin-empty">Loading…</div>
      ) : !item ? (
        <div className="admin-empty">Application not found.</div>
      ) : (
        <div className="space-y-4">
          <header className="admin-page-header">
            <p className="admin-page-eyebrow">Section — Approval · Detail</p>
            <h1 className="admin-page-title">
              {item.kind === 'REPRESENTATIVE'
                ? item.representativeApplication?.fullName
                : item.organizationApplication?.name}
            </h1>
            <p className="admin-page-sub">
              Applicant: {item.applicant.name} · {item.applicant.email}
            </p>
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <span className={`admin-chip admin-chip-status-${item.status}`}>{item.status}</span>
              <span className="admin-chip admin-chip-age-pending">
                Submitted {timeSince(item.submittedAt)} ago
              </span>
            </div>
          </header>

          <section className="admin-table-shell" style={{ padding: '1.5rem' }}>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-3">
                <DetailField label="Application type" value={item.kind} />
                <DetailField label="Status" value={item.status} />
                <DetailField
                  label="Submitted"
                  value={new Date(item.submittedAt).toLocaleString()}
                />
                <DetailField
                  label="Reviewed"
                  value={item.reviewedAt ? new Date(item.reviewedAt).toLocaleString() : '—'}
                />
                <DetailField label="Applicant role today" value={item.applicant.role} />
                <DetailField
                  label="Email verified"
                  value={item.applicant.emailVerified ? 'Yes' : 'No'}
                />
              </div>

              <div className="space-y-3">
                {item.kind === 'REPRESENTATIVE' && item.representativeApplication ? (
                  <>
                    <DetailField label="Title" value={item.representativeApplication.title} />
                    <DetailField label="Position" value={item.representativeApplication.position} />
                    <DetailField
                      label="Constituency"
                      value={item.representativeApplication.constituency}
                    />
                    <DetailField
                      label="Community ID"
                      value={item.representativeApplication.communityId}
                    />
                    <DetailField
                      label="Party"
                      value={item.representativeApplication.party ?? '—'}
                    />
                    <DetailField
                      label="Official email"
                      value={item.representativeApplication.officialEmail ?? '—'}
                    />
                    <DetailField
                      label="Official phone"
                      value={item.representativeApplication.officialPhone ?? '—'}
                    />
                    <DetailField
                      label="Website"
                      value={item.representativeApplication.website ?? '—'}
                    />
                    <DetailField
                      label="Bio"
                      value={item.representativeApplication.bio ?? '—'}
                      multiline
                    />
                    <DetailField
                      label="Proof URLs"
                      value={(item.representativeApplication.proofUrls ?? []).join(', ') || '—'}
                      multiline
                    />
                  </>
                ) : item.organizationApplication ? (
                  <>
                    <DetailField label="Slug" value={item.organizationApplication.slug} />
                    <DetailField label="Kind" value={item.organizationApplication.kind} />
                    <DetailField
                      label="Jurisdiction"
                      value={item.organizationApplication.jurisdiction}
                    />
                    <DetailField label="State" value={item.organizationApplication.state ?? '—'} />
                    <DetailField label="LGA" value={item.organizationApplication.lga ?? '—'} />
                    <DetailField
                      label="Official email"
                      value={item.organizationApplication.officialEmail ?? '—'}
                    />
                    <DetailField
                      label="Official phone"
                      value={item.organizationApplication.officialPhone ?? '—'}
                    />
                    <DetailField
                      label="Website"
                      value={item.organizationApplication.website ?? '—'}
                    />
                    <DetailField
                      label="Description"
                      value={item.organizationApplication.description ?? '—'}
                      multiline
                    />
                    <DetailField
                      label="Proof URLs"
                      value={(item.organizationApplication.proofUrls ?? []).join(', ') || '—'}
                      multiline
                    />
                  </>
                ) : null}
              </div>
            </div>
          </section>

          <section className="admin-table-shell" style={{ padding: '1.5rem', maxWidth: '760px' }}>
            <h2 className="text-lg font-semibold text-slate-900">Review note</h2>
            <p className="mt-1 text-sm text-slate-600">
              Notes are required for needs-changes and rejection decisions and are shown back to the
              applicant.
            </p>
            <textarea
              className="admin-table-search mt-4"
              rows={4}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Why was this approved or rejected?"
            />
            {error && (
              <div className="admin-login-error" style={{ marginTop: '1rem' }}>
                {error}
              </div>
            )}
            <div className="mt-4 flex items-center gap-2">
              <button
                type="button"
                className="admin-btn admin-btn-primary"
                onClick={() => review.mutate({ status: 'APPROVED' })}
                disabled={review.isPending}
              >
                <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
                Approve
              </button>
              <button
                type="button"
                className="admin-btn admin-btn-secondary"
                onClick={() => review.mutate({ status: 'NEEDS_CHANGES' })}
                disabled={review.isPending}
              >
                <AlertCircle className="h-4 w-4" aria-hidden="true" />
                Needs changes
              </button>
              <button
                type="button"
                className="admin-btn admin-btn-danger"
                onClick={() => review.mutate({ status: 'REJECTED' })}
                disabled={review.isPending}
              >
                <XCircle className="h-4 w-4" aria-hidden="true" />
                Reject
              </button>
            </div>
          </section>

          <section className="admin-table-shell" style={{ padding: '1.5rem', maxWidth: '760px' }}>
            <h2 className="text-lg font-semibold text-slate-900">Review history</h2>
            <p className="mt-1 text-sm text-slate-600">
              Every approval decision and review note stays on the record.
            </p>
            <div className="mt-4 space-y-3">
              {item.reviewHistory.length === 0 ? (
                <div className="admin-empty" style={{ margin: 0 }}>
                  No review decisions yet.
                </div>
              ) : (
                item.reviewHistory.map((entry) => (
                  <div key={entry.id} className="admin-review-event">
                    <div className="admin-review-event-row">
                      <div>
                        <div className="font-semibold text-slate-900">{entry.reviewerName}</div>
                        <div className="text-xs text-slate-500">
                          {new Date(entry.createdAt).toLocaleString()}
                        </div>
                      </div>
                      <span className={`admin-chip admin-chip-status-${entry.status}`}>
                        {entry.status}
                      </span>
                    </div>
                    <div className="mt-2 text-sm text-slate-700 whitespace-pre-wrap">
                      {entry.note?.trim() || 'No note left.'}
                    </div>
                  </div>
                ))
              )}
            </div>
          </section>
        </div>
      )}
    </>
  );
}

function DetailField({
  label,
  value,
  multiline = false,
}: {
  label: string;
  value: string;
  multiline?: boolean;
}) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
        {label}
      </div>
      <div
        className={`mt-1 text-sm text-slate-900 ${multiline ? 'whitespace-pre-wrap break-words' : ''}`}
      >
        {value}
      </div>
    </div>
  );
}
