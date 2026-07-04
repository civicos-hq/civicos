import { useState, type FormEvent } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Flag, ShieldAlert } from 'lucide-react';
import { Button } from '@civicos/ui';
import { api } from '../../lib/api';
import { Modal } from '../Modal';

// The seven content types the moderation queue accepts. Kept as a
// string union so callers get compile-time safety when wiring the
// button into a particular surface.
export type ReportableType =
  | 'ISSUE'
  | 'ISSUE_COMMENT'
  | 'PETITION'
  | 'PETITION_COMMENT'
  | 'REPRESENTATIVE_COMMENT'
  | 'ANNOUNCEMENT'
  | 'PROGRESS_UPDATE';

interface Props {
  contentType: ReportableType;
  contentId: string;
  onClose: () => void;
}

const REASONS = ['SPAM', 'ABUSE', 'MISINFO', 'HATE', 'OTHER'] as const;
type Reason = (typeof REASONS)[number];

const MAX_DESCRIPTION = 500;

export function ReportModal({ contentType, contentId, onClose }: Props) {
  const { t } = useTranslation();
  const [reason, setReason] = useState<Reason>('SPAM');
  const [description, setDescription] = useState('');
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [retryAfter, setRetryAfter] = useState<number | null>(null);

  const submit = useMutation({
    mutationFn: async () => {
      await api.post('/api/v1/flags', {
        contentType,
        contentId,
        reason,
        description: description.trim() || undefined,
      });
    },
    onError: (err) => {
      const res = (
        err as {
          response?: { status: number; data?: { code?: string; data?: { retryAfter?: number } } };
        }
      ).response;
      const code = res?.data?.code ?? null;
      setErrorCode(code ?? 'GENERIC');
      if (res?.status === 429 && res.data?.data?.retryAfter) {
        setRetryAfter(res.data.data.retryAfter);
      }
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setErrorCode(null);
    setRetryAfter(null);
    submit.mutate();
  }

  return (
    <Modal title={t('report.modal.title')} onClose={onClose}>
      {submit.isSuccess ? (
        <div className="flex flex-col items-center gap-3 py-4 text-center">
          <ShieldAlert className="h-8 w-8 text-civic-700" aria-hidden="true" />
          <p className="text-sm font-semibold text-slate-900">{t('report.success.title')}</p>
          <p className="text-sm text-slate-600">{t('report.success.body')}</p>
          <Button onClick={onClose} className="mt-2">
            {t('report.success.close')}
          </Button>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          <p className="text-sm text-slate-600">{t('report.intro')}</p>

          <fieldset className="space-y-2">
            <legend className="text-sm font-medium text-slate-700">
              {t('report.reasonLegend')}
            </legend>
            {REASONS.map((r) => (
              <label
                key={r}
                className="flex items-start gap-2 rounded-lg border border-slate-200 p-2 hover:border-civic-300 cursor-pointer"
              >
                <input
                  type="radio"
                  name="report-reason"
                  value={r}
                  checked={reason === r}
                  onChange={() => setReason(r)}
                  className="mt-1"
                />
                <span>
                  <span className="block text-sm font-semibold text-slate-900">
                    {t(`report.reasons.${r}.label`)}
                  </span>
                  <span className="block text-xs text-slate-600">
                    {t(`report.reasons.${r}.hint`)}
                  </span>
                </span>
              </label>
            ))}
          </fieldset>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="report-description">
              {t('report.descriptionLabel')}
            </label>
            <textarea
              id="report-description"
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value.slice(0, MAX_DESCRIPTION))}
              placeholder={t('report.descriptionPlaceholder')}
              className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            />
            <p className="text-xs text-slate-500">
              {description.length}/{MAX_DESCRIPTION}
            </p>
          </div>

          {errorCode && <ErrorPanel code={errorCode} retryAfter={retryAfter} />}

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" loading={submit.isPending} disabled={submit.isPending}>
              <Flag className="h-4 w-4" aria-hidden="true" />
              {t('report.submit')}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}

function ErrorPanel({ code, retryAfter }: { code: string; retryAfter: number | null }) {
  const { t } = useTranslation();
  switch (code) {
    case 'ALREADY_FLAGGED':
      return (
        <p className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          {t('report.errors.alreadyFlagged')}
        </p>
      );
    case 'EMAIL_NOT_VERIFIED':
      return (
        <p className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          {t('report.errors.emailNotVerified')}
        </p>
      );
    case 'RATE_LIMITED':
      return (
        <p className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          {t('report.errors.rateLimited', { seconds: retryAfter ?? 60 })}
        </p>
      );
    default:
      return (
        <p className="rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900">
          {t('report.errors.generic')}
        </p>
      );
  }
}
