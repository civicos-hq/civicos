import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Loader2, XCircle } from 'lucide-react';
import { api } from '../lib/api';

type State =
  { kind: 'verifying' } | { kind: 'success' } | { kind: 'expired' } | { kind: 'invalid' };

export function VerifyEmailPage() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [state, setState] = useState<State>({ kind: 'verifying' });
  const [nextRoute, setNextRoute] = useState<'/discover' | '/onboarding'>('/discover');

  useEffect(() => {
    const token = params.get('token');
    if (!token) {
      setState({ kind: 'invalid' });
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await api.post('/api/v1/auth/verify-email', { token });
        if (cancelled) return;
        // Identity-service returns fresh tokens so the new emailVerified=true
        // claim takes effect immediately for write-gated endpoints.
        const tokens = res.data?.data?.tokens;
        if (tokens?.accessToken) {
          localStorage.setItem('accessToken', tokens.accessToken);
        }
        if (tokens?.refreshToken) {
          localStorage.setItem('refreshToken', tokens.refreshToken);
        }
        await queryClient.invalidateQueries({ queryKey: ['me'] });
        const verifiedUser = res.data?.data?.user;
        if (verifiedUser && !verifiedUser.activeCommunityId) {
          setNextRoute('/onboarding');
        }
        setState({ kind: 'success' });
      } catch (err: unknown) {
        if (cancelled) return;
        const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
        setState({ kind: code === 'VERIFICATION_TOKEN_EXPIRED' ? 'expired' : 'invalid' });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [params, queryClient]);

  return (
    <section className="auth-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <div className="auth-grid auth-grid--single">
        <div className="auth-card auth-card--centered">
          {state.kind === 'verifying' && (
            <>
              <span className="verify-icon verify-icon--info" aria-hidden="true">
                <Loader2 className="h-7 w-7 verify-spin" />
              </span>
              <h1 className="auth-card-title">{t('auth.verify.verifyingTitle')}</h1>
            </>
          )}

          {state.kind === 'success' && (
            <>
              <span className="verify-icon verify-icon--ok" aria-hidden="true">
                <CheckCircle2 className="h-7 w-7" />
              </span>
              <h1 className="auth-card-title">{t('auth.verify.successTitle')}</h1>
              <p className="auth-card-subtitle">{t('auth.verify.successSub')}</p>
              <button type="button" className="auth-submit" onClick={() => navigate(nextRoute)}>
                {t('auth.verify.successCta')}
              </button>
            </>
          )}

          {(state.kind === 'expired' || state.kind === 'invalid') && (
            <>
              <span className="verify-icon verify-icon--err" aria-hidden="true">
                <XCircle className="h-7 w-7" />
              </span>
              <h1 className="auth-card-title">
                {state.kind === 'expired'
                  ? t('auth.verify.expiredTitle')
                  : t('auth.verify.invalidTitle')}
              </h1>
              <p className="auth-card-subtitle">
                {state.kind === 'expired'
                  ? t('auth.verify.expiredSub')
                  : t('auth.verify.invalidSub')}
              </p>
              <Link to="/login" className="auth-submit auth-submit--link">
                {t('auth.verify.errorCta')}
              </Link>
            </>
          )}
        </div>
      </div>
    </section>
  );
}
