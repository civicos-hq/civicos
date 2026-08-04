import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { PageHeader } from '../components/PageHeader';
import { CampaignDraftAssist } from '../components/civicai/CampaignDraftAssist';
import { getApiError } from '../lib/api';
import {
  categoryKey,
  useCreateCampaign,
  useCreateMilestone,
  type CampaignCategory,
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

const FIELD =
  'mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500';

/** Major → minor once, at the boundary. Everything downstream stays integer. */
function toMinor(major: string): number {
  const n = Number.parseFloat(major.replace(/,/g, ''));
  return Number.isFinite(n) && n > 0 ? Math.round(n * 100) : 0;
}

/**
 * Creating a campaign, on its own page.
 *
 * Matches how consultations, announcements and projects work — a full page
 * at `/org/:orgId/<section>/new` rather than a modal. A campaign asks for
 * more than any of them (content, a goal, and the first line of a spend
 * plan), so it was the worst fit for a dialog.
 */
export function OrgCampaignCreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { orgId } = useParams<{ orgId: string }>();
  const createMutation = useCreateCampaign(orgId);
  const addMilestone = useCreateMilestone(orgId);

  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState<CampaignCategory>('EMERGENCY_RELIEF');
  const [goalNaira, setGoalNaira] = useState('');
  const [state, setState] = useState('');
  const [lga, setLga] = useState('');
  const [isEmergency, setIsEmergency] = useState(false);
  const [milestoneTitle, setMilestoneTitle] = useState('');
  const [milestoneNaira, setMilestoneNaira] = useState('');
  const [error, setError] = useState('');

  const goalMinor = toMinor(goalNaira);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      const created = await createMutation.mutateAsync({
        title: title.trim(),
        summary: summary.trim(),
        description: description.trim(),
        category,
        goalMinor,
        state: state.trim() || undefined,
        lga: lga.trim() || undefined,
        isEmergency,
      });
      // Best-effort: if the milestone fails the campaign still exists as a
      // draft, and the plan can be completed before submitting for review.
      if (created?.id && milestoneTitle.trim()) {
        await addMilestone
          .mutateAsync({
            campaignId: created.id,
            title: milestoneTitle.trim(),
            targetMinor: toMinor(milestoneNaira) || goalMinor,
          })
          .catch(() => undefined);
      }
      navigate(`/org/${orgId}?tab=campaigns`);
    } catch (err) {
      const apiErr = getApiError(err);
      setError(apiErr?.message ?? t('orgCampaigns.createError'));
    }
  }

  return (
    <section className="space-y-6">
      <Link
        to={`/org/${orgId}?tab=campaigns`}
        className="text-sm font-semibold text-civic-700 hover:underline dark:text-civic-200"
      >
        {t('orgCampaignCreate.back')}
      </Link>

      <PageHeader
        eyebrow={t('orgCampaignCreate.eyebrow')}
        title={t('orgCampaignCreate.title')}
        subtitle={t('orgCampaignCreate.subtitle')}
      />

      <form onSubmit={onSubmit} className="max-w-3xl space-y-4">
        {/* Above the fields, because it is a way of filling them in — but
            nothing lands in the form until the organization clicks Use. */}
        <CampaignDraftAssist
          goalMinor={goalMinor}
          currency="NGN"
          state={state}
          lga={lga}
          isEmergency={isEmergency}
          onApply={(d) => {
            setTitle(d.title);
            setSummary(d.summary);
            setDescription(d.description);
          }}
        />

        <div>
          <label
            htmlFor="campaign-title"
            className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
          >
            {t('orgCampaigns.titleLabel')}
          </label>
          <input
            id="campaign-title"
            name="title"
            className={FIELD}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            minLength={4}
            maxLength={160}
          />
        </div>

        <div>
          <label
            htmlFor="campaign-summary"
            className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
          >
            {t('orgCampaigns.summaryLabel')}
          </label>
          <input
            id="campaign-summary"
            name="summary"
            className={FIELD}
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            required
            minLength={10}
            maxLength={300}
          />
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            {t('orgCampaigns.summaryHint')}
          </p>
        </div>

        <div>
          <label
            htmlFor="campaign-description"
            className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
          >
            {t('orgCampaigns.descriptionLabel')}
          </label>
          <textarea
            id="campaign-description"
            name="description"
            className={FIELD}
            rows={6}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            required
            minLength={40}
          />
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            {t('orgCampaigns.descriptionHint')}
          </p>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label
              htmlFor="campaign-category"
              className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
            >
              {t('orgCampaigns.categoryLabel')}
            </label>
            <select
              id="campaign-category"
              name="category"
              className={FIELD}
              value={category}
              onChange={(e) => setCategory(e.target.value as CampaignCategory)}
            >
              {CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {t(`campaigns.categories.${categoryKey(c)}`, c)}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label
              htmlFor="campaign-goal"
              className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
            >
              {t('orgCampaigns.goalLabel')}
            </label>
            <input
              id="campaign-goal"
              name="goalMajor"
              className={FIELD}
              inputMode="decimal"
              value={goalNaira}
              onChange={(e) => setGoalNaira(e.target.value)}
              required
            />
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label
              htmlFor="campaign-state"
              className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
            >
              {t('orgCampaigns.stateLabel')}
            </label>
            <input
              id="campaign-state"
              name="state"
              className={FIELD}
              value={state}
              onChange={(e) => setState(e.target.value)}
            />
          </div>
          <div>
            <label
              htmlFor="campaign-lga"
              className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
            >
              {t('orgCampaigns.lgaLabel')}
            </label>
            <input
              id="campaign-lga"
              name="lga"
              className={FIELD}
              value={lga}
              onChange={(e) => setLga(e.target.value)}
            />
          </div>
        </div>

        {/* Asked for at creation deliberately: a campaign with a goal and no
            spend plan is the vague ask this feature exists to prevent, and
            review would send it back anyway. */}
        <fieldset className="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
          <legend className="px-1 text-sm font-semibold text-slate-700 dark:text-slate-300">
            {t('orgCampaigns.planHeading')}
          </legend>
          <div className="grid gap-4 md:grid-cols-[1fr_auto]">
            <div>
              <label
                htmlFor="campaign-milestone"
                className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
              >
                {t('orgCampaigns.milestoneLabel')}
              </label>
              <input
                id="campaign-milestone"
                name="milestoneTitle"
                className={FIELD}
                value={milestoneTitle}
                onChange={(e) => setMilestoneTitle(e.target.value)}
              />
            </div>
            <div>
              <label
                htmlFor="campaign-milestone-amount"
                className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
              >
                {t('orgCampaigns.targetPlaceholder')}
              </label>
              <input
                id="campaign-milestone-amount"
                name="milestoneTarget"
                className={FIELD}
                inputMode="decimal"
                value={milestoneNaira}
                onChange={(e) => setMilestoneNaira(e.target.value)}
              />
            </div>
          </div>
          <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
            {t('orgCampaigns.milestoneHint')}
          </p>
        </fieldset>

        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
          <input
            type="checkbox"
            name="isEmergency"
            checked={isEmergency}
            onChange={(e) => setIsEmergency(e.target.checked)}
          />
          {t('orgCampaigns.emergencyLabel')}
        </label>

        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        <p className="text-xs text-slate-500 dark:text-slate-400">{t('orgCampaigns.reviewNote')}</p>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? t('orgCampaigns.creating') : t('orgCampaigns.createDraft')}
          </Button>
          <Link
            to={`/org/${orgId}?tab=campaigns`}
            className="text-sm font-semibold text-slate-600 hover:underline dark:text-slate-300"
          >
            {t('common.cancel')}
          </Link>
        </div>
      </form>
    </section>
  );
}
