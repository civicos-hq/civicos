import { useEffect, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { GripVertical } from 'lucide-react';
import { Button, Input } from '@civicos/ui';
import {
  ConsultationQuestionType,
  ConsultationStatus,
  type ConsultationQuestion,
  type ConsultationQuestionInput,
} from '@civicos/types';
import { PageHeader } from '../components/PageHeader';
import { getApiError } from '../lib/api';
import {
  useAddConsultationQuestion,
  useCloseConsultation,
  useConsultation,
  useConsultationAnalytics,
  useConsultationOutcome,
  useConsultationQuestions,
  useDeleteConsultation,
  useDeleteConsultationQuestion,
  useMyOrganizations,
  usePublishConsultation,
  usePublishConsultationOutcome,
  useReorderConsultationQuestions,
  useUpdateConsultationQuestion,
} from '../hooks/useConsultations';

const QUESTION_TYPES: Array<{ value: ConsultationQuestionType; i18n: string }> = [
  {
    value: ConsultationQuestionType.SHORT_TEXT,
    i18n: 'consultationDetail.questionTypes.SHORT_TEXT',
  },
  { value: ConsultationQuestionType.LONG_TEXT, i18n: 'consultationDetail.questionTypes.LONG_TEXT' },
  {
    value: ConsultationQuestionType.SINGLE_CHOICE,
    i18n: 'consultationDetail.questionTypes.SINGLE_CHOICE',
  },
  {
    value: ConsultationQuestionType.MULTI_CHOICE,
    i18n: 'consultationDetail.questionTypes.MULTI_CHOICE',
  },
  { value: ConsultationQuestionType.YES_NO, i18n: 'consultationDetail.questionTypes.YES_NO' },
];

// Only the two choice types accept an options list — everything else
// hides that field.
function typeNeedsOptions(t: ConsultationQuestionType): boolean {
  return (
    t === ConsultationQuestionType.SINGLE_CHOICE || t === ConsultationQuestionType.MULTI_CHOICE
  );
}

export function OrgConsultationDetailPage() {
  const { t } = useTranslation();
  const { orgId, id } = useParams<{ orgId: string; id: string }>();

  const { data: memberships = [] } = useMyOrganizations();
  const membership = memberships.find((m) => m.organization.id === orgId);

  const consultation = useConsultation(id);
  const questions = useConsultationQuestions(id);
  const outcome = useConsultationOutcome(id);
  const analytics = useConsultationAnalytics(id);

  const publishMutation = usePublishConsultation(id);
  const closeMutation = useCloseConsultation(id);
  const deleteMutation = useDeleteConsultation(id);

  const [globalError, setGlobalError] = useState('');

  async function runMutation(fn: () => Promise<unknown>, refreshOnly = false) {
    setGlobalError('');
    try {
      await fn();
    } catch (err) {
      const apiErr = getApiError(err);
      setGlobalError(apiErr?.message ?? t('orgConsultationDetail.errors.generic'));

      refreshOnly;
    }
  }

  if (consultation.isLoading) {
    return <p className="text-sm text-slate-600">{t('common.loading')}</p>;
  }
  if (consultation.isError || !consultation.data || !id || !orgId) {
    return (
      <section className="space-y-4">
        <Link to={`/org/${orgId ?? ''}`} className="text-sm font-semibold text-civic-700">
          {t('orgConsultationDetail.back')}
        </Link>
        <p className="text-sm text-red-600">{t('orgConsultationDetail.loadError')}</p>
      </section>
    );
  }

  const c = consultation.data;
  const isDraft = c.status === ConsultationStatus.DRAFT;
  const isPublished = c.status === ConsultationStatus.PUBLISHED;
  const isClosed = c.status === ConsultationStatus.CLOSED;
  const hasOutcome = !!outcome.data;
  const orgAdmin =
    !!membership &&
    (membership.membership.role === 'OWNER' || membership.membership.role === 'ADMIN');

  return (
    <section className="space-y-6">
      <Link to={`/org/${orgId}`} className="text-sm font-semibold text-civic-700 hover:underline">
        {t('orgConsultationDetail.back')}
      </Link>

      <PageHeader
        eyebrow={t(`consultationsPage.status.${c.status}`)}
        title={c.title}
        subtitle={c.summary}
        titleAs="h2"
        actions={
          orgAdmin && (
            <div className="flex gap-2">
              {isDraft && (
                <>
                  <Button
                    size="sm"
                    onClick={() => runMutation(() => publishMutation.mutateAsync())}
                    disabled={publishMutation.isPending}
                  >
                    {publishMutation.isPending
                      ? t('orgConsultationDetail.publishing')
                      : t('orgConsultationDetail.publish')}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      if (window.confirm(t('orgConsultationDetail.confirmDelete'))) {
                        runMutation(() => deleteMutation.mutateAsync());
                      }
                    }}
                    disabled={deleteMutation.isPending}
                  >
                    {t('common.delete')}
                  </Button>
                </>
              )}
              {isPublished && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    if (window.confirm(t('orgConsultationDetail.confirmClose'))) {
                      runMutation(() => closeMutation.mutateAsync());
                    }
                  }}
                  disabled={closeMutation.isPending}
                >
                  {closeMutation.isPending
                    ? t('orgConsultationDetail.closing')
                    : t('orgConsultationDetail.close')}
                </Button>
              )}
            </div>
          )
        }
      />

      {globalError && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {globalError}
        </p>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="whitespace-pre-wrap text-sm text-slate-700">{c.description}</p>
      </article>

      {/* Questions — edit builder in DRAFT, read-only otherwise.
          Drag-to-reorder is only available while the caller is an org
          admin and the consultation is still DRAFT — matches the
          server-side NOT_DRAFT guard on the reorder endpoint. */}
      <section>
        <h3 className="mb-3 font-fraunces text-base font-semibold text-slate-900">
          {t('orgConsultationDetail.questionsHeading')}
        </h3>
        <SortableQuestions
          consultationId={id}
          questions={questions.data ?? []}
          reorderable={isDraft && orgAdmin}
          editable={isDraft && orgAdmin}
        />
        {isDraft && orgAdmin && <QuestionAdder consultationId={id} />}
      </section>

      {/* Analytics — visible from PUBLISHED onwards to org members. */}
      {!isDraft && analytics.data && (
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="mb-3 font-fraunces text-base font-semibold text-slate-900">
            {t('orgConsultationDetail.analyticsHeading', { count: analytics.data.responseCount })}
          </h3>
          {analytics.data.responseCount === 0 ? (
            <p className="text-sm text-slate-600">{t('orgConsultationDetail.noResponses')}</p>
          ) : (
            <ul className="space-y-4">
              {analytics.data.questions.map((agg) => (
                <li
                  key={agg.questionId}
                  className="border-t border-slate-100 pt-3 first:border-none first:pt-0"
                >
                  <p className="text-sm font-semibold text-slate-800">{agg.prompt}</p>
                  {agg.optionCounts && (
                    <ul className="mt-2 space-y-1">
                      {Object.entries(agg.optionCounts)
                        .sort(([, a], [, b]) => b - a)
                        .map(([opt, n]) => (
                          <li key={opt} className="flex items-center gap-3 text-xs text-slate-600">
                            <span className="w-32 truncate">{opt}</span>
                            <span className="h-2 flex-1 overflow-hidden rounded-full bg-slate-100">
                              <span
                                className="block h-full bg-civic-500"
                                style={{
                                  width: `${Math.round((n / Math.max(agg.answerCount, 1)) * 100)}%`,
                                }}
                              />
                            </span>
                            <span className="w-10 text-right">{n}</span>
                          </li>
                        ))}
                    </ul>
                  )}
                  {agg.textValues && agg.textValues.length > 0 && (
                    <ul className="mt-2 space-y-1 text-xs italic text-slate-600">
                      {agg.textValues.slice(0, 8).map((v, i) => (
                        <li key={i}>&ldquo;{v}&rdquo;</li>
                      ))}
                      {agg.textValues.length > 8 && (
                        <li className="not-italic">
                          {t('orgConsultationDetail.andMore', { n: agg.textValues.length - 8 })}
                        </li>
                      )}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Outcome — form when CLOSED without outcome, view when present. */}
      {isClosed && !hasOutcome && orgAdmin && <OutcomeForm consultationId={id} />}

      {hasOutcome && outcome.data && (
        <section className="space-y-3 rounded-2xl border border-emerald-200 bg-emerald-50/50 p-6 shadow-sm">
          <h3 className="font-fraunces text-lg font-semibold text-emerald-900">
            {t('consultationDetail.outcome.heading')}
          </h3>
          <p className="whitespace-pre-wrap text-sm text-slate-700">
            <strong>{t('consultationDetail.outcome.summary')}:</strong> {outcome.data.summary}
          </p>
          <p className="whitespace-pre-wrap text-sm text-slate-700">
            <strong>{t('consultationDetail.outcome.decisions')}:</strong> {outcome.data.decisions}
          </p>
          <p className="whitespace-pre-wrap text-sm text-slate-700">
            <strong>{t('consultationDetail.outcome.nextSteps')}:</strong> {outcome.data.nextSteps}
          </p>
        </section>
      )}
    </section>
  );
}

// ── Question builder pieces ──────────────────────────────────────

// SortableQuestions renders the question list and — when `reorderable`
// — layers on drag-to-reorder. The local `items` state is the source of
// truth mid-drag so the row visibly moves before the server confirms.
// On drop the full ordering is PATCHed; on error we reset from the
// server's copy on the next invalidation.
function SortableQuestions({
  consultationId,
  questions,
  reorderable,
  editable,
}: {
  consultationId: string;
  questions: ConsultationQuestion[];
  reorderable: boolean;
  editable: boolean;
}) {
  const reorderMutation = useReorderConsultationQuestions(consultationId);
  const [items, setItems] = useState<ConsultationQuestion[]>(questions);

  // Keep local order aligned with the server whenever the query
  // refetches — this is what un-does an optimistic drop if the PATCH
  // fails and the invalidation returns the pre-drag ordering.
  useEffect(() => {
    setItems(questions);
  }, [questions]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  function onDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = items.findIndex((q) => q.id === active.id);
    const newIndex = items.findIndex((q) => q.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    const next = arrayMove(items, oldIndex, newIndex);
    setItems(next);
    const ordering: Record<string, number> = {};
    next.forEach((q, i) => {
      ordering[q.id] = i;
    });
    reorderMutation.mutate(ordering);
  }

  // Non-draft or non-admin path: no drag context, no handle — plain
  // read-only rendering keeps the DOM stable for responders.
  if (!reorderable) {
    return (
      <ul className="space-y-3">
        {items.map((q, i) => (
          <QuestionRow
            key={q.id}
            index={i + 1}
            consultationId={consultationId}
            question={q}
            editable={editable}
          />
        ))}
      </ul>
    );
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <SortableContext items={items.map((q) => q.id)} strategy={verticalListSortingStrategy}>
        <ul className="space-y-3">
          {items.map((q, i) => (
            <QuestionRow
              key={q.id}
              index={i + 1}
              consultationId={consultationId}
              question={q}
              editable={editable}
              draggable
            />
          ))}
        </ul>
      </SortableContext>
    </DndContext>
  );
}

function QuestionRow({
  index,
  consultationId,
  question,
  editable,
  draggable = false,
}: {
  index: number;
  consultationId: string;
  question: ConsultationQuestion;
  editable: boolean;
  draggable?: boolean;
}) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const deleteMutation = useDeleteConsultationQuestion(consultationId, question.id);
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: question.id,
    disabled: !draggable,
  });
  const style = draggable
    ? {
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.6 : 1,
      }
    : undefined;

  if (editing) {
    return (
      <li
        ref={draggable ? setNodeRef : undefined}
        style={style}
        className="rounded-2xl border border-civic-200 bg-white p-4 shadow-sm"
      >
        <QuestionForm
          consultationId={consultationId}
          initial={question}
          onDone={() => setEditing(false)}
        />
      </li>
    );
  }

  return (
    <li
      ref={draggable ? setNodeRef : undefined}
      style={style}
      className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
    >
      <div className="flex items-start justify-between gap-3">
        {draggable && (
          // Deliberately a span, not a button — a native <button> would
          // swallow Space as a click and beat dnd-kit's KeyboardSensor
          // to the event, breaking keyboard drag pickup.
          <span
            role="button"
            tabIndex={0}
            className="mt-0.5 inline-flex cursor-grab touch-none text-slate-400 hover:text-slate-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-civic-500 active:cursor-grabbing"
            aria-label={t('orgConsultationDetail.dragHandle')}
            {...attributes}
            {...listeners}
          >
            <GripVertical size={18} />
          </span>
        )}
        <div className="flex-1">
          <p className="text-sm font-semibold text-slate-800">
            {index}. {question.prompt}
            {question.required && <span className="ml-1 text-red-500">*</span>}
          </p>
          {question.helpText && <p className="mt-1 text-xs text-slate-500">{question.helpText}</p>}
          <p className="mt-1 text-xs uppercase tracking-wide text-slate-400">
            {t(`consultationDetail.questionTypes.${question.type}`)}
          </p>
          {question.options.length > 0 && (
            <ul className="mt-1 list-disc pl-5 text-xs text-slate-600">
              {question.options.map((o) => (
                <li key={o}>{o}</li>
              ))}
            </ul>
          )}
        </div>
        {editable && (
          <div className="flex flex-shrink-0 gap-2">
            <Button size="sm" variant="ghost" onClick={() => setEditing(true)}>
              {t('common.edit')}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (window.confirm(t('orgConsultationDetail.confirmDeleteQuestion'))) {
                  deleteMutation.mutate();
                }
              }}
            >
              {t('common.delete')}
            </Button>
          </div>
        )}
      </div>
    </li>
  );
}

function QuestionAdder({ consultationId }: { consultationId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  if (!open) {
    return (
      <div className="mt-3">
        <Button size="sm" onClick={() => setOpen(true)}>
          {t('orgConsultationDetail.addQuestion')}
        </Button>
      </div>
    );
  }
  return (
    <div className="mt-3 rounded-2xl border border-civic-200 bg-white p-4 shadow-sm">
      <QuestionForm consultationId={consultationId} onDone={() => setOpen(false)} />
    </div>
  );
}

function QuestionForm({
  consultationId,
  initial,
  onDone,
}: {
  consultationId: string;
  initial?: ConsultationQuestion;
  onDone: () => void;
}) {
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState(initial?.prompt ?? '');
  const [helpText, setHelpText] = useState(initial?.helpText ?? '');
  const [type, setType] = useState<ConsultationQuestionType>(
    initial?.type ?? ConsultationQuestionType.SHORT_TEXT,
  );
  const [options, setOptions] = useState<string>((initial?.options ?? []).join('\n'));
  const [required, setRequired] = useState<boolean>(initial?.required ?? false);
  const [error, setError] = useState('');

  const addMutation = useAddConsultationQuestion(consultationId);
  const updateMutation = useUpdateConsultationQuestion(consultationId, initial?.id);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    const opts = options
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    if (typeNeedsOptions(type) && opts.length < 2) {
      setError(t('orgConsultationDetail.errors.optionsRequired'));
      return;
    }
    const input: ConsultationQuestionInput = {
      prompt: prompt.trim(),
      helpText: helpText.trim() || undefined,
      type,
      options: typeNeedsOptions(type) ? opts : [],
      required,
    };
    try {
      if (initial) {
        await updateMutation.mutateAsync(input);
      } else {
        await addMutation.mutateAsync(input);
      }
      onDone();
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgConsultationDetail.errors.generic'));
    }
  }

  const isSaving = addMutation.isPending || updateMutation.isPending;

  return (
    <form onSubmit={onSubmit} className="space-y-3">
      <div>
        <label className="block text-sm font-semibold text-slate-700">
          {t('orgConsultationDetail.fields.prompt')}
        </label>
        <Input
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          required
          minLength={3}
          maxLength={500}
        />
      </div>
      <div>
        <label className="block text-sm font-semibold text-slate-700">
          {t('orgConsultationDetail.fields.helpText')}
        </label>
        <Input value={helpText} onChange={(e) => setHelpText(e.target.value)} />
      </div>
      <div>
        <label className="block text-sm font-semibold text-slate-700">
          {t('orgConsultationDetail.fields.type')}
        </label>
        <select
          value={type}
          onChange={(e) => setType(e.target.value as ConsultationQuestionType)}
          className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
        >
          {QUESTION_TYPES.map((qt) => (
            <option key={qt.value} value={qt.value}>
              {t(qt.i18n)}
            </option>
          ))}
        </select>
      </div>
      {typeNeedsOptions(type) && (
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('orgConsultationDetail.fields.options')}
          </label>
          <textarea
            value={options}
            onChange={(e) => setOptions(e.target.value)}
            rows={4}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
            placeholder={t('orgConsultationDetail.fields.optionsPlaceholder')}
          />
          <p className="mt-1 text-xs text-slate-500">
            {t('orgConsultationDetail.fields.optionsHelp')}
          </p>
        </div>
      )}
      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input
          type="checkbox"
          checked={required}
          onChange={(e) => setRequired(e.target.checked)}
          className="rounded text-civic-600 focus:ring-civic-500"
        />
        {t('orgConsultationDetail.fields.required')}
      </label>
      {error && (
        <p className="rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-700">
          {error}
        </p>
      )}
      <div className="flex justify-end gap-2">
        <Button size="sm" variant="ghost" type="button" onClick={onDone}>
          {t('common.cancel')}
        </Button>
        <Button size="sm" type="submit" disabled={isSaving}>
          {isSaving ? t('common.saving') : t('common.save')}
        </Button>
      </div>
    </form>
  );
}

function OutcomeForm({ consultationId }: { consultationId: string }) {
  const { t } = useTranslation();
  const [summary, setSummary] = useState('');
  const [decisions, setDecisions] = useState('');
  const [nextSteps, setNextSteps] = useState('');
  const [error, setError] = useState('');
  const mutation = usePublishConsultationOutcome(consultationId);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await mutation.mutateAsync({ summary, decisions, nextSteps });
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgConsultationDetail.errors.generic'));
    }
  }

  return (
    <section className="space-y-3 rounded-2xl border border-emerald-300 bg-emerald-50 p-6 shadow-sm">
      <h3 className="font-fraunces text-base font-semibold text-emerald-900">
        {t('orgConsultationDetail.outcomeHeading')}
      </h3>
      <p className="text-xs text-emerald-800">{t('orgConsultationDetail.outcomeIntro')}</p>
      <form onSubmit={onSubmit} className="space-y-3">
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('consultationDetail.outcome.summary')}
          </label>
          <textarea
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            required
            minLength={10}
            rows={3}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
        </div>
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('consultationDetail.outcome.decisions')}
          </label>
          <textarea
            value={decisions}
            onChange={(e) => setDecisions(e.target.value)}
            required
            minLength={10}
            rows={3}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
        </div>
        <div>
          <label className="block text-sm font-semibold text-slate-700">
            {t('consultationDetail.outcome.nextSteps')}
          </label>
          <textarea
            value={nextSteps}
            onChange={(e) => setNextSteps(e.target.value)}
            required
            minLength={10}
            rows={3}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500"
          />
        </div>
        {error && (
          <p className="rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </p>
        )}
        <div className="flex justify-end">
          <Button size="sm" type="submit" disabled={mutation.isPending}>
            {mutation.isPending
              ? t('orgConsultationDetail.publishingOutcome')
              : t('orgConsultationDetail.publishOutcome')}
          </Button>
        </div>
      </form>
    </section>
  );
}
