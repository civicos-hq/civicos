import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, KeyRound, XCircle } from 'lucide-react';
import { api } from '../lib/api';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

type State =
  | { kind: 'form' }
  | { kind: 'submitting' }
  | { kind: 'success' }
  | { kind: 'expired' }
  | { kind: 'invalid' }
  | { kind: 'other'; message: string };

export function ResetPasswordPage() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const token = params.get('token') ?? '';

  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [clientError, setClientError] = useState('');
  const [state, setState] = useState<State>(token ? { kind: 'form' } : { kind: 'invalid' });

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setClientError('');
    if (password.length < 8) {
      setClientError(t('auth.reset.tooShort'));
      return;
    }
    if (password !== confirm) {
      setClientError(t('auth.reset.mismatch'));
      return;
    }
    setState({ kind: 'submitting' });
    try {
      const res = await api.post('/api/v1/auth/reset-password', {
        token,
        newPassword: password,
      });
      // Auto-login: identity-service returns fresh tokens so the user lands
      // on the dashboard without re-entering credentials.
      const tokens = res.data?.data?.tokens;
      if (tokens?.accessToken) {
        localStorage.setItem('accessToken', tokens.accessToken);
      }
      if (tokens?.refreshToken) {
        localStorage.setItem('refreshToken', tokens.refreshToken);
      }
      await queryClient.invalidateQueries({ queryKey: ['me'] });
      setState({ kind: 'success' });
    } catch (err: unknown) {
      const code = (err as { response?: { data?: { code?: string; message?: string } } })?.response
        ?.data?.code;
      if (code === 'RESET_TOKEN_EXPIRED') setState({ kind: 'expired' });
      else if (code === 'RESET_TOKEN_INVALID') setState({ kind: 'invalid' });
      else setState({ kind: 'form' });
      if (code === 'PASSWORD_TOO_SHORT') {
        setClientError(t('auth.reset.tooShort'));
      }
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
          <p className="auth-eyebrow">{t('auth.reset.eyebrow')}</p>
          <h1 className="auth-title">{t('auth.reset.title')}</h1>
          <p className="auth-description">{t('auth.reset.description')}</p>
        </aside>

        {(state.kind === 'form' || state.kind === 'submitting') && (
          <form className="auth-card" onSubmit={onSubmit}>
            <h2 className="auth-card-title">{t('auth.reset.cardTitle')}</h2>

            <label className="auth-label" htmlFor="new-password">
              {t('auth.reset.newPassword')}
            </label>
            <input
              id="new-password"
              type="password"
              className="auth-input"
              placeholder={t('auth.reset.placeholder')}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              required
              minLength={8}
            />

            <label className="auth-label" htmlFor="confirm-password">
              {t('auth.reset.confirmPassword')}
            </label>
            <input
              id="confirm-password"
              type="password"
              className="auth-input"
              placeholder={t('auth.reset.placeholder')}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              autoComplete="new-password"
              required
              minLength={8}
            />

            {clientError && <p className="auth-error">{clientError}</p>}

            <button type="submit" className="auth-submit" disabled={state.kind === 'submitting'}>
              <KeyRound className="h-4 w-4" />
              {state.kind === 'submitting' ? t('auth.reset.submitting') : t('auth.reset.submit')}
            </button>
          </form>
        )}

        {state.kind === 'success' && (
          <div className="auth-card auth-card--centered">
            <span className="verify-icon verify-icon--ok" aria-hidden="true">
              <CheckCircle2 className="h-7 w-7" />
            </span>
            <h2 className="auth-card-title">{t('auth.reset.successTitle')}</h2>
            <p className="auth-card-subtitle">{t('auth.reset.successSub')}</p>
            <button
              type="button"
              className="auth-submit"
              onClick={() => {
                // If the just-reset user still has no community, drop them
                // into the wizard rather than the empty-feed Discover.
                const cached = queryClient.getQueryData<{ activeCommunityId?: string | null }>([
                  'me',
                ]);
                navigate(cached?.activeCommunityId ? '/discover' : '/onboarding');
              }}
            >
              {t('auth.reset.successCta')}
            </button>
          </div>
        )}

        {(state.kind === 'expired' || state.kind === 'invalid') && (
          <div className="auth-card auth-card--centered">
            <span className="verify-icon verify-icon--err" aria-hidden="true">
              <XCircle className="h-7 w-7" />
            </span>
            <h2 className="auth-card-title">
              {state.kind === 'expired'
                ? t('auth.reset.expiredTitle')
                : t('auth.reset.invalidTitle')}
            </h2>
            <p className="auth-card-subtitle">
              {state.kind === 'expired' ? t('auth.reset.expiredSub') : t('auth.reset.invalidSub')}
            </p>
            <Link to="/forgot-password" className="auth-submit auth-submit--link">
              {t('auth.reset.errorCta')}
            </Link>
          </div>
        )}
      </div>
    </section>
  );
}
