import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation } from '@tanstack/react-query';
import { AIWarnings, CampaignAIPanel } from './CampaignAIPanel';
import { draftCampaign, type CampaignDraft } from '../../lib/civicai';
import { formatMoney } from '../../hooks/useCampaigns';

/**
 * "Draft with AI" on the new-campaign form.
 *
 * The brief goes in, a title/summary/description and a suggested milestone
 * split come back. Nothing is written into the form until the organization
 * clicks Use — a draft they have not read is worse than a blank form,
 * because a blank form does not look finished.
 *
 * Milestone targets are shown but NOT applied: the server rejects a plan
 * that exceeds the goal, and how the money is split is the organization's
 * decision about its own campaign. They are here to be copied deliberately.
 */
export function CampaignDraftAssist({
  goalMinor,
  currency,
  state,
  lga,
  isEmergency,
  organizationName,
  onApply,
}: {
  goalMinor: number;
  currency: string;
  state: string;
  lga: string;
  isEmergency: boolean;
  organizationName?: string;
  onApply: (d: { title: string; summary: string; description: string }) => void;
}) {
  const { t, i18n } = useTranslation();
  const [brief, setBrief] = useState('');
  const [draft, setDraft] = useState<CampaignDraft | null>(null);
  const [error, setError] = useState('');

  const gen = useMutation({
    mutationFn: () =>
      draftCampaign({
        brief: brief.trim(),
        goalMinor: goalMinor > 0 ? goalMinor : undefined,
        currency,
        state: state || undefined,
        lga: lga || undefined,
        isEmergency,
        organizationName,
      }),
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
      title={t('campaignAi.draftCampaign.title')}
      hint={t('campaignAi.draftCampaign.hint')}
      brief={brief}
      onBriefChange={setBrief}
      placeholder={t('campaignAi.draftCampaign.placeholder')}
      minBrief={20}
      onGenerate={() => gen.mutate()}
      isPending={gen.isPending}
      error={error}
      provenance={draft ?? undefined}
      onApply={
        draft
          ? () =>
              onApply({
                title: draft.title,
                summary: draft.summary,
                description: draft.description,
              })
          : undefined
      }
      applyLabel={t('campaignAi.draftCampaign.apply')}
    >
      {draft && (
        <>
          <AIWarnings warnings={draft.warnings} />
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {t('orgCampaigns.titleLabel')}
            </p>
            <p className="font-semibold text-slate-900 dark:text-slate-100">{draft.title}</p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {t('orgCampaigns.summaryLabel')}
            </p>
            <p className="text-sm text-slate-700 dark:text-slate-200">{draft.summary}</p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {t('orgCampaigns.descriptionLabel')}
            </p>
            <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-200">
              {draft.description}
            </p>
          </div>
          {draft.milestones.length > 0 && (
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                {t('campaignAi.draftCampaign.suggestedPlan')}
              </p>
              {/* Not applied by "Use" — see the note at the top of this file. */}
              <ul className="mt-1 space-y-1 text-sm text-slate-700 dark:text-slate-200">
                {draft.milestones.map((m, i) => (
                  <li key={i}>
                    <strong>{m.title}</strong>
                    {m.targetMinor > 0 &&
                      ` — ${formatMoney(m.targetMinor, currency, i18n.language)}`}
                    {m.description && (
                      <span className="block text-xs text-slate-600 dark:text-slate-300">
                        {m.description}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {t('campaignAi.draftCampaign.planNote')}
              </p>
            </div>
          )}
        </>
      )}
    </CampaignAIPanel>
  );
}
