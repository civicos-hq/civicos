import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { CheckCircle2, Landmark } from 'lucide-react';
import type { Organization } from '@civicos/types';
import { useBanks, useConnectPayout, useFundingEligibility } from '../hooks/useCampaigns';

const FIELD =
  'mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm shadow-sm focus:border-civic-500 focus:outline-none focus:ring-1 focus:ring-civic-500';

/**
 * Connecting where donations are paid out to.
 *
 * This is organization-level, not per-campaign: an organization has one bank
 * account, and every campaign it runs settles there. Until it is connected
 * the organization cannot take a single donation — the donate endpoint
 * refuses with ORG_NOT_FUNDING_ELIGIBLE — so this is the first thing an
 * organization needs after being verified.
 *
 * The account number is sent to Paystack once and is never stored by
 * CivicOS. What comes back is a sub-account code, and only that plus the
 * bank name and last four digits are kept, which is why a connected account
 * can be shown but never re-displayed in full.
 */
export function PayoutAccount({ org }: { org: Organization }) {
  const { t } = useTranslation();
  const connected = !!org.pspSubaccountCode;
  const [editing, setEditing] = useState(false);
  const eligibility = useFundingEligibility(org.id);

  return (
    <section className="space-y-3">
      <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
        <Landmark
          className="mr-2 inline h-4 w-4 text-civic-700 dark:text-civic-200"
          aria-hidden="true"
        />
        {t('payout.heading')}
      </h2>

      {connected ? (
        <div className="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
          <p className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
            <CheckCircle2 className="h-4 w-4 text-green-600" aria-hidden="true" />
            {t('payout.connected', {
              bank: org.pspBankName ?? '',
              last4: org.pspAccountLast4 ?? '••••',
            })}
          </p>
          <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
            {t('payout.connectedNote')}
          </p>
          {!editing && (
            <button
              type="button"
              className="mt-2 text-xs font-semibold text-civic-700 hover:underline dark:text-civic-200"
              onClick={() => setEditing(true)}
            >
              {t('payout.change')}
            </button>
          )}
        </div>
      ) : (
        <p className="rounded-lg border-l-4 border-amber-500 bg-amber-50 px-3 py-2 text-sm text-slate-700 dark:bg-amber-900/20 dark:text-slate-200">
          {t('payout.notConnected')}
        </p>
      )}

      {/* What else stands between this organization and taking donations.
          Read from the server so the rule lives in exactly one place. */}
      {eligibility.data && !eligibility.data.eligible && eligibility.data.missing.length > 0 && (
        <div className="rounded-lg border border-slate-200 p-3 text-sm dark:border-slate-700">
          <p className="font-semibold text-slate-800 dark:text-slate-200">
            {t('payout.outstandingHeading')}
          </p>
          <ul className="mt-1 list-inside list-disc text-slate-600 dark:text-slate-300">
            {eligibility.data.missing.map((m) => (
              <li key={m}>{m}</li>
            ))}
          </ul>
        </div>
      )}

      {(!connected || editing) && <PayoutForm org={org} onDone={() => setEditing(false)} />}
    </section>
  );
}

function PayoutForm({ org, onDone }: { org: Organization; onDone: () => void }) {
  const { t } = useTranslation();
  const banksQuery = useBanks(org.id);
  const connect = useConnectPayout(org.id);

  const [bankCode, setBankCode] = useState('');
  const [accountNumber, setAccountNumber] = useState('');
  const [businessName, setBusinessName] = useState(org.name ?? '');
  const [contactEmail, setContactEmail] = useState('');
  const [error, setError] = useState('');

  const banks = banksQuery.data ?? [];
  // Nigerian account numbers are 10 digits; the server accepts 6-20 for
  // other rails, so this only stops the obvious typo before it reaches
  // Paystack and comes back as a rejection.
  const digitsOnly = accountNumber.replace(/\D/g, '');
  const canSubmit =
    !!bankCode && digitsOnly.length >= 10 && businessName.trim().length >= 2 && !connect.isPending;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    try {
      await connect.mutateAsync({
        bankCode,
        accountNumber: digitsOnly,
        businessName: businessName.trim(),
        contactEmail: contactEmail.trim() || undefined,
      });
      // Deliberately cleared: the number is not ours to keep on screen.
      setAccountNumber('');
      onDone();
    } catch (err) {
      const res = (err as { response?: { data?: { message?: string } } }).response;
      setError(res?.data?.message ?? t('payout.error'));
    }
  }

  return (
    <form
      className="max-w-xl space-y-3 rounded-lg border border-slate-200 p-4 dark:border-slate-700"
      onSubmit={onSubmit}
    >
      <div>
        <label
          htmlFor="payout-bank"
          className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
        >
          {t('payout.bankLabel')}
        </label>
        <select
          id="payout-bank"
          name="bankCode"
          className={FIELD}
          value={bankCode}
          onChange={(e) => setBankCode(e.target.value)}
          disabled={banksQuery.isLoading || banksQuery.isError}
          required
        >
          <option value="">
            {banksQuery.isLoading ? t('common.loading') : t('payout.bankPlaceholder')}
          </option>
          {banks.map((b) => (
            <option key={b.code} value={b.code}>
              {b.name}
            </option>
          ))}
        </select>
        {banksQuery.isError && (
          <p className="mt-1 text-xs text-red-600 dark:text-red-400">{t('payout.banksError')}</p>
        )}
      </div>

      <div>
        <label
          htmlFor="payout-account"
          className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
        >
          {t('payout.accountLabel')}
        </label>
        <input
          id="payout-account"
          name="accountNumber"
          className={FIELD}
          inputMode="numeric"
          autoComplete="off"
          value={accountNumber}
          onChange={(e) => setAccountNumber(e.target.value)}
          maxLength={20}
          required
        />
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t('payout.accountHelp')}</p>
      </div>

      <div>
        <label
          htmlFor="payout-business"
          className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
        >
          {t('payout.businessLabel')}
        </label>
        <input
          id="payout-business"
          name="businessName"
          className={FIELD}
          value={businessName}
          onChange={(e) => setBusinessName(e.target.value)}
          required
        />
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          {t('payout.businessHelp')}
        </p>
      </div>

      <div>
        <label
          htmlFor="payout-email"
          className="block text-sm font-semibold text-slate-700 dark:text-slate-300"
        >
          {t('payout.emailLabel')}
        </label>
        <input
          id="payout-email"
          name="contactEmail"
          type="email"
          className={FIELD}
          value={contactEmail}
          onChange={(e) => setContactEmail(e.target.value)}
        />
      </div>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      {/* Stated before they submit, not after. */}
      <p className="text-xs text-slate-500 dark:text-slate-400">{t('payout.privacyNote')}</p>

      <Button type="submit" size="sm" disabled={!canSubmit}>
        {connect.isPending ? t('payout.connecting') : t('payout.connect')}
      </Button>
    </form>
  );
}
