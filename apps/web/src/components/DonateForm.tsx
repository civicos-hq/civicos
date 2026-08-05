import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Loader2 } from 'lucide-react';
import {
  formatMoney,
  formatMoneyExact,
  previewSplit,
  useCreateDonationIntent,
  type PublicCampaignDetail,
} from '../hooks/useCampaigns';

// Preset amounts in MAJOR units, converted to minor at the boundary. Chosen
// to be plausible for a Nigerian civic donation rather than round dollars.
const PRESETS_MAJOR = [1_000, 2_500, 5_000, 10_000];

/**
 * The donate flow.
 *
 * CivicOS never touches card details: this form only opens a transaction and
 * hands the donor to Paystack's hosted checkout. Nothing sensitive is
 * collected here, which is the whole reason the flow is shaped this way.
 */
export function DonateForm({ campaign }: { campaign: PublicCampaignDetail }) {
  const { t, i18n } = useTranslation();
  const [amountMajor, setAmountMajor] = useState<string>(String(PRESETS_MAJOR[1]));
  const [email, setEmail] = useState('');
  const [donorName, setDonorName] = useState('');
  const [isAnonymous, setIsAnonymous] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState<string | null>(null);

  const intent = useCreateDonationIntent(campaign.id);

  // Parse to an integer number of minor units. The multiply happens once,
  // here at the boundary — everything downstream stays integer.
  const parsedMajor = Number.parseFloat(amountMajor.replace(/,/g, ''));
  const amountMinor =
    Number.isFinite(parsedMajor) && parsedMajor > 0 ? Math.round(parsedMajor * 100) : 0;
  const split = previewSplit(amountMinor, campaign.platformFeeBps);
  const canSubmit = amountMinor > 0 && email.trim().length > 3 && !intent.isPending;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const res = await intent.mutateAsync({
        amountMinor,
        email: email.trim(),
        donorName: donorName.trim() || undefined,
        isAnonymous,
        message: message.trim() || undefined,
      });
      // Hand off to Paystack. A full navigation rather than a new tab —
      // popup blockers eat the tab and the donor thinks nothing happened.
      window.location.href = res.authorizationUrl;
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('campaigns.donate.genericError'));
    }
  }

  return (
    <form className="fund-donate" onSubmit={onSubmit}>
      <h2 className="fund-donate-heading">{t('campaigns.donate.heading')}</h2>

      <fieldset className="fund-donate-presets">
        <legend className="sr-only">{t('campaigns.donate.amountLabel')}</legend>
        {PRESETS_MAJOR.map((p) => (
          <button
            key={p}
            type="button"
            className={`fund-preset${Number(amountMajor) === p ? ' is-active' : ''}`}
            aria-pressed={Number(amountMajor) === p}
            onClick={() => setAmountMajor(String(p))}
          >
            {formatMoney(p * 100, campaign.currency, i18n.language)}
          </button>
        ))}
      </fieldset>

      <label className="fund-field">
        <span>{t('campaigns.donate.amountLabel')}</span>
        <input
          type="text"
          inputMode="decimal"
          value={amountMajor}
          onChange={(e) => setAmountMajor(e.target.value)}
          aria-describedby="fund-split"
          required
        />
      </label>

      {/* The fee disclosure. Shown live against the amount being typed,
          before the donor commits — not in a footnote on the receipt. */}
      <p id="fund-split" className="fund-split">
        {campaign.platformFeeBps > 0 ? (
          <>
            <strong>
              {formatMoneyExact(split.organizationMinor, campaign.currency, i18n.language)}
            </strong>{' '}
            {t('campaigns.donate.reachesOrg', { org: campaign.organizationName ?? '' })}
            <br />
            {/* Both deductions, itemised. The form used to show only the
                CivicOS fee, which overstated what the organization received
                on every single donation. */}
            <span className="fund-split-fee">
              {t('campaigns.donate.platformFee', {
                fee: formatMoneyExact(split.platformFeeMinor, campaign.currency, i18n.language),
                percent: (campaign.platformFeeBps / 100).toString(),
              })}
              {' · '}
              {t('campaigns.donate.pspFee', {
                fee: formatMoneyExact(split.pspFeeMinor, campaign.currency, i18n.language),
              })}
            </span>
          </>
        ) : (
          <strong>{t('campaigns.donate.noFee')}</strong>
        )}
      </p>

      <label className="fund-field">
        <span>{t('campaigns.donate.emailLabel')}</span>
        <input
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <small>{t('campaigns.donate.emailHelp')}</small>
      </label>

      <label className="fund-field">
        <span>{t('campaigns.donate.nameLabel')}</span>
        <input
          type="text"
          autoComplete="name"
          value={donorName}
          onChange={(e) => setDonorName(e.target.value)}
          disabled={isAnonymous}
        />
      </label>

      <label className="fund-check">
        <input
          type="checkbox"
          checked={isAnonymous}
          onChange={(e) => setIsAnonymous(e.target.checked)}
        />
        <span>{t('campaigns.donate.anonymous')}</span>
      </label>

      <label className="fund-field">
        <span>{t('campaigns.donate.messageLabel')}</span>
        <textarea rows={2} value={message} onChange={(e) => setMessage(e.target.value)} />
      </label>

      {error && (
        <p className="fund-donate-error" role="alert">
          {error}
        </p>
      )}

      <button
        type="submit"
        className="home-btn home-btn-primary fund-donate-submit"
        disabled={!canSubmit}
      >
        {intent.isPending ? (
          <>
            <Loader2 className="h-4 w-4 fund-spin" aria-hidden="true" />
            {t('campaigns.donate.submitting')}
          </>
        ) : (
          t('campaigns.donate.submit', {
            amount: formatMoney(amountMinor, campaign.currency, i18n.language),
          })
        )}
      </button>

      <p className="fund-donate-note">{t('campaigns.donate.paystackNote')}</p>
    </form>
  );
}
