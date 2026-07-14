import { useMemo, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import {
  ConsultationQuestionType,
  ConsultationStatus,
  type ConsultationAnswerInput,
  type ConsultationQuestion,
} from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { UnverifiedBanner } from '../components/UnverifiedBanner';
import { getApiError, uploadUrl } from '../lib/api';
import { useMe } from '../hooks/useMe';
import {
  useConsultation,
  useConsultationOutcome,
  useConsultationQuestions,
  useMyConsultationResponses,
  useSubmitConsultationResponse,
} from '../hooks/useConsultations';

type AnswerState = {
  textValue?: string;
  selections: string[]; // for YES_NO stores ["YES"] or ["NO"]
};

function isAnswered(q: ConsultationQuestion, a: AnswerState | undefined) {
  if (!a) return false;
  switch (q.type) {
    case ConsultationQuestionType.SHORT_TEXT:
    case ConsultationQuestionType.LONG_TEXT:
      return !!a.textValue && a.textValue.trim() !== '';
    default:
      return a.selections.length > 0;
  }
}

// toAnswerPayload converts the in-page state into the API DTO shape.
// Empty answers are dropped so the server's "required unanswered"
// enforcement isn't confused by empty strings for optional questions.
function toAnswerPayload(
  questions: ConsultationQuestion[],
  answers: Record<string, AnswerState>,
): ConsultationAnswerInput[] {
  const out: ConsultationAnswerInput[] = [];
  for (const q of questions) {
    const a = answers[q.id];
    if (!a || !isAnswered(q, a)) continue;
    if (
      q.type === ConsultationQuestionType.SHORT_TEXT ||
      q.type === ConsultationQuestionType.LONG_TEXT
    ) {
      out.push({ questionId: q.id, textValue: a.textValue });
    } else {
      out.push({ questionId: q.id, selections: a.selections });
    }
  }
  return out;
}

export function ConsultationDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const me = useMe();

  const consultationQuery = useConsultation(id);
  const questionsQuery = useConsultationQuestions(id);
  const outcomeQuery = useConsultationOutcome(id);
  const respondedQuery = useMyConsultationResponses();
  const submitMutation = useSubmitConsultationResponse(id);

  const [answers, setAnswers] = useState<Record<string, AnswerState>>({});
  const [clientError, setClientError] = useState<string>('');

  const consultation = consultationQuery.data;
  const questions = useMemo(
    () => (questionsQuery.data ?? []).slice().sort((a, b) => a.position - b.position),
    [questionsQuery.data],
  );
  const outcome = outcomeQuery.data ?? null;
  const alreadyResponded = respondedQuery.data?.has(id ?? '') ?? false;
  const emailVerified = me.data?.emailVerified ?? false;

  const canSubmit =
    !!consultation &&
    consultation.status === ConsultationStatus.PUBLISHED &&
    !alreadyResponded &&
    emailVerified &&
    !submitMutation.isPending;

  function updateAnswer(q: ConsultationQuestion, patch: Partial<AnswerState>) {
    setAnswers((prev) => {
      const current = prev[q.id] ?? { selections: [] };
      return { ...prev, [q.id]: { ...current, ...patch } };
    });
    setClientError('');
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    setClientError('');
    if (!consultation || !questions.length) return;

    const missingRequired = questions.some((q) => q.required && !isAnswered(q, answers[q.id]));
    if (missingRequired) {
      setClientError(t('consultationDetail.errors.required'));
      return;
    }

    const payload = toAnswerPayload(questions, answers);
    if (payload.length === 0) {
      setClientError(t('consultationDetail.errors.empty'));
      return;
    }
    submitMutation.mutate({ answers: payload });
  }

  if (consultationQuery.isLoading || questionsQuery.isLoading) {
    return <p className="text-sm text-slate-600 dark:text-slate-400">{t('common.loading')}</p>;
  }

  if (consultationQuery.isError || !consultation) {
    return (
      <section className="space-y-4">
        <Link
          to="/consultations"
          className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
        >
          {t('consultationDetail.back')}
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">
          {t('consultationDetail.loadError')}
        </p>
      </section>
    );
  }

  const isClosed = consultation.status === ConsultationStatus.CLOSED;
  const isDraft = consultation.status === ConsultationStatus.DRAFT;
  const serverError = submitMutation.isError ? getApiError(submitMutation.error) : null;

  return (
    <section className="space-y-6">
      <Link
        to="/consultations"
        className="text-sm font-semibold text-civic-700 dark:text-civic-200 hover:underline"
      >
        {t('consultationDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t(`consultationsPage.status.${consultation.status}`)}
        title={consultation.title}
        subtitle={consultation.summary}
        titleAs="h2"
      />

      {consultation.coverImageUrl && (
        <img
          src={uploadUrl(consultation.coverImageUrl)}
          alt=""
          className="h-48 w-full rounded-2xl border border-slate-200 object-cover shadow-sm sm:h-64"
        />
      )}

      {consultation.description && (
        <article className="prose prose-slate max-w-none rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-6 text-sm shadow-sm">
          {/* Description is markdown per the API contract; rendering as
              plain text here is deliberate. When the citizen app gains a
              markdown renderer for issues/petitions, switch this too. */}
          <p className="whitespace-pre-wrap text-slate-700 dark:text-slate-300">
            {consultation.description}
          </p>
        </article>
      )}

      {isDraft && (
        <p className="rounded-lg border border-amber-200 dark:border-amber-500/40 bg-amber-50 dark:bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-200">
          {t('consultationDetail.draftNotice')}
        </p>
      )}

      {alreadyResponded && !isDraft && (
        <p className="rounded-lg border border-emerald-200 bg-emerald-50 dark:bg-emerald-500/10 p-4 text-sm text-emerald-800">
          {t('consultationDetail.alreadyResponded')}
        </p>
      )}

      {!alreadyResponded && !emailVerified && <UnverifiedBanner />}

      {outcome && (
        <section
          id="outcome"
          className="space-y-4 rounded-2xl border border-emerald-200 bg-emerald-50/50 dark:bg-emerald-500/10 p-6 shadow-sm"
        >
          <h3 className="font-fraunces text-lg font-semibold text-emerald-900">
            {t('consultationDetail.outcome.heading')}
          </h3>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
              {t('consultationDetail.outcome.summary')}
            </p>
            <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
              {outcome.summary}
            </p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
              {t('consultationDetail.outcome.decisions')}
            </p>
            <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
              {outcome.decisions}
            </p>
          </div>
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
              {t('consultationDetail.outcome.nextSteps')}
            </p>
            <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-300">
              {outcome.nextSteps}
            </p>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            {t('consultationDetail.outcome.publishedBy', {
              name: outcome.authorName,
              date: new Date(outcome.publishedAt).toLocaleDateString(),
            })}
          </p>
        </section>
      )}

      {/* Show the form only if the consultation is open AND the user hasn't
          already responded. Otherwise, render the questions in read-only
          mode so the record is still visible. */}
      {consultation.status === ConsultationStatus.PUBLISHED && !alreadyResponded ? (
        <form onSubmit={onSubmit} className="space-y-6">
          {questions.map((q, idx) => (
            <QuestionField
              key={q.id}
              index={idx + 1}
              question={q}
              answer={answers[q.id]}
              onChange={(patch) => updateAnswer(q, patch)}
            />
          ))}

          {clientError && (
            <p className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-700">
              {clientError}
            </p>
          )}
          {serverError && (
            <p className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-700">
              {mapServerError(t, serverError.code, serverError.message)}
            </p>
          )}
          {submitMutation.isSuccess && (
            <p className="rounded-lg border border-emerald-200 bg-emerald-50 dark:bg-emerald-500/10 p-3 text-sm text-emerald-800">
              {t('consultationDetail.submitted')}
            </p>
          )}

          <div className="flex justify-end">
            <Button type="submit" disabled={!canSubmit}>
              {submitMutation.isPending
                ? t('consultationDetail.submitting')
                : t('consultationDetail.submit')}
            </Button>
          </div>
        </form>
      ) : (
        <section className="space-y-4">
          <h3 className="font-fraunces text-base font-semibold text-slate-900 dark:text-slate-100">
            {t('consultationDetail.questionsHeading')}
          </h3>
          {questions.map((q, idx) => (
            <div
              key={q.id}
              className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-4 text-sm shadow-sm"
            >
              <p className="font-semibold text-slate-800">
                {idx + 1}. {q.prompt}
                {q.required && <span className="ml-1 text-red-500">*</span>}
              </p>
              {q.helpText && (
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{q.helpText}</p>
              )}
              <p className="mt-2 text-xs uppercase tracking-wide text-slate-400">
                {t(`consultationDetail.questionTypes.${q.type}`)}
              </p>
              {q.options.length > 0 && (
                <ul className="mt-2 list-disc pl-5 text-xs text-slate-600 dark:text-slate-400">
                  {q.options.map((o) => (
                    <li key={o}>{o}</li>
                  ))}
                </ul>
              )}
            </div>
          ))}
          {isClosed && !outcome && (
            <p className="rounded-lg border border-amber-200 dark:border-amber-500/40 bg-amber-50 dark:bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-200">
              {t('consultationDetail.closedNoOutcome')}
            </p>
          )}
        </section>
      )}
    </section>
  );
}

// mapServerError translates known server codes into i18n keys. Unknown
// codes fall through to the message the server sent so operators can
// still see something meaningful.
function mapServerError(t: (key: string) => string, code: string, fallbackMessage: string): string {
  switch (code) {
    case 'ALREADY_RESPONDED':
      return t('consultationDetail.errors.alreadyResponded');
    case 'NOT_ACCEPTING_RESPONSES':
      return t('consultationDetail.errors.notAccepting');
    case 'PAST_DEADLINE':
      return t('consultationDetail.errors.pastDeadline');
    case 'REQUIRED_UNANSWERED':
      return t('consultationDetail.errors.required');
    case 'UNKNOWN_OPTION':
      return t('consultationDetail.errors.unknownOption');
    case 'EMAIL_NOT_VERIFIED':
      return t('consultationDetail.errors.emailNotVerified');
    default:
      return fallbackMessage;
  }
}

// QuestionField renders one of five input types based on question.type.
// Kept in-file — this is the only place these controls are used.
function QuestionField({
  index,
  question,
  answer,
  onChange,
}: {
  index: number;
  question: ConsultationQuestion;
  answer: AnswerState | undefined;
  onChange: (patch: Partial<AnswerState>) => void;
}) {
  const { t } = useTranslation();
  const label = (
    <label className="block text-sm font-semibold text-slate-800">
      {index}. {question.prompt}
      {question.required && <span className="ml-1 text-red-500">*</span>}
    </label>
  );
  const help = question.helpText ? (
    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{question.helpText}</p>
  ) : null;
  const wrap = (children: React.ReactNode) => (
    <div className="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60 p-4 shadow-sm">
      {label}
      {help}
      <div className="mt-3">{children}</div>
    </div>
  );

  switch (question.type) {
    case ConsultationQuestionType.SHORT_TEXT:
      return wrap(
        <input
          type="text"
          value={answer?.textValue ?? ''}
          onChange={(e) => onChange({ textValue: e.target.value })}
          maxLength={500}
          className="w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
        />,
      );
    case ConsultationQuestionType.LONG_TEXT:
      return wrap(
        <textarea
          value={answer?.textValue ?? ''}
          onChange={(e) => onChange({ textValue: e.target.value })}
          rows={4}
          maxLength={5000}
          className="w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
        />,
      );
    case ConsultationQuestionType.SINGLE_CHOICE:
      return wrap(
        <fieldset className="space-y-2">
          <legend className="sr-only">{question.prompt}</legend>
          {question.options.map((opt) => (
            <label
              key={opt}
              className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300"
            >
              <input
                type="radio"
                name={`q-${question.id}`}
                value={opt}
                checked={(answer?.selections?.[0] ?? '') === opt}
                onChange={() => onChange({ selections: [opt] })}
                className="text-civic-600 dark:text-civic-300 focus:ring-civic-500"
              />
              {opt}
            </label>
          ))}
        </fieldset>,
      );
    case ConsultationQuestionType.MULTI_CHOICE: {
      const set = new Set(answer?.selections ?? []);
      return wrap(
        <fieldset className="space-y-2">
          <legend className="sr-only">{question.prompt}</legend>
          {question.options.map((opt) => (
            <label
              key={opt}
              className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300"
            >
              <input
                type="checkbox"
                value={opt}
                checked={set.has(opt)}
                onChange={(e) => {
                  const next = new Set(set);
                  if (e.target.checked) next.add(opt);
                  else next.delete(opt);
                  onChange({ selections: Array.from(next) });
                }}
                className="rounded text-civic-600 dark:text-civic-300 focus:ring-civic-500"
              />
              {opt}
            </label>
          ))}
        </fieldset>,
      );
    }
    case ConsultationQuestionType.YES_NO:
      return wrap(
        <fieldset className="flex gap-4">
          <legend className="sr-only">{question.prompt}</legend>
          {(['YES', 'NO'] as const).map((v) => (
            <label
              key={v}
              className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300"
            >
              <input
                type="radio"
                name={`q-${question.id}`}
                value={v}
                checked={(answer?.selections?.[0] ?? '') === v}
                onChange={() => onChange({ selections: [v] })}
                className="text-civic-600 dark:text-civic-300 focus:ring-civic-500"
              />
              {t(`consultationDetail.yesNo.${v}`)}
            </label>
          ))}
        </fieldset>,
      );
    default:
      return null;
  }
}
