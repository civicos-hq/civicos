import { useState, type FormEvent } from 'react';
import { useMutation } from '@tanstack/react-query';
import { EyeOff, ShieldAlert } from 'lucide-react';
import { apiPost } from '../lib/api';

const CONTENT_TYPES = [
  'ISSUE_COMMENT',
  'PETITION_COMMENT',
  'REPRESENTATIVE_COMMENT',
  'ANNOUNCEMENT',
  'PROGRESS_UPDATE',
  'ISSUE',
  'PETITION',
] as const;

const REASONS = ['SPAM', 'ABUSE', 'MISINFO', 'HATE', 'OTHER'] as const;

export function DirectHidePage() {
  const [contentType, setContentType] = useState<(typeof CONTENT_TYPES)[number]>('ISSUE_COMMENT');
  const [contentId, setContentId] = useState('');
  const [reason, setReason] = useState<(typeof REASONS)[number]>('ABUSE');
  const [note, setNote] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const submit = useMutation({
    mutationFn: () =>
      apiPost<{ flag: { id: string; status: string } }>('/api/v1/flags/direct-hide', {
        contentType,
        contentId: contentId.trim(),
        reason,
        resolutionNote: note.trim() || undefined,
      }),
    onSuccess: (data) => {
      setError(null);
      setSuccess(
        `Hidden as flag ${data.flag.id.slice(0, 8)}… — content will be masked on next fetch.`,
      );
      setContentId('');
      setNote('');
    },
    onError: (err) => {
      setSuccess(null);
      const msg = (err as { response?: { data?: { message?: string; code?: string } } }).response
        ?.data;
      setError(msg?.message ?? 'Direct hide failed.');
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    submit.mutate();
  }

  return (
    <>
      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Moderation · Direct hide</p>
        <h1 className="admin-page-title">Hide content by ID</h1>
        <p className="admin-page-sub">
          When you've spotted content that shouldn't be public and don't want to wait for a citizen
          to file a flag. Paste the content ID (from a URL, a database query, or the audit log) and
          this will create a HIDDEN flag on your behalf and record it to the audit log.
        </p>
      </header>

      <form
        onSubmit={handleSubmit}
        className="admin-table-shell"
        style={{ padding: '1.5rem', maxWidth: '640px' }}
      >
        <div className="space-y-4">
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="dh-type">
              Content type
            </label>
            <select
              id="dh-type"
              className="admin-table-search"
              value={contentType}
              onChange={(e) => setContentType(e.target.value as (typeof CONTENT_TYPES)[number])}
            >
              {CONTENT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="dh-id">
              Content ID <span className="text-xs text-slate-500 mono">(UUID)</span>
            </label>
            <input
              id="dh-id"
              className="admin-table-search mono"
              placeholder="e.g. 3fa85f64-5717-4562-b3fc-2c963f66afa6"
              value={contentId}
              onChange={(e) => setContentId(e.target.value)}
              required
              pattern="^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="dh-reason">
              Reason
            </label>
            <select
              id="dh-reason"
              className="admin-table-search"
              value={reason}
              onChange={(e) => setReason(e.target.value as (typeof REASONS)[number])}
            >
              {REASONS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="dh-note">
              Resolution note <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <textarea
              id="dh-note"
              rows={3}
              className="admin-table-search"
              placeholder="Why you're hiding this. Written to the audit trail."
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>

          {error && (
            <div
              className="flex items-start gap-2 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900"
              role="alert"
            >
              <ShieldAlert className="h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <span>{error}</span>
            </div>
          )}

          {success && (
            <div
              className="flex items-start gap-2 rounded-lg border border-emerald-300 bg-emerald-50 p-3 text-sm text-emerald-900"
              role="status"
            >
              <EyeOff className="h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <span>{success}</span>
            </div>
          )}

          <div className="flex justify-end">
            <button
              type="submit"
              className="admin-btn admin-btn-danger"
              disabled={submit.isPending || !contentId.trim()}
            >
              <EyeOff className="h-4 w-4" aria-hidden="true" />
              {submit.isPending ? 'Hiding…' : 'Hide content'}
            </button>
          </div>
        </div>
      </form>
    </>
  );
}
