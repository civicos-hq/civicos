import { Link, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, ShieldCheck } from 'lucide-react';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';

/**
 * Where Paystack returns the donor after checkout.
 *
 * This page deliberately does NOT claim the donation succeeded.
 *
 * Landing here only means the browser came back from Paystack — a donor can
 * reach this URL by closing the payment sheet, hitting back, or typing it in.
 * Settlement is decided solely by the signed webhook, so the honest thing to
 * say is "we're confirming", not "thank you, it worked". Telling someone
 * their flood-relief donation went through when it may not have is a worse
 * failure than making them wait a moment for confirmation.
 */
export function DonationCompletePage() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  // Paystack appends both `reference` and `trxref`; either may be present.
  const reference = params.get('reference') ?? params.get('trxref');

  useSeo({
    title: t('campaigns.complete.seoTitle'),
    description: t('campaigns.complete.body'),
  });

  return (
    <div className="home-shell">
      <TopNav />
      <section className="home-section fund-section">
        <div className="fund-complete">
          <CheckCircle2 className="h-10 w-10 fund-complete-icon" aria-hidden="true" />
          <h1 className="fund-complete-title">{t('campaigns.complete.title')}</h1>
          <p className="fund-complete-body">{t('campaigns.complete.body')}</p>

          {reference && (
            <p className="fund-complete-ref">
              {t('campaigns.complete.reference')} <code>{reference}</code>
            </p>
          )}

          <p className="fund-complete-note">
            <ShieldCheck className="h-4 w-4" aria-hidden="true" />
            {t('campaigns.complete.receiptNote')}
          </p>

          <Link to="/campaigns" className="home-btn home-btn-primary">
            {t('campaigns.complete.backToCampaigns')}
          </Link>
        </div>
      </section>
      <Footer />
    </div>
  );
}
