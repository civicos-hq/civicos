import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { CampaignImpactSummary } from './civicai/CampaignImpactSummary';
import { DonorUpdateAssist } from './civicai/UpdateDraftAssist';
import { Loader2, Trash2 } from 'lucide-react';
import {
  formatMoneyExact,
  useCreateFundingUpdate,
  useCreateSpend,
  useDeleteSpend,
  type PublicCampaignDetail,
  type SpendRecord,
} from '../hooks/useCampaigns';

/**
 * The organization's console, rendered inline on its own campaign page.
 *
 * Deliberately here rather than in a separate admin area: an organization
 * publishing an account of its spending should be looking at the same page
 * its donors read, with the same figures in front of it. Filing a report
 * into a form somewhere else makes it easy to forget who it is addressed to.
 *
 * Visibility is gated on org membership, but that is a rendering decision
 * only — every write is authorised again server-side against the owning
 * organization.
 */
export function CampaignConsole({
  campaign,
  spend,
  locale,
}: {
  campaign: PublicCampaignDetail;
  spend: SpendRecord[];
  locale: string;
}) {
  const { t } = useTranslation();
  return (
    <section className="fund-console" aria-label={t('campaigns.console.heading')}>
      <h2 className="fund-console-heading">{t('campaigns.console.heading')}</h2>
      <p className="fund-console-lede">{t('campaigns.console.lede')}</p>
      <SpendForm campaign={campaign} />
      <PublishedSpend campaign={campaign} spend={spend} locale={locale} />
      {/* Sits after the published spend and before the update form on
          purpose: it reads what has been published, and what it finds
          missing is usually what the next update should say. */}
      <CampaignImpactSummary campaignId={campaign.id} />
      <UpdateForm campaign={campaign} />
    </section>
  );
}

function SpendForm({ campaign }: { campaign: PublicCampaignDetail }) {
  const { t } = useTranslation();
  const [milestoneId, setMilestoneId] = useState(campaign.milestones[0]?.id ?? '');
  const [amountMajor, setAmountMajor] = useState('');
  const [description, setDescription] = useState('');
  const [spentAt, setSpentAt] = useState('');
  const [receiptUrl, setReceiptUrl] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const create = useCreateSpend(campaign.id);

  // Parsed once, here at the boundary; everything downstream stays integer.
  const parsed = Number.parseFloat(amountMajor.replace(/,/g, ''));
  const amountMinor = Number.isFinite(parsed) && parsed > 0 ? Math.round(parsed * 100) : 0;
  // Today, in the input's own format. The server refuses future dates, so
  // the picker should not offer them either.
  const today = new Date().toISOString().slice(0, 10);
  const canSubmit =
    !!milestoneId &&
    amountMinor > 0 &&
    description.trim().length >= 3 &&
    !!spentAt &&
    !create.isPending;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setDone(false);
    try {
      await create.mutateAsync({
        milestoneId,
        amountMinor,
        description: description.trim(),
        spentAt,
        receiptUrl: receiptUrl.trim() || undefined,
      });
      setAmountMajor('');
      setDescription('');
      setSpentAt('');
      setReceiptUrl('');
      setDone(true);
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('campaigns.console.genericError'));
    }
  }

  if (campaign.milestones.length === 0) {
    return <p className="fund-console-empty">{t('campaigns.console.noMilestones')}</p>;
  }

  return (
    <form className="fund-console-form" onSubmit={onSubmit}>
      <h3 className="fund-console-sub">{t('campaigns.console.spendHeading')}</h3>

      <label className="fund-field">
        <span>{t('campaigns.console.milestoneLabel')}</span>
        <select value={milestoneId} onChange={(e) => setMilestoneId(e.target.value)} required>
          {campaign.milestones.map((m) => (
            <option key={m.id} value={m.id}>
              {m.title}
            </option>
          ))}
        </select>
      </label>

      <label className="fund-field">
        <span>{t('campaigns.console.amountLabel')}</span>
        <input
          type="text"
          inputMode="decimal"
          value={amountMajor}
          onChange={(e) => setAmountMajor(e.target.value)}
          required
        />
      </label>

      <label className="fund-field">
        <span>{t('campaigns.console.descriptionLabel')}</span>
        <textarea
          rows={2}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
        />
        <small>{t('campaigns.console.descriptionHelp')}</small>
      </label>

      <label className="fund-field">
        <span>{t('campaigns.console.spentAtLabel')}</span>
        <input
          type="date"
          value={spentAt}
          max={today}
          onChange={(e) => setSpentAt(e.target.value)}
          required
        />
        <small>{t('campaigns.console.spentAtHelp')}</small>
      </label>

      <label className="fund-field">
        <span>{t('campaigns.console.receiptLabel')}</span>
        <input
          type="url"
          value={receiptUrl}
          onChange={(e) => setReceiptUrl(e.target.value)}
          placeholder="https://"
        />
      </label>

      {error && (
        <p className="fund-donate-error" role="alert">
          {error}
        </p>
      )}
      {done && (
        <p className="fund-console-ok" role="status">
          {t('campaigns.console.spendPublished')}
        </p>
      )}

      <button
        type="submit"
        className="home-btn home-btn-primary fund-console-submit"
        disabled={!canSubmit}
      >
        {create.isPending ? (
          <>
            <Loader2 className="h-4 w-4 fund-spin" aria-hidden="true" />
            {t('campaigns.console.publishing')}
          </>
        ) : (
          t('campaigns.console.publishSpend')
        )}
      </button>

      {/* Said before they publish, not after. */}
      <p className="fund-console-note">{t('campaigns.console.spendNote')}</p>
    </form>
  );
}

/** Already-published records, with the ability to withdraw one. */
function PublishedSpend({
  campaign,
  spend,
  locale,
}: {
  campaign: PublicCampaignDetail;
  spend: SpendRecord[];
  locale: string;
}) {
  const { t } = useTranslation();
  const del = useDeleteSpend(campaign.id);
  const [confirming, setConfirming] = useState<string | null>(null);

  if (spend.length === 0) return null;

  return (
    <div className="fund-console-published">
      <h3 className="fund-console-sub">{t('campaigns.console.publishedHeading')}</h3>
      <ul className="fund-console-list">
        {spend.map((r) => (
          <li key={r.id} className="fund-console-item">
            <span className="fund-console-item-amount">
              {formatMoneyExact(r.amountMinor, r.currency, locale)}
            </span>
            <span className="fund-console-item-desc">{r.description}</span>
            {confirming === r.id ? (
              <span className="fund-console-confirm">
                <button
                  type="button"
                  className="fund-console-danger"
                  onClick={() => del.mutate(r.id, { onSettled: () => setConfirming(null) })}
                >
                  {t('campaigns.console.confirmWithdraw')}
                </button>
                <button type="button" onClick={() => setConfirming(null)}>
                  {t('common.cancel')}
                </button>
              </span>
            ) : (
              <button
                type="button"
                className="fund-console-icon"
                onClick={() => setConfirming(r.id)}
                aria-label={t('campaigns.console.withdraw')}
              >
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </button>
            )}
          </li>
        ))}
      </ul>
      {/* Withdrawing is audited server-side; say so rather than let it feel
          like a silent delete. */}
      <p className="fund-console-note">{t('campaigns.console.withdrawNote')}</p>
    </div>
  );
}

function UpdateForm({ campaign }: { campaign: PublicCampaignDetail }) {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [attachments, setAttachments] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const create = useCreateFundingUpdate(campaign.organizationId, campaign.id);
  const canSubmit = body.trim().length >= 2 && !create.isPending;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setDone(false);
    try {
      await create.mutateAsync({
        campaignId: campaign.id,
        title: title.trim() || undefined,
        body: body.trim(),
        attachmentUrls: attachments
          .split('\n')
          .map((s) => s.trim())
          .filter(Boolean),
      });
      setTitle('');
      setBody('');
      setAttachments('');
      setDone(true);
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('campaigns.console.genericError'));
    }
  }

  return (
    <form className="fund-console-form" onSubmit={onSubmit}>
      <h3 className="fund-console-sub">{t('campaigns.console.updateHeading')}</h3>

      <label className="fund-field">
        <span>{t('campaigns.console.titleLabel')}</span>
        <input
          type="text"
          maxLength={200}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </label>

      <label className="fund-field">
        <span>{t('campaigns.console.bodyLabel')}</span>
        <textarea rows={3} value={body} onChange={(e) => setBody(e.target.value)} required />
      </label>

      {/* Fills the fields above on Use; the org still has to submit. */}
      <DonorUpdateAssist
        campaignId={campaign.id}
        onApply={(d) => {
          setTitle(d.title);
          setBody(d.body);
        }}
      />

      <label className="fund-field">
        <span>{t('campaigns.console.attachmentsLabel')}</span>
        <textarea
          rows={2}
          value={attachments}
          onChange={(e) => setAttachments(e.target.value)}
          placeholder="https://"
        />
        <small>{t('campaigns.console.attachmentsHelp')}</small>
      </label>

      {error && (
        <p className="fund-donate-error" role="alert">
          {error}
        </p>
      )}
      {done && (
        <p className="fund-console-ok" role="status">
          {t('campaigns.console.updatePublished')}
        </p>
      )}

      <button
        type="submit"
        className="home-btn home-btn-primary fund-console-submit"
        disabled={!canSubmit}
      >
        {create.isPending ? (
          <>
            <Loader2 className="h-4 w-4 fund-spin" aria-hidden="true" />
            {t('campaigns.console.publishing')}
          </>
        ) : (
          t('campaigns.console.publishUpdate')
        )}
      </button>
      <p className="fund-console-note">{t('campaigns.console.updateNote')}</p>
    </form>
  );
}
