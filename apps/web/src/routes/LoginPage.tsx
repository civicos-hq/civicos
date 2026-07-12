import { useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { api } from '../lib/api';
import { LanguageSwitcher } from '../components/LanguageSwitcher';
import { useSeo } from '../hooks/useSeo';

export function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  useSeo({
    title: 'Sign in — CivicOS',
    description:
      'Sign in to your CivicOS account to report issues, sign petitions, and follow your representatives.',
  });
  const location = useLocation();
  // If routed here from "Sign in instead" on the register page, pre-fill the
  // email so the user doesn't retype it.
  const routedEmail = (location.state as { email?: string } | null)?.email ?? '';
  const [email, setEmail] = useState(routedEmail);
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage('');
    setIsSubmitting(true);

    try {
      const response = await api.post('/api/v1/auth/login', { email, password });
      const accessToken = response.data?.data?.tokens?.accessToken;
      const refreshToken = response.data?.data?.tokens?.refreshToken;
      const user = response.data?.data?.user;

      if (!accessToken) {
        throw new Error('ACCESS_TOKEN_MISSING');
      }

      localStorage.setItem('accessToken', accessToken);
      if (refreshToken) {
        localStorage.setItem('refreshToken', refreshToken);
      }

      // Users who never picked a community land on the wizard first.
      // Once they've joined one (or skipped it), they go straight to Discover.
      const destination = user?.activeCommunityId ? '/discover' : '/onboarding';
      navigate(destination, { replace: true });
    } catch {
      setErrorMessage(t('auth.login.error'));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="auth-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <Link to="/" className="auth-home" aria-label={t('common.backToHome')}>
        {t('common.backToHomeShort')}
      </Link>

      <div className="auth-lang">
        <LanguageSwitcher />
      </div>

      <div className="auth-grid">
        <aside className="auth-copy">
          <Link to="/" className="auth-mark-link" aria-label={t('common.backToHome')}>
            <img src="/civicos-mark.png" alt="CivicOS" className="auth-mark" />
          </Link>
          <p className="auth-eyebrow">{t('auth.login.eyebrow')}</p>
          <h1 className="auth-title">{t('auth.login.title')}</h1>
          <p className="auth-description">{t('auth.login.description')}</p>
        </aside>

        <form className="auth-card" onSubmit={onSubmit}>
          <h2 className="auth-card-title">{t('auth.login.cardTitle')}</h2>
          <p className="auth-card-subtitle">{t('auth.login.cardSubtitle')}</p>

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
            autoComplete="current-password"
            required
            minLength={8}
          />

          <div className="auth-forgot-row">
            <Link to="/forgot-password" className="auth-link auth-link--muted">
              {t('auth.forgot.title')}
            </Link>
          </div>

          {errorMessage && <p className="auth-error">{errorMessage}</p>}

          <button type="submit" className="auth-submit" disabled={isSubmitting}>
            {isSubmitting ? t('auth.login.submitting') : t('auth.login.submit')}
          </button>

          <p className="auth-footer">
            {t('auth.login.footerNoAccount')}{' '}
            <Link to="/register" className="auth-link">
              {t('auth.login.footerCreate')}
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
