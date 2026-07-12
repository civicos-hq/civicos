import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MailCheck, Send } from 'lucide-react';
import type { RequestedAccountType } from '@civicos/types';
import { api } from '../lib/api';

type LocationState = { email?: string; requestedAccountType?: RequestedAccountType };

export function VerifyEmailSentPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const state = (location.state as LocationState | null) ?? null;
  const email = state?.email ?? '';
  const requestedAccountType = state?.requestedAccountType ?? 'CITIZEN';
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent' | 'error'>('idle');

  async function resend() {
    setStatus('sending');
    try {
      await api.post('/api/v1/auth/resend-verification');
      setStatus('sent');
    } catch {
      setStatus('error');
    }
  }

  return (
    <section className="auth-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <Link to="/" className="auth-home" aria-label={t('common.backToHome')}>
        {t('common.backToHomeShort')}
      </Link>

      <div className="auth-grid auth-grid--single">
        <div className="auth-card auth-card--centered">
          <span className="verify-icon verify-icon--info" aria-hidden="true">
            <MailCheck className="h-7 w-7" />
          </span>
          <h1 className="auth-card-title">{t('auth.verify.sentTitle')}</h1>
          <p className="auth-card-subtitle">
            {t('auth.verify.sentSub', { email: email || 'your inbox' })}
          </p>
          <p className="verify-tip">{t('auth.verify.sentTip')}</p>
          {requestedAccountType !== 'CITIZEN' && (
            <p className="verify-tip">{t('auth.verify.pendingApproval')}</p>
          )}

          <button
            type="button"
            className="auth-submit"
            onClick={resend}
            disabled={status === 'sending'}
          >
            <Send className="h-4 w-4" />
            {status === 'sending' ? t('auth.verify.resending') : t('auth.verify.resend')}
          </button>
          {status === 'sent' && <p className="verify-success">{t('auth.verify.resent')}</p>}
          {status === 'error' && <p className="auth-error">{t('auth.register.error')}</p>}

          <p className="auth-footer">
            <Link to="/login" className="auth-link">
              {t('auth.verify.goToLogin')}
            </Link>
          </p>
        </div>
      </div>
    </section>
  );
}
