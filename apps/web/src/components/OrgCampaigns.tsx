import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { CampaignDraftAssist } from './civicai/CampaignDraftAssist';
import { CompletionReportAssist } from './civicai/UpdateDraftAssist';
import { Link } from 'react-router-dom';
import { Button, Input } from '@civicos/ui';
import { Modal } from './Modal';
import {
  formatMoney,
  useCampaignLifecycle,
  useCreateCampaign,
  useCreateMilestone,
  useOrgCampaigns,
  categoryKey,
  useUpdateCampaign,
  useFileReport,
  useCampaignSpend,
  needsFinalReport,
  useMilestones,
  useDeleteMilestone,
  useCompleteMilestone,
  useDeleteCampaign,
  isEditable,
  isDeletable,
  type CampaignCategory,
  type CreateCampaignInput,
  type OrgCampaign,
} from '../hooks/useCampaigns';

const CATEGORIES: CampaignCategory[] = [
  'EMERGENCY_RELIEF',
  'COMMUNITY_DEVELOPMENT',
  'EDUCATION',
  'HEALTHCARE',
  'ENVIRONMENT',
  'AGRICULTURE',
  'OTHER',
];

/**
 * An organization's own campaigns, on its dashboard.
 *
 * Members only. This shows statuses the public never sees — DRAFT,
 * PENDING_REVIEW, NEEDS_CHANGES, REJECTED — along with the reviewer's note,
 * which is a private conversation between the platform and the organization
 * and is deliberately absent from every public surface.
 */
export function OrgCampaigns({ orgId, locale }: { orgId: string; locale: string }) {
  const { t } = useTranslation();
  const q = useOrgCampaigns(orgId);
  const campaigns = q.data ?? [];

  return (
    <div className="space-y-3">
      {q.isLoading && (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('common.loading')}</p>
      )}
      {!q.isLoading && campaigns.length === 0 && (
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('orgCampaigns.empty')}</p>
      )}
      <ul className="space-y-2">
        {campaigns.map((c) => (
          <CampaignRow key={c.id} campaign={c} orgId={orgId} locale={locale} />
        ))}
      </ul>
    </div>
  );
}

function CampaignRow({
  campaign,
  orgId,
  locale,
}: {
  campaign: OrgCampaign;
  orgId: string;
  locale: string;
}) {
  const { t } = useTranslation();
  const lifecycle = useCampaignLifecycle(orgId);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const del = useDeleteCampaign(orgId);

  // Which lifecycle step the organization can take next. Everything else —
  // review, pause, resume, archive — belongs to the platform admin: an
  // organization approving its own fundraiser would defeat the review.
  const nextAction =
    campaign.status === 'DRAFT' || campaign.status === 'NEEDS_CHANGES'
      ? ('submit' as const)
      : campaign.status === 'APPROVED'
        ? ('publish' as const)
        : null;

  async function run(action: 'submit' | 'publish') {
    setError('');
    try {
      await lifecycle.mutateAsync({ campaignId: campaign.id, action });
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('orgCampaigns.actionError'));
    }
  }

  return (
    <li className="rounded-lg border border-slate-200 p-3 dark:border-slate-700">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate font-semibold text-slate-900 dark:text-slate-100">
            {campaign.title}
          </p>
          <p className="text-xs text-slate-600 dark:text-slate-300">
            {t(`campaigns.status.${campaign.status}`, campaign.status)} ·{' '}
            {formatMoney(campaign.raisedMinor, campaign.currency, locale)}
            {' / '}
            {formatMoney(campaign.goalMinor, campaign.currency, locale)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Only a publicly visible campaign has a page to link to. */}
          {['PUBLISHED', 'FUNDED', 'COMPLETED', 'REPORTED'].includes(campaign.status) && (
            <Link
              to={`/campaigns/${campaign.slug}`}
              className="text-sm font-semibold text-civic-700 hover:underline dark:text-civic-200"
            >
              {t('orgCampaigns.view')}
            </Link>
          )}
          {/* Content is frozen once a campaign leaves the org's hands. */}
          {isEditable(campaign.status) && (
            <Button size="sm" variant="secondary" onClick={() => setEditing(true)}>
              {t('orgCampaigns.edit')}
            </Button>
          )}
          {needsFinalReport(campaign.status) && (
            <FinalReportForm campaign={campaign} orgId={orgId} />
          )}
          {nextAction && (
            <Button size="sm" variant="secondary" onClick={() => run(nextAction)}>
              {t(`orgCampaigns.${nextAction}`)}
            </Button>
          )}
          {/* Drafts only. Anything already submitted is archived rather than
              deleted, so a campaign reviewers or donors have seen leaves a
              trail instead of vanishing. */}
          {isDeletable(campaign.status) &&
            (confirmingDelete ? (
              <span className="flex items-center gap-1 whitespace-nowrap text-xs">
                <button
                  type="button"
                  className="font-semibold text-red-600 dark:text-red-400"
                  onClick={() => del.mutate(campaign.id)}
                >
                  {t('orgCampaigns.confirmDelete')}
                </button>
                <button
                  type="button"
                  className="text-slate-600 dark:text-slate-300"
                  onClick={() => setConfirmingDelete(false)}
                >
                  {t('common.cancel')}
                </button>
              </span>
            ) : (
              <button
                type="button"
                className="text-xs text-slate-600 hover:underline dark:text-slate-300"
                onClick={() => setConfirmingDelete(true)}
              >
                {t('orgCampaigns.delete')}
              </button>
            ))}
        </div>
      </div>

      {/* The reviewer's feedback, shown only here. */}
      {campaign.status === 'NEEDS_CHANGES' && campaign.reviewNote && (
        <p className="mt-2 rounded border-l-2 border-amber-500 bg-amber-50 px-2 py-1 text-xs text-slate-700 dark:bg-amber-900/20 dark:text-slate-200">
          {campaign.reviewNote}
        </p>
      )}
      {campaign.status === 'DRAFT' && (
        <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">
          {t('orgCampaigns.draftHint')}
        </p>
      )}
      {error && <p className="mt-2 text-xs text-red-600 dark:text-red-400">{error}</p>}
      {/* Progress reporting on a live campaign — the one plan change allowed
          after review, and what notifies everyone who funded it. */}
      {['PUBLISHED', 'FUNDED'].includes(campaign.status) && (
        <LivePlanProgress campaign={campaign} orgId={orgId} />
      )}

      {editing && (
        <EditCampaignModal campaign={campaign} orgId={orgId} onClose={() => setEditing(false)} />
      )}
    </li>
  );
}

/** The values both the create and edit forms operate on. */
interface CampaignFields {
  title: string;
  summary: string;
  description: string;
  category: CampaignCategory;
  goalMajor: string;
  state: string;
  lga: string;
  isEmergency: boolean;
}

/** Major → minor happens once, at this boundary. */
function toMinor(major: string): number {
  const n = Number.parseFloat(major.replace(/,/g, ''));
  return Number.isFinite(n) && n > 0 ? Math.round(n * 100) : 0;
}

/**
 * Mirrors the server's binding rules, so an organization is not shown a 400
 * for a description four characters too short.
 */
function fieldsValid(f: CampaignFields): boolean {
  return (
    f.title.trim().length >= 4 &&
    f.summary.trim().length >= 10 &&
    f.description.trim().length >= 40 &&
    toMinor(f.goalMajor) > 0
  );
}

function fieldsToInput(f: CampaignFields): CreateCampaignInput {
  return {
    title: f.title.trim(),
    summary: f.summary.trim(),
    description: f.description.trim(),
    category: f.category,
    goalMinor: toMinor(f.goalMajor),
    state: f.state.trim() || undefined,
    lga: f.lga.trim() || undefined,
    isEmergency: f.isEmergency,
  };
}

const INPUT_CLASS =
  'w-full rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500';

/** The campaign content fields, shared so create and edit cannot drift. */
function CampaignFieldSet({
  f,
  set,
  showAssist = false,
}: {
  f: CampaignFields;
  set: <K extends keyof CampaignFields>(k: K, v: CampaignFields[K]) => void;
  showAssist?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <>
      {/* Create only. An edit is a correction to something a reviewer may
          already have read, not a rewrite. */}
      {showAssist && (
        <CampaignDraftAssist
          goalMinor={toMinor(f.goalMajor)}
          currency="NGN"
          state={f.state}
          lga={f.lga}
          isEmergency={f.isEmergency}
          onApply={(d) => {
            set('title', d.title);
            set('summary', d.summary);
            set('description', d.description);
          }}
        />
      )}
      <Input
        label={t('orgCampaigns.titleLabel')}
        name="title"
        value={f.title}
        onChange={(e) => set('title', e.target.value)}
        maxLength={160}
        required
      />

      <Field label={t('orgCampaigns.summaryLabel')} hint={t('orgCampaigns.summaryHint')}>
        <input
          className={INPUT_CLASS}
          name="summary"
          value={f.summary}
          onChange={(e) => set('summary', e.target.value)}
          maxLength={300}
          required
        />
      </Field>

      <Field label={t('orgCampaigns.descriptionLabel')} hint={t('orgCampaigns.descriptionHint')}>
        <textarea
          className={INPUT_CLASS}
          rows={3}
          name="description"
          value={f.description}
          onChange={(e) => set('description', e.target.value)}
          required
        />
      </Field>

      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t('orgCampaigns.categoryLabel')}>
          <select
            className={INPUT_CLASS}
            name="category"
            value={f.category}
            onChange={(e) => set('category', e.target.value as CampaignCategory)}
          >
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {t(`campaigns.categories.${categoryKey(c)}`, c)}
              </option>
            ))}
          </select>
        </Field>

        <Field label={t('orgCampaigns.goalLabel')}>
          <input
            className={INPUT_CLASS}
            inputMode="decimal"
            name="goalMajor"
            value={f.goalMajor}
            onChange={(e) => set('goalMajor', e.target.value)}
            required
          />
        </Field>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t('orgCampaigns.stateLabel')}>
          <input
            className={INPUT_CLASS}
            name="state"
            value={f.state}
            onChange={(e) => set('state', e.target.value)}
          />
        </Field>
        <Field label={t('orgCampaigns.lgaLabel')}>
          <input
            className={INPUT_CLASS}
            name="lga"
            value={f.lga}
            onChange={(e) => set('lga', e.target.value)}
          />
        </Field>
      </div>

      <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200">
        <input
          type="checkbox"
          name="isEmergency"
          checked={f.isEmergency}
          onChange={(e) => set('isEmergency', e.target.checked)}
        />
        {t('orgCampaigns.emergencyLabel')}
      </label>
    </>
  );
}

/**
 * Edits a campaign's content.
 *
 * Only reachable while the campaign is DRAFT or NEEDS_CHANGES. The server
 * enforces the same window — content freezes once a campaign is in review or
 * live, so a donor is always giving to the thing they read.
 */

/**
 * The spend plan, editable alongside the campaign's content.
 *
 * Same window as the rest of the content: once a campaign is in review or
 * live, its milestones are part of what reviewers and donors were shown, so
 * they stop being editable here. Marking one COMPLETE is the exception and
 * lives on the campaign row, because that is progress reporting rather than
 * rewriting the plan.
 */
function SpendPlanEditor({ campaign, orgId }: { campaign: OrgCampaign; orgId: string }) {
  const { t } = useTranslation();
  const q = useMilestones(campaign.id);
  const add = useCreateMilestone(orgId);
  const remove = useDeleteMilestone(orgId, campaign.id);
  const [title, setTitle] = useState('');
  const [targetMajor, setTargetMajor] = useState('');
  const [error, setError] = useState('');

  const milestones = q.data ?? [];
  const allocated = milestones.reduce((sum, m) => sum + m.targetMinor, 0);
  const remaining = Math.max(0, campaign.goalMinor - allocated);
  // The server refuses a plan adding up to more than the goal. Catching it
  // here means the org sees what is left rather than a rejected form.
  const overAllocates = toMinor(targetMajor) > remaining;
  const canAdd =
    title.trim().length >= 3 && toMinor(targetMajor) > 0 && !overAllocates && !add.isPending;

  async function onAdd() {
    setError('');
    try {
      await add.mutateAsync({
        campaignId: campaign.id,
        title: title.trim(),
        targetMinor: toMinor(targetMajor),
      });
      setTitle('');
      setTargetMajor('');
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('orgCampaigns.planError'));
    }
  }

  return (
    <div className="space-y-2 rounded-lg border border-slate-200 p-3 dark:border-slate-700">
      <p className="text-sm font-medium text-slate-700 dark:text-slate-200">
        {t('orgCampaigns.planHeading')}
      </p>

      {milestones.length > 0 && (
        <ul className="space-y-1" data-testid="plan-list">
          {milestones.map((m) => (
            <li
              key={m.id}
              className="flex items-center justify-between gap-2 text-sm text-slate-700 dark:text-slate-200"
            >
              <span className="min-w-0 flex-1 truncate">{m.title}</span>
              <span className="tabular-nums">
                {formatMoney(m.targetMinor, campaign.currency, 'en-NG')}
              </span>
              <button
                type="button"
                className="text-xs text-red-600 hover:underline dark:text-red-400"
                onClick={() => remove.mutate(m.id)}
                aria-label={t('orgCampaigns.removeMilestone')}
              >
                {t('orgCampaigns.remove')}
              </button>
            </li>
          ))}
        </ul>
      )}

      {/* Allocation against the goal. Not enforced — an org may deliberately
          under-allocate while still planning — but shown, because a plan that
          does not add up to the ask is what a reviewer will bounce. */}
      <p className="text-xs text-slate-500 dark:text-slate-400">
        {t('orgCampaigns.allocated', {
          allocated: formatMoney(allocated, campaign.currency, 'en-NG'),
          goal: formatMoney(campaign.goalMinor, campaign.currency, 'en-NG'),
        })}{' '}
        {remaining > 0
          ? t('orgCampaigns.remaining', {
              remaining: formatMoney(remaining, campaign.currency, 'en-NG'),
            })
          : t('orgCampaigns.fullyAllocated')}
      </p>

      <div className="grid grid-cols-[1fr_auto_auto] gap-2">
        <input
          className={INPUT_CLASS}
          name="newMilestoneTitle"
          placeholder={t('orgCampaigns.milestoneLabel')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <input
          className={INPUT_CLASS}
          name="newMilestoneTarget"
          inputMode="decimal"
          placeholder={t('orgCampaigns.targetPlaceholder')}
          value={targetMajor}
          onChange={(e) => setTargetMajor(e.target.value)}
        />
        <Button type="button" size="sm" variant="secondary" disabled={!canAdd} onClick={onAdd}>
          {t('orgCampaigns.addMilestone')}
        </Button>
      </div>
      {/* Fixed-height slot. Showing and hiding this as the user types was
          changing the dialog's height on every keystroke, which is what made
          it jump. The space is always reserved; only the text changes. */}
      <p className="min-h-[1.25rem] text-xs" aria-live="polite">
        {overAllocates ? (
          <span className="text-amber-700 dark:text-amber-300">
            {t('orgCampaigns.exceedsGoal', {
              remaining: formatMoney(remaining, campaign.currency, 'en-NG'),
            })}
          </span>
        ) : error ? (
          <span className="text-red-600 dark:text-red-400">{error}</span>
        ) : null}
      </p>
    </div>
  );
}

function EditCampaignModal({
  campaign,
  orgId,
  onClose,
}: {
  campaign: OrgCampaign;
  orgId: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const update = useUpdateCampaign(orgId);
  const [f, setF] = useState<CampaignFields>({
    title: campaign.title,
    summary: campaign.summary,
    description: campaign.description ?? '',
    category: campaign.category,
    // Back to major units for display; the goal is stored in kobo.
    goalMajor: String(campaign.goalMinor / 100),
    state: campaign.state ?? '',
    lga: campaign.lga ?? '',
    isEmergency: campaign.isEmergency,
  });
  const [error, setError] = useState('');
  const set = <K extends keyof CampaignFields>(k: K, v: CampaignFields[K]) =>
    setF((prev) => ({ ...prev, [k]: v }));

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await update.mutateAsync({ campaignId: campaign.id, input: fieldsToInput(f) });
      onClose();
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('orgCampaigns.updateError'));
    }
  }

  return (
    <Modal title={t('orgCampaigns.editCampaign')} onClose={onClose} size="xl">
      <form className="space-y-3" onSubmit={onSubmit}>
        <CampaignFieldSet f={f} set={set} />

        <SpendPlanEditor campaign={campaign} orgId={orgId} />

        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" size="sm" disabled={!fieldsValid(f) || update.isPending}>
            {update.isPending ? t('orgCampaigns.saving') : t('orgCampaigns.saveChanges')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

/**
 * Marking milestones complete on a live campaign.
 *
 * The only change to a spend plan permitted after review, because it is
 * progress reporting rather than rewriting what donors were shown. Each
 * completion notifies everyone who funded the campaign.
 */
function LivePlanProgress({ campaign, orgId }: { campaign: OrgCampaign; orgId: string }) {
  const { t } = useTranslation();
  const q = useMilestones(campaign.id);
  const complete = useCompleteMilestone(orgId, campaign.id);
  const open = (q.data ?? []).filter((m) => m.status !== 'COMPLETED');

  if ((q.data ?? []).length === 0) return null;

  return (
    <div className="mt-2 border-t border-slate-200 pt-2 dark:border-slate-700">
      <p className="mb-1 text-xs font-medium text-slate-600 dark:text-slate-300">
        {t('orgCampaigns.progressHeading')}
      </p>
      {open.length === 0 ? (
        <p className="text-xs text-slate-600 dark:text-slate-300">
          {t('orgCampaigns.allComplete')}
        </p>
      ) : (
        <ul className="space-y-1">
          {open.map((m) => (
            <li key={m.id} className="flex items-center justify-between gap-2 text-xs">
              <span className="min-w-0 flex-1 truncate text-slate-700 dark:text-slate-200">
                {m.title}
              </span>
              <button
                type="button"
                className="font-semibold text-civic-700 hover:underline dark:text-civic-200"
                onClick={() => complete.mutate(m.id)}
                disabled={complete.isPending}
              >
                {t('orgCampaigns.markComplete')}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/**
 * Filing the closing account on a completed campaign.
 *
 * Offered only in COMPLETED, which is the one state where it applies. The
 * shortfall is shown before filing rather than after, so an organization
 * knows exactly what it is publishing alongside its report.
 */
function FinalReportForm({ campaign, orgId }: { campaign: OrgCampaign; orgId: string }) {
  const { t } = useTranslation();
  const file = useFileReport(orgId);
  const [open, setOpen] = useState(false);
  const [body, setBody] = useState('');
  const [attachments, setAttachments] = useState('');
  const [error, setError] = useState('');

  // Sum the published spend rather than trusting a field on the campaign:
  // the org-scoped campaign list returns the stored model, which carries no
  // spend total, so anything read from there is silently zero.
  const spend = useCampaignSpend(campaign.id);
  const reported = (spend.data ?? []).reduce((sum, r) => sum + r.amountMinor, 0);
  const unaccounted = Math.max(0, campaign.raisedMinor - reported);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await file.mutateAsync({
        campaignId: campaign.id,
        input: {
          body: body.trim(),
          attachmentUrls: attachments
            .split('\n')
            .map((x) => x.trim())
            .filter(Boolean),
        },
      });
      setOpen(false);
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('orgCampaigns.reportError'));
    }
  }

  if (!open) {
    return (
      <Button size="sm" onClick={() => setOpen(true)}>
        {t('orgCampaigns.fileReport')}
      </Button>
    );
  }

  return (
    <Modal title={t('orgCampaigns.fileReport')} onClose={() => setOpen(false)} size="xl">
      <form className="space-y-3" onSubmit={onSubmit}>
        <CompletionReportAssist campaignId={campaign.id} onApply={setBody} />

        <Field label={t('orgCampaigns.reportBodyLabel')} hint={t('orgCampaigns.reportBodyHint')}>
          <textarea
            className={INPUT_CLASS}
            rows={6}
            name="reportBody"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            required
            minLength={40}
          />
        </Field>

        <Field label={t('orgCampaigns.reportFilesLabel')} hint={t('orgCampaigns.attachmentsHint')}>
          <textarea
            className={INPUT_CLASS}
            rows={2}
            name="reportFiles"
            value={attachments}
            onChange={(e) => setAttachments(e.target.value)}
            placeholder="https://"
          />
        </Field>

        {/* Said before they file, not after. Filing with money unexplained is
            allowed — but it is published alongside the report. */}
        {unaccounted > 0 && (
          <p className="rounded border-l-4 border-amber-500 bg-amber-50 px-3 py-2 text-xs text-slate-700 dark:bg-amber-900/20 dark:text-slate-200">
            {t('orgCampaigns.reportShortfall', {
              amount: formatMoney(unaccounted, campaign.currency, 'en-NG'),
            })}
          </p>
        )}

        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={() => setOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" size="sm" disabled={body.trim().length < 40 || file.isPending}>
            {file.isPending ? t('orgCampaigns.filing') : t('orgCampaigns.publishReport')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

export function NewCampaignButton({ orgId }: { orgId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button size="sm" variant="secondary" onClick={() => setOpen(true)}>
        {t('orgCampaigns.newCampaign')}
      </Button>
      {open && <NewCampaignModal orgId={orgId} onClose={() => setOpen(false)} />}
    </>
  );
}

/**
 * Creates a campaign and its first milestone in one pass.
 *
 * The milestone is part of the same form on purpose. A campaign with a goal
 * and no spend plan is exactly the vague ask the whole feature exists to
 * prevent, and review would bounce it anyway — asking for the first line of
 * the plan up front is cheaper than a rejection round-trip.
 */
function NewCampaignModal({ orgId, onClose }: { orgId: string; onClose: () => void }) {
  const { t } = useTranslation();
  const create = useCreateCampaign(orgId);
  const addMilestone = useCreateMilestone(orgId);

  const [f, setF] = useState<CampaignFields>({
    title: '',
    summary: '',
    description: '',
    category: 'EMERGENCY_RELIEF',
    goalMajor: '',
    state: '',
    lga: '',
    isEmergency: false,
  });
  const [milestoneTitle, setMilestoneTitle] = useState('');
  const [milestoneTarget, setMilestoneTarget] = useState('');
  const [error, setError] = useState('');
  const set = <K extends keyof CampaignFields>(k: K, v: CampaignFields[K]) =>
    setF((prev) => ({ ...prev, [k]: v }));

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      const campaign = await create.mutateAsync(fieldsToInput(f));
      // Best-effort: if the milestone fails, the campaign still exists as a
      // draft and the plan can be added before submitting for review.
      if (campaign?.id && milestoneTitle.trim()) {
        await addMilestone
          .mutateAsync({
            campaignId: campaign.id,
            title: milestoneTitle.trim(),
            // The amount they entered, NOT the whole goal. Defaulting to the
            // full goal would leave no room to add a second milestone
            // without first shrinking this one.
            targetMinor: toMinor(milestoneTarget) || toMinor(f.goalMajor),
          })
          .catch(() => undefined);
      }
      onClose();
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('orgCampaigns.createError'));
    }
  }

  return (
    <Modal title={t('orgCampaigns.newCampaign')} onClose={onClose} size="xl">
      <form className="space-y-3" onSubmit={onSubmit}>
        <CampaignFieldSet f={f} set={set} showAssist />

        {/* Asked for at creation on purpose: a campaign with a goal and no
            spend plan is the vague ask this feature exists to prevent, and
            review would bounce it anyway. */}
        <Field label={t('orgCampaigns.milestoneLabel')} hint={t('orgCampaigns.milestoneHint')}>
          <div className="grid grid-cols-[1fr_auto] gap-2">
            <input
              className={INPUT_CLASS}
              name="milestoneTitle"
              value={milestoneTitle}
              onChange={(e) => setMilestoneTitle(e.target.value)}
            />
            <input
              className={INPUT_CLASS}
              name="milestoneTarget"
              inputMode="decimal"
              placeholder={t('orgCampaigns.targetPlaceholder')}
              value={milestoneTarget}
              onChange={(e) => setMilestoneTarget(e.target.value)}
            />
          </div>
        </Field>

        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        <p className="text-xs text-slate-600 dark:text-slate-300">{t('orgCampaigns.reviewNote')}</p>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" size="sm" disabled={!fieldsValid(f) || create.isPending}>
            {create.isPending ? t('orgCampaigns.creating') : t('orgCampaigns.createDraft')}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block space-y-1">
      <span className="text-sm font-medium text-slate-700 dark:text-slate-200">{label}</span>
      {children}
      {hint && <span className="block text-xs text-slate-500 dark:text-slate-400">{hint}</span>}
    </label>
  );
}
