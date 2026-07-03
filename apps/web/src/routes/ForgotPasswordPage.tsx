import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, MailCheck } from 'lucide-react';
import { api } from '../lib/api';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

export function ForgotPasswordPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent'>('idle');
  const [submittedTo, setSubmittedTo] = useState('');

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (status === 'sending') return;
    setStatus('sending');
    try {
      await api.post('/api/v1/auth/forgot-password', { email });
      // The API deliberately returns 200 whether the email exists or not, so
      // there is no failure path to surface to the user here.
    } finally {
      setSubmittedTo(email);
      setStatus('sent');
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
          <p className="auth-eyebrow">{t('auth.forgot.eyebrow')}</p>
          <h1 className="auth-title">{t('auth.forgot.title')}</h1>
          <p className="auth-description">{t('auth.forgot.description')}</p>
        </aside>

        {status === 'sent' ? (
          <div className="auth-card auth-card--centered">
            <span className="verify-icon verify-icon--ok" aria-hidden="true">
              <CheckCircle2 className="h-7 w-7" />
            </span>
            <h2 className="auth-card-title">{t('auth.forgot.sentTitle')}</h2>
            <p className="auth-card-subtitle">
              {t('auth.forgot.sentSub', {
                email: submittedTo || t('auth.fields.emailPlaceholder'),
              })}
            </p>
            <p className="verify-tip">{t('auth.forgot.unknownEmailNote')}</p>
            <p className="auth-footer">
              <Link to="/login" className="auth-link">
                {t('auth.forgot.backToLogin')}
              </Link>
            </p>
          </div>
        ) : (
          <form className="auth-card" onSubmit={onSubmit}>
            <h2 className="auth-card-title">{t('auth.forgot.cardTitle')}</h2>
            <p className="auth-card-subtitle">{t('auth.forgot.cardSubtitle')}</p>

            <label className="auth-label" htmlFor="forgot-email">
              {t('auth.fields.email')}
            </label>
            <input
              id="forgot-email"
              type="email"
              className="auth-input"
              placeholder={t('auth.fields.emailPlaceholder')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
            />

            <button type="submit" className="auth-submit" disabled={status === 'sending'}>
              <MailCheck className="h-4 w-4" />
              {status === 'sending' ? t('auth.forgot.submitting') : t('auth.forgot.submit')}
            </button>

            <p className="auth-footer">
              <Link to="/login" className="auth-link">
                {t('auth.forgot.backToLogin')}
              </Link>
            </p>
          </form>
        )}
      </div>
    </section>
  );
}
