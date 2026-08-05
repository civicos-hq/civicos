import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MapPin, ShieldCheck } from 'lucide-react';
import {
  categoryKey,
  formatMoney,
  progressPercent,
  type PublicCampaign,
} from '../hooks/useCampaigns';

/**
 * One campaign, as shown anywhere it is listed — the browse page and the
 * homepage funding section.
 *
 * Shared rather than duplicated per surface: this card states how much money
 * a campaign has raised against its goal, and two copies of that would be two
 * places for the figure to drift. A visitor should see the same numbers on the
 * homepage as on the campaign's own page.
 */
export function CampaignCard({ c }: { c: PublicCampaign }) {
  const { t, i18n } = useTranslation();
  const pct = progressPercent(c.raisedMinor, c.goalMinor);
  const place = [c.lga, c.state].filter(Boolean).join(', ');

  return (
    <Link to={`/campaigns/${c.slug}`} className="fund-card">
      {c.coverImageUrl && (
        <img className="fund-card-img" src={c.coverImageUrl} alt="" loading="lazy" />
      )}
      <div className="fund-card-body">
        <div className="fund-card-tags">
          {c.isEmergency && (
            <span className="fund-tag fund-tag--emergency">{t('campaigns.emergency')}</span>
          )}
          <span className="fund-tag">{t(`campaigns.categories.${categoryKey(c.category)}`)}</span>
        </div>

        <h3 className="fund-card-title">{c.title}</h3>
        <p className="fund-card-summary">{c.summary}</p>

        {c.organizationName && (
          <p className="fund-card-org">
            <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
            {c.organizationName}
          </p>
        )}
        {place && (
          <p className="fund-card-place">
            <MapPin className="h-3.5 w-3.5" aria-hidden="true" />
            {place}
          </p>
        )}

        {/* Progress is shown even at zero — a campaign that has raised
            nothing yet is exactly what a citizen should be able to see. */}
        <div
          className="fund-progress"
          role="progressbar"
          aria-valuenow={pct}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={t('campaigns.progressLabel', { percent: pct })}
        >
          <span className="fund-progress-fill" style={{ width: `${pct}%` }} />
        </div>
        <p className="fund-card-money">
          <strong>{formatMoney(c.raisedMinor, c.currency, i18n.language)}</strong>{' '}
          {t('campaigns.ofGoal', {
            goal: formatMoney(c.goalMinor, c.currency, i18n.language),
          })}
        </p>
      </div>
    </Link>
  );
}
