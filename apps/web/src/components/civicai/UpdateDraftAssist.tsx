import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from '@tanstack/react-query';
import { AIWarnings, CampaignAIPanel } from './CampaignAIPanel';
import {
  draftCompletionReport,
  draftDonorUpdate,
  type CompletionReportDraft,
  type DonorUpdateDraft,
} from '../../lib/civicai';
import { formatMoneyExact } from '../../hooks/useCampaigns';

/**
 * "Draft with AI" for a funding update — the post donors actually read.
 *
 * The brief is required by the API, deliberately: an update generated from
 * the ledger alone would be the platform writing in the organization's voice
 * about work only the organization witnessed.
 */
export function DonorUpdateAssist({
  campaignId,
  onApply,
}: {
  campaignId: string;
  onApply: (d: { title: string; body: string }) => void;
}) {
  const { t } = useTranslation();
  const [brief, setBrief] = useState('');
  const [draft, setDraft] = useState<DonorUpdateDraft | null>(null);
  const [error, setError] = useState('');

  const gen = useMutation({
    mutationFn: () => draftDonorUpdate({ campaignId, brief: brief.trim() }),
    onSuccess: (d) => {
      setDraft(d);
      setError('');
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? t('campaignAi.genericError'));
    },
  });

  return (
    <CampaignAIPanel
      title={t('campaignAi.donorUpdate.title')}
      hint={t('campaignAi.donorUpdate.hint')}
      brief={brief}
      onBriefChange={setBrief}
      placeholder={t('campaignAi.donorUpdate.placeholder')}
      minBrief={10}
      onGenerate={() => gen.mutate()}
      isPending={gen.isPending}
      error={error}
      provenance={draft ?? undefined}
      onApply={draft ? () => onApply({ title: draft.title, body: draft.body }) : undefined}
    >
      {draft && (
        <>
          <AIWarnings warnings={draft.warnings} />
          <p className="font-semibold text-slate-900 dark:text-slate-100">{draft.title}</p>
          <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-200">
            {draft.body}
          </p>
        </>
      )}
    </CampaignAIPanel>
  );
}

/**
 * "Draft with AI" for the closing report.
 *
 * The shortfall shown here comes from the server, computed from the ledger —
 * never from the model. It is the figure that gets frozen onto the public
 * page the moment the report is filed, so it is rendered separately from the
 * generated prose and labelled as a fact rather than a draft.
 */
export function CompletionReportAssist({
  campaignId,
  onApply,
}: {
  campaignId: string;
  onApply: (body: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const [brief, setBrief] = useState('');
  const [draft, setDraft] = useState<CompletionReportDraft | null>(null);
  const [error, setError] = useState('');

  const gen = useMutation({
    mutationFn: () => draftCompletionReport({ campaignId, brief: brief.trim() }),
    onSuccess: (d) => {
      setDraft(d);
      setError('');
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      setError(msg ?? t('campaignAi.genericError'));
    },
  });

  return (
    <CampaignAIPanel
      title={t('campaignAi.completionReport.title')}
      hint={t('campaignAi.completionReport.hint')}
      brief={brief}
      onBriefChange={setBrief}
      placeholder={t('campaignAi.completionReport.placeholder')}
      minBrief={20}
      onGenerate={() => gen.mutate()}
      isPending={gen.isPending}
      error={error}
      provenance={draft ?? undefined}
      onApply={draft ? () => onApply(draft.body) : undefined}
    >
      {draft && (
        <>
          {/* Ledger arithmetic, not model output. Shown above the prose so
              it is read first. */}
          {draft.mustExplain && (
            <p className="rounded-lg border border-amber-300 dark:border-amber-500/50 bg-amber-50 dark:bg-amber-500/10 p-2.5 text-sm font-medium text-amber-900 dark:text-amber-100">
              {t('campaignAi.completionReport.shortfall', {
                amount: formatMoneyExact(draft.unaccountedMinor, draft.currency, i18n.language),
              })}
            </p>
          )}
          <AIWarnings warnings={draft.warnings} />
          <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-200">
            {draft.body}
          </p>
        </>
      )}
    </CampaignAIPanel>
  );
}
