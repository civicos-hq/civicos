import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { api } from '../lib/api';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

type RegisterError = { kind: 'emailInUse' } | { kind: 'other'; message: string } | null;

export function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<RegisterError>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const res = await api.post('/api/v1/auth/register', { name, email, password });
      const accessToken = res.data?.data?.tokens?.accessToken;
      const refreshToken = res.data?.data?.tokens?.refreshToken;
      if (accessToken) {
        localStorage.setItem('accessToken', accessToken);
      }
      if (refreshToken) {
        localStorage.setItem('refreshToken', refreshToken);
      }
      navigate('/verify-email-sent', { replace: true, state: { email } });
    } catch (err) {
      const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
      if (code === 'EMAIL_ALREADY_IN_USE') {
        setError({ kind: 'emailInUse' });
      } else {
        console.error('Registration error:', err);
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

        <form className="auth-card" onSubmit={onSubmit}>
          <h2 className="auth-card-title">{t('auth.register.cardTitle')}</h2>
          <p className="auth-card-subtitle">{t('auth.register.cardSubtitle')}</p>

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
            autoComplete="new-password"
            required
            minLength={8}
          />

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
