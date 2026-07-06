import { useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { RequestedAccountType, type ApiResponse, type Community } from '@civicos/types';
import { api } from '../lib/api';
import { LanguageSwitcher } from '../components/LanguageSwitcher';
import { NIGERIAN_STATES } from '../data/nigeria';

type RegisterError = { kind: 'emailInUse' } | { kind: 'other'; message: string } | null;

type AccountType = RequestedAccountType;

const ORG_KINDS = ['GOVERNMENT', 'AGENCY', 'NGO', 'UTILITY', 'OTHER'] as const;
const ORG_JURISDICTIONS = ['NATIONAL', 'STATE', 'LGA', 'COMMUNITY'] as const;
const ACCOUNT_TYPE_OPTIONS = [
  RequestedAccountType.CITIZEN,
  RequestedAccountType.REPRESENTATIVE,
  RequestedAccountType.ORGANIZATION,
] as const;

function toSlug(input: string): string {
  return input
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');
}

function parseProofUrls(input: string): string[] {
  return input
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [accountType, setAccountType] = useState<AccountType>(RequestedAccountType.CITIZEN);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<RegisterError>(null);

  const [repTitle, setRepTitle] = useState('Hon.');
  const [repPosition, setRepPosition] = useState('');
  const [repConstituency, setRepConstituency] = useState('');
  const [repCommunityId, setRepCommunityId] = useState('');
  const [repParty, setRepParty] = useState('');
  const [repBio, setRepBio] = useState('');
  const [repOfficialEmail, setRepOfficialEmail] = useState('');
  const [repOfficialPhone, setRepOfficialPhone] = useState('');
  const [repWebsite, setRepWebsite] = useState('');
  const [repProofUrls, setRepProofUrls] = useState('');

  const [orgName, setOrgName] = useState('');
  const [orgSlug, setOrgSlug] = useState('');
  const [orgSlugTouched, setOrgSlugTouched] = useState(false);
  const [orgKind, setOrgKind] = useState<(typeof ORG_KINDS)[number]>('GOVERNMENT');
  const [orgJurisdiction, setOrgJurisdiction] = useState<(typeof ORG_JURISDICTIONS)[number]>('LGA');
  const [orgState, setOrgState] = useState('');
  const [orgLga, setOrgLga] = useState('');
  const [orgDescription, setOrgDescription] = useState('');
  const [orgOfficialEmail, setOrgOfficialEmail] = useState('');
  const [orgOfficialPhone, setOrgOfficialPhone] = useState('');
  const [orgWebsite, setOrgWebsite] = useState('');
  const [orgProofUrls, setOrgProofUrls] = useState('');

  const communitiesQuery = useQuery({
    queryKey: ['register-communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });

  const lgaOptions = useMemo(
    () => NIGERIAN_STATES.find((state) => state.name === orgState)?.lgas ?? [],
    [orgState],
  );
  const isRepresentative = accountType === RequestedAccountType.REPRESENTATIVE;
  const isOrganization = accountType === RequestedAccountType.ORGANIZATION;
  const requiresApplication = isRepresentative || isOrganization;
  const orgNeedsState =
    orgJurisdiction === 'STATE' || orgJurisdiction === 'LGA' || orgJurisdiction === 'COMMUNITY';
  const orgNeedsLga = orgJurisdiction === 'LGA' || orgJurisdiction === 'COMMUNITY';

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const payload: Record<string, unknown> = {
        name,
        email,
        password,
        requestedAccountType: accountType,
      };

      if (accountType === 'REPRESENTATIVE') {
        payload.representativeApplication = {
          fullName: name.trim(),
          title: repTitle.trim(),
          position: repPosition.trim(),
          constituency: repConstituency.trim(),
          communityId: repCommunityId,
          party: repParty.trim() || undefined,
          bio: repBio.trim() || undefined,
          officialEmail: repOfficialEmail.trim() || undefined,
          officialPhone: repOfficialPhone.trim() || undefined,
          website: repWebsite.trim() || undefined,
          proofUrls: parseProofUrls(repProofUrls),
        };
      }

      if (accountType === 'ORGANIZATION') {
        payload.organizationApplication = {
          name: orgName.trim(),
          slug: (orgSlug.trim() || toSlug(orgName)).toLowerCase(),
          kind: orgKind,
          jurisdiction: orgJurisdiction,
          state: orgNeedsState && orgState ? orgState : undefined,
          lga: orgNeedsLga && orgLga ? orgLga : undefined,
          description: orgDescription.trim() || undefined,
          officialEmail: orgOfficialEmail.trim() || undefined,
          officialPhone: orgOfficialPhone.trim() || undefined,
          website: orgWebsite.trim() || undefined,
          proofUrls: parseProofUrls(orgProofUrls),
        };
      }

      const res = await api.post('/api/v1/auth/register', payload);
      const accessToken = res.data?.data?.tokens?.accessToken;
      const refreshToken = res.data?.data?.tokens?.refreshToken;
      if (accessToken) localStorage.setItem('accessToken', accessToken);
      if (refreshToken) localStorage.setItem('refreshToken', refreshToken);
      navigate('/verify-email-sent', {
        replace: true,
        state: { email, requestedAccountType: accountType },
      });
    } catch (err) {
      const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
      if (code === 'EMAIL_ALREADY_IN_USE') {
        setError({ kind: 'emailInUse' });
      } else {
        setError({ kind: 'other', message: t('auth.register.error') });
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="auth-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <div className="auth-lang">
        <LanguageSwitcher />
      </div>

      <div className="auth-grid">
        <aside className="auth-copy">
          <img src="/civicos-mark.png" alt="CivicOS" className="auth-mark" />
          <p className="auth-eyebrow">{t('auth.register.eyebrow')}</p>
          <h1 className="auth-title">{t('auth.register.title')}</h1>
          <p className="auth-description">{t('auth.register.description')}</p>
        </aside>

        <form className="auth-card auth-card--register" onSubmit={onSubmit}>
          <div className="auth-card-head">
            <div>
              <h2 className="auth-card-title">{t('auth.register.cardTitle')}</h2>
              <p className="auth-card-subtitle">{t('auth.register.cardSubtitle')}</p>
            </div>
            <span className="auth-account-badge">
              {t(`auth.register.accountType.options.${accountType}.label`)}
            </span>
          </div>

          <div className="auth-progress" aria-hidden="true">
            <span className="auth-progress-step auth-progress-step--active">
              <span className="auth-progress-dot">1</span>
              {t('auth.register.progress.path')}
            </span>
            <span className="auth-progress-step auth-progress-step--active">
              <span className="auth-progress-dot">2</span>
              {t('auth.register.progress.account')}
            </span>
            <span
              className={`auth-progress-step${requiresApplication ? ' auth-progress-step--active' : ''}`}
            >
              <span className="auth-progress-dot">3</span>
              {requiresApplication
                ? t('auth.register.progress.review')
                : t('auth.register.progress.finish')}
            </span>
          </div>

          <fieldset className="auth-choice-group">
            <legend className="auth-label">{t('auth.register.accountType.legend')}</legend>
            <div className="auth-choice-list">
              {ACCOUNT_TYPE_OPTIONS.map((value) => (
                <label key={value} className="auth-choice-card">
                  <input
                    type="radio"
                    name="accountType"
                    value={value}
                    checked={accountType === value}
                    onChange={() => setAccountType(value)}
                  />
                  <span className="auth-choice-copy">
                    <strong>{t(`auth.register.accountType.options.${value}.label`)}</strong>
                    <span>{t(`auth.register.accountType.options.${value}.hint`)}</span>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          <section className="auth-form-section">
            <div className="auth-section-header">
              <div>
                <div className="auth-section-title">
                  {t('auth.register.sections.account.title')}
                </div>
                <p className="auth-section-sub">{t('auth.register.sections.account.sub')}</p>
              </div>
            </div>

            <div className="auth-fields-grid auth-fields-grid--two">
              <div className="auth-field">
                <label className="auth-label" htmlFor="name">
                  {t('auth.fields.fullName')}
                </label>
                <input
                  id="name"
                  type="text"
                  className="auth-input"
                  placeholder={t('auth.fields.namePlaceholder')}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  autoComplete="name"
                  required
                  minLength={2}
                />
              </div>

              <div className="auth-field">
                <label className="auth-label" htmlFor="email">
                  {t('auth.fields.email')}
                </label>
                <input
                  id="email"
                  type="email"
                  className="auth-input"
                  placeholder={t('auth.fields.emailPlaceholder')}
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  autoComplete="email"
                  required
                />
              </div>

              <div className="auth-field auth-field--span-2">
                <label className="auth-label" htmlFor="password">
                  {t('auth.fields.password')}
                </label>
                <input
                  id="password"
                  type="password"
                  className="auth-input"
                  placeholder={t('auth.fields.passwordPlaceholder')}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="new-password"
                  required
                  minLength={8}
                />
              </div>
            </div>
          </section>

          {!requiresApplication && (
            <div className="auth-note auth-note--success">
              <strong>{t('auth.register.citizenNotice.title')}</strong>
              <span>{t('auth.register.citizenNotice.body')}</span>
            </div>
          )}

          {isRepresentative && (
            <section className="auth-form-section">
              <div className="auth-section-header">
                <div>
                  <div className="auth-section-title">
                    {t('auth.register.representative.title')}
                  </div>
                  <p className="auth-section-sub">{t('auth.register.representative.sub')}</p>
                </div>
              </div>

              <div className="auth-note">
                <strong>{t('auth.register.applicationNotice.title')}</strong>
                <span>{t('auth.register.applicationNotice.body')}</span>
              </div>

              <div className="auth-fields-grid auth-fields-grid--two">
                <div className="auth-field">
                  <label className="auth-label" htmlFor="rep-title">
                    {t('auth.register.representative.fields.title')}
                  </label>
                  <input
                    id="rep-title"
                    type="text"
                    className="auth-input"
                    value={repTitle}
                    onChange={(event) => setRepTitle(event.target.value)}
                    required
                  />
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="rep-position">
                    {t('auth.register.representative.fields.position')}
                  </label>
                  <input
                    id="rep-position"
                    type="text"
                    className="auth-input"
                    placeholder={t('auth.register.representative.fields.positionPlaceholder')}
                    value={repPosition}
                    onChange={(event) => setRepPosition(event.target.value)}
                    required
                  />
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="rep-constituency">
                    {t('auth.register.representative.fields.constituency')}
                  </label>
                  <input
                    id="rep-constituency"
                    type="text"
                    className="auth-input"
                    placeholder={t('auth.register.representative.fields.constituencyPlaceholder')}
                    value={repConstituency}
                    onChange={(event) => setRepConstituency(event.target.value)}
                    required
                  />
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="rep-community">
                    {t('auth.register.representative.fields.community')}
                  </label>
                  <select
                    id="rep-community"
                    className="auth-input"
                    value={repCommunityId}
                    onChange={(event) => setRepCommunityId(event.target.value)}
                    required
                    disabled={communitiesQuery.isLoading}
                  >
                    <option value="">
                      {communitiesQuery.isLoading
                        ? t('auth.register.representative.loadingCommunities')
                        : t('auth.register.representative.fields.communityPlaceholder')}
                    </option>
                    {(communitiesQuery.data ?? []).map((community) => (
                      <option key={community.id} value={community.id}>
                        {community.name} · {community.state} / {community.lga}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <details className="auth-disclosure">
                <summary>{t('auth.register.optionalSection')}</summary>
                <div className="auth-fields-grid auth-fields-grid--two">
                  <div className="auth-field">
                    <label className="auth-label" htmlFor="rep-party">
                      {t('auth.register.representative.fields.party')}
                    </label>
                    <input
                      id="rep-party"
                      type="text"
                      className="auth-input"
                      placeholder={t('auth.register.representative.fields.partyPlaceholder')}
                      value={repParty}
                      onChange={(event) => setRepParty(event.target.value)}
                    />
                  </div>

                  <div className="auth-field">
                    <label className="auth-label" htmlFor="rep-official-email">
                      {t('auth.register.representative.fields.officialEmail')}
                    </label>
                    <input
                      id="rep-official-email"
                      type="email"
                      className="auth-input"
                      placeholder={t(
                        'auth.register.representative.fields.officialEmailPlaceholder',
                      )}
                      value={repOfficialEmail}
                      onChange={(event) => setRepOfficialEmail(event.target.value)}
                    />
                  </div>

                  <div className="auth-field">
                    <label className="auth-label" htmlFor="rep-official-phone">
                      {t('auth.register.representative.fields.officialPhone')}
                    </label>
                    <input
                      id="rep-official-phone"
                      type="text"
                      className="auth-input"
                      placeholder={t(
                        'auth.register.representative.fields.officialPhonePlaceholder',
                      )}
                      value={repOfficialPhone}
                      onChange={(event) => setRepOfficialPhone(event.target.value)}
                    />
                  </div>

                  <div className="auth-field">
                    <label className="auth-label" htmlFor="rep-website">
                      {t('auth.register.representative.fields.website')}
                    </label>
                    <input
                      id="rep-website"
                      type="url"
                      className="auth-input"
                      placeholder={t('auth.register.representative.fields.websitePlaceholder')}
                      value={repWebsite}
                      onChange={(event) => setRepWebsite(event.target.value)}
                    />
                  </div>

                  <div className="auth-field auth-field--span-2">
                    <label className="auth-label" htmlFor="rep-bio">
                      {t('auth.register.representative.fields.bio')}
                    </label>
                    <textarea
                      id="rep-bio"
                      className="auth-input auth-textarea"
                      placeholder={t('auth.register.representative.fields.bioPlaceholder')}
                      value={repBio}
                      onChange={(event) => setRepBio(event.target.value)}
                      rows={3}
                    />
                  </div>

                  <div className="auth-field auth-field--span-2">
                    <label className="auth-label" htmlFor="rep-proof">
                      {t('auth.register.representative.fields.proofUrls')}
                    </label>
                    <input
                      id="rep-proof"
                      type="text"
                      className="auth-input"
                      placeholder={t('auth.register.representative.fields.proofUrlsPlaceholder')}
                      value={repProofUrls}
                      onChange={(event) => setRepProofUrls(event.target.value)}
                    />
                  </div>
                </div>
              </details>
            </section>
          )}

          {isOrganization && (
            <section className="auth-form-section">
              <div className="auth-section-header">
                <div>
                  <div className="auth-section-title">{t('auth.register.organization.title')}</div>
                  <p className="auth-section-sub">{t('auth.register.organization.sub')}</p>
                </div>
              </div>

              <div className="auth-note">
                <strong>{t('auth.register.applicationNotice.title')}</strong>
                <span>{t('auth.register.applicationNotice.body')}</span>
              </div>

              <div className="auth-fields-grid auth-fields-grid--two">
                <div className="auth-field">
                  <label className="auth-label" htmlFor="org-name">
                    {t('auth.register.organization.fields.name')}
                  </label>
                  <input
                    id="org-name"
                    type="text"
                    className="auth-input"
                    placeholder={t('auth.register.organization.fields.namePlaceholder')}
                    value={orgName}
                    onChange={(event) => {
                      setOrgName(event.target.value);
                      if (!orgSlugTouched) setOrgSlug(toSlug(event.target.value));
                    }}
                    required
                  />
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="org-slug">
                    {t('auth.register.organization.fields.slug')}
                  </label>
                  <input
                    id="org-slug"
                    type="text"
                    className="auth-input"
                    placeholder={t('auth.register.organization.fields.slugPlaceholder')}
                    value={orgSlug}
                    onChange={(event) => {
                      setOrgSlug(toSlug(event.target.value));
                      setOrgSlugTouched(true);
                    }}
                    required
                  />
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="org-kind">
                    {t('auth.register.organization.fields.kind')}
                  </label>
                  <select
                    id="org-kind"
                    className="auth-input"
                    value={orgKind}
                    onChange={(event) =>
                      setOrgKind(event.target.value as (typeof ORG_KINDS)[number])
                    }
                  >
                    {ORG_KINDS.map((kind) => (
                      <option key={kind} value={kind}>
                        {t(`auth.register.organization.kinds.${kind}`)}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="auth-field">
                  <label className="auth-label" htmlFor="org-jurisdiction">
                    {t('auth.register.organization.fields.jurisdiction')}
                  </label>
                  <select
                    id="org-jurisdiction"
                    className="auth-input"
                    value={orgJurisdiction}
                    onChange={(event) => {
                      const next = event.target.value as (typeof ORG_JURISDICTIONS)[number];
                      setOrgJurisdiction(next);
                      if (next === 'NATIONAL') {
                        setOrgState('');
                        setOrgLga('');
                      }
                    }}
                  >
                    {ORG_JURISDICTIONS.map((jurisdiction) => (
                      <option key={jurisdiction} value={jurisdiction}>
                        {t(`auth.register.organization.jurisdictions.${jurisdiction}`)}
                      </option>
                    ))}
                  </select>
                </div>

                {orgNeedsState && (
                  <div className="auth-field">
                    <label className="auth-label" htmlFor="org-state">
                      {t('auth.register.organization.fields.state')}
                    </label>
                    <select
                      id="org-state"
                      className="auth-input"
                      value={orgState}
                      onChange={(event) => {
                        setOrgState(event.target.value);
                        setOrgLga('');
                      }}
                      required
                    >
                      <option value="">
                        {t('auth.register.organization.fields.statePlaceholder')}
                      </option>
                      {NIGERIAN_STATES.map((state) => (
                        <option key={state.code} value={state.name}>
                          {state.name}
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                {orgNeedsLga && (
                  <div className="auth-field">
                    <label className="auth-label" htmlFor="org-lga">
                      {t('auth.register.organization.fields.lga')}
                    </label>
                    <select
                      id="org-lga"
                      className="auth-input"
                      value={orgLga}
                      onChange={(event) => setOrgLga(event.target.value)}
                      required
                      disabled={!orgState}
                    >
                      <option value="">
                        {t('auth.register.organization.fields.lgaPlaceholder')}
                      </option>
                      {lgaOptions.map((lga) => (
                        <option key={lga} value={lga}>
                          {lga}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
              </div>

              <details className="auth-disclosure">
                <summary>{t('auth.register.optionalSection')}</summary>
                <div className="auth-fields-grid auth-fields-grid--two">
                  <div className="auth-field auth-field--span-2">
                    <label className="auth-label" htmlFor="org-description">
                      {t('auth.register.organization.fields.description')}
                    </label>
                    <textarea
                      id="org-description"
                      className="auth-input auth-textarea"
                      placeholder={t('auth.register.organization.fields.descriptionPlaceholder')}
                      value={orgDescription}
                      onChange={(event) => setOrgDescription(event.target.value)}
                      rows={3}
                    />
                  </div>

                  <div className="auth-field">
                    <label className="auth-label" htmlFor="org-official-email">
                      {t('auth.register.organization.fields.officialEmail')}
                    </label>
                    <input
                      id="org-official-email"
                      type="email"
                      className="auth-input"
                      placeholder={t('auth.register.organization.fields.officialEmailPlaceholder')}
                      value={orgOfficialEmail}
                      onChange={(event) => setOrgOfficialEmail(event.target.value)}
                    />
                  </div>

                  <div className="auth-field">
                    <label className="auth-label" htmlFor="org-official-phone">
                      {t('auth.register.organization.fields.officialPhone')}
                    </label>
                    <input
                      id="org-official-phone"
                      type="text"
                      className="auth-input"
                      placeholder={t('auth.register.organization.fields.officialPhonePlaceholder')}
                      value={orgOfficialPhone}
                      onChange={(event) => setOrgOfficialPhone(event.target.value)}
                    />
                  </div>

                  <div className="auth-field auth-field--span-2">
                    <label className="auth-label" htmlFor="org-website">
                      {t('auth.register.organization.fields.website')}
                    </label>
                    <input
                      id="org-website"
                      type="url"
                      className="auth-input"
                      placeholder={t('auth.register.organization.fields.websitePlaceholder')}
                      value={orgWebsite}
                      onChange={(event) => setOrgWebsite(event.target.value)}
                    />
                  </div>

                  <div className="auth-field auth-field--span-2">
                    <label className="auth-label" htmlFor="org-proof">
                      {t('auth.register.organization.fields.proofUrls')}
                    </label>
                    <input
                      id="org-proof"
                      type="text"
                      className="auth-input"
                      placeholder={t('auth.register.organization.fields.proofUrlsPlaceholder')}
                      value={orgProofUrls}
                      onChange={(event) => setOrgProofUrls(event.target.value)}
                    />
                  </div>
                </div>
              </details>
            </section>
          )}

          {error?.kind === 'other' && <p className="auth-error">{error.message}</p>}

          {error?.kind === 'emailInUse' && (
            <div className="auth-hint">
              <p className="auth-hint-title">{t('auth.register.emailInUse')}</p>
              <p className="auth-hint-sub">
                <Link to="/login" state={{ email }} className="auth-link">
                  {t('auth.register.emailInUseSignIn')}
                </Link>{' '}
                <Link to="/forgot-password" className="auth-link">
                  {t('auth.register.emailInUseForgot')}
                </Link>
                .
              </p>
            </div>
          )}

          <button type="submit" className="auth-submit" disabled={isSubmitting}>
            {isSubmitting ? t('auth.register.submitting') : t('auth.register.submit')}
          </button>

          <p className="auth-footer">
            {t('auth.register.footerHave')}{' '}
            <Link to="/login" className="auth-link">
              {t('auth.register.footerSignIn')}
            </Link>
          </p>
          <p className="auth-legal">
            <Link to="/privacy" className="auth-link auth-link--muted">
              {t('footer.legal.privacy')}
            </Link>
            {' · '}
            <Link to="/terms" className="auth-link auth-link--muted">
              {t('footer.legal.terms')}
            </Link>
          </p>
        </form>
      </div>
    </section>
  );
}
