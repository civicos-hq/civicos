import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Sparkles, RefreshCw, CheckCircle2 } from 'lucide-react';
import { Button } from '@civicos/ui';
import {
  draftAnnouncement,
  type AnnouncementDraft,
  type DraftAudience,
  type DraftTone,
} from '../../lib/civicai';
import { getApiError } from '../../lib/api';

// DraftWithAIPanel wraps a Gemini-backed announcement drafter. The parent
// owns the actual title + body form fields; this panel produces a draft
// and hands it back via onApply. Keeping the parent in control means the
// AI draft is a *starting point*, never the source of truth.

interface Props {
  orgName?: string;
  orgKind?: string;
  onApply: (draft: AnnouncementDraft) => void;
}

const TONES: DraftTone[] = ['formal', 'friendly', 'urgent', 'empathetic'];
const AUDIENCES: DraftAudience[] = ['all', 'members'];
const MIN_BRIEF = 20;

export function DraftWithAIPanel({ orgName, orgKind, onApply }: Props) {
  const { t } = useTranslation();
  const [brief, setBrief] = useState('');
  const [tone, setTone] = useState<DraftTone>('friendly');
  const [audience, setAudience] = useState<DraftAudience>('all');
  const [draft, setDraft] = useState<AnnouncementDraft | null>(null);
  const [applied, setApplied] = useState(false);

  const mutation = useMutation({
    mutationFn: () =>
      draftAnnouncement({
        brief: brief.trim(),
        tone,
        audience,
        orgName,
        orgKind,
      }),
    onSuccess: (data) => {
      setDraft(data);
      setApplied(false);
    },
  });

  const canGenerate = brief.trim().length >= MIN_BRIEF && !mutation.isPending;
  const errorMsg = mutation.isError
    ? (getApiError(mutation.error)?.message ?? t('civicai.draft.genericError'))
    : null;

  function apply() {
    if (!draft) return;
    onApply(draft);
    setApplied(true);
  }

  return (
    <section className="rounded-2xl border border-civic-200/70 dark:border-civic-500/30 bg-gradient-to-br from-civic-50/60 to-white dark:from-civic-500/10 dark:to-slate-900 p-4 md:p-5 shadow-sm">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-8 w-8 items-center justify-center rounded-full bg-civic-100 dark:bg-civic-500/25 text-civic-700 dark:text-civic-200">
            <Sparkles className="h-4 w-4" aria-hidden="true" />
          </span>
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {t('civicai.draft.title')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('civicai.draft.subtitle')}
            </p>
          </div>
        </div>
        <span className="rounded-full bg-white dark:bg-slate-800 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400 ring-1 ring-slate-200 dark:ring-slate-700">
          {t('civicai.draft.reviewBadge')}
        </span>
      </header>

      <div className="mt-4 space-y-3">
        <div>
          <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
            {t('civicai.draft.briefLabel')}
          </label>
          <textarea
            value={brief}
            onChange={(e) => setBrief(e.target.value)}
            rows={4}
            minLength={MIN_BRIEF}
            placeholder={t('civicai.draft.briefPlaceholder')}
            className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
          <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
            {t('civicai.draft.briefHelp', { min: MIN_BRIEF })}
          </p>
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <SegmentedControl
            label={t('civicai.draft.toneLabel')}
            options={TONES}
            value={tone}
            onChange={setTone}
            labelFor={(v) => t(`civicai.draft.tone.${v}`)}
          />
          <SegmentedControl
            label={t('civicai.draft.audienceLabel')}
            options={AUDIENCES}
            value={audience}
            onChange={setAudience}
            labelFor={(v) => t(`civicai.draft.audience.${v}`)}
          />
        </div>

        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button
            type="button"
            variant={draft ? 'secondary' : 'primary'}
            onClick={() => mutation.mutate()}
            loading={mutation.isPending}
            disabled={!canGenerate}
          >
            {draft ? (
              <span className="flex items-center gap-1.5">
                <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
                {t('civicai.draft.regenerate')}
              </span>
            ) : (
              t('civicai.draft.generate')
            )}
          </Button>
        </div>

        {errorMsg && (
          <p className="rounded-lg bg-rose-50 dark:bg-rose-500/15 px-3 py-2 text-xs text-rose-700 dark:text-rose-300">
            {errorMsg}
          </p>
        )}

        {draft && <DraftPreview draft={draft} applied={applied} onApply={apply} />}
      </div>
    </section>
  );
}

function DraftPreview({
  draft,
  applied,
  onApply,
}: {
  draft: AnnouncementDraft;
  applied: boolean;
  onApply: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="mt-2 space-y-3 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900/60 p-4">
      <div>
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          {t('civicai.draft.previewTitle')}
        </h4>
        <p className="mt-1 text-sm font-semibold text-slate-900 dark:text-slate-100">
          {draft.title}
        </p>
      </div>

      <div>
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          {t('civicai.draft.previewBody')}
        </h4>
        <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-slate-800 dark:text-slate-200">
          {draft.body}
        </p>
      </div>

      {draft.keyPoints?.length > 0 && (
        <div>
          <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            {t('civicai.draft.previewKeyPoints')}
          </h4>
          <ul className="mt-1 space-y-1 text-sm text-slate-800 dark:text-slate-200">
            {draft.keyPoints.map((p, i) => (
              <li key={i} className="flex items-start gap-2">
                <span className="mt-1.5 h-1.5 w-1.5 flex-none rounded-full bg-civic-500/70 dark:bg-civic-400/70" />
                <span>{p}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex items-center justify-between gap-2 border-t border-slate-200 dark:border-slate-700 pt-3">
        <span className="text-[11px] text-slate-500 dark:text-slate-400">
          {t('civicai.draft.meta', { model: draft.model, tone: draft.tone })}
        </span>
        <Button type="button" variant={applied ? 'secondary' : 'primary'} onClick={onApply}>
          {applied ? (
            <span className="flex items-center gap-1.5">
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
              {t('civicai.draft.applied')}
            </span>
          ) : (
            t('civicai.draft.applyToForm')
          )}
        </Button>
      </div>
    </div>
  );
}

function SegmentedControl<T extends string>({
  label,
  options,
  value,
  onChange,
  labelFor,
}: {
  label: string;
  options: T[];
  value: T;
  onChange: (v: T) => void;
  labelFor: (v: T) => string;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
        {label}
      </label>
      <div
        role="radiogroup"
        className="inline-flex flex-wrap gap-1 rounded-lg bg-slate-100 dark:bg-slate-800 p-1"
      >
        {options.map((opt) => {
          const active = opt === value;
          return (
            <button
              key={opt}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => onChange(opt)}
              className={
                active
                  ? 'rounded-md bg-white dark:bg-slate-900 px-2.5 py-1 text-xs font-semibold text-civic-700 dark:text-civic-200 shadow-sm'
                  : 'rounded-md px-2.5 py-1 text-xs font-medium text-slate-600 dark:text-slate-300 hover:text-civic-700 dark:hover:text-civic-200'
              }
            >
              {labelFor(opt)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
