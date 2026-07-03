import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Mail, Send } from 'lucide-react';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';

export function UnverifiedBanner() {
  const { t } = useTranslation();
  const { data: me } = useMe();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent' | 'error'>('idle');

  if (!me || me.emailVerified) return null;

  async function resend() {
    setStatus('sending');
    try {
      await api.post('/api/v1/auth/resend-verification');
      setStatus('sent');
      // Refresh /me so a separate-tab verification reflects here.
      queryClient.invalidateQueries({ queryKey: ['me'] });
    } catch {
      setStatus('error');
    }
  }

  return (
    <div className="verify-banner" role="status">
      <span className="verify-banner-icon" aria-hidden="true">
        <Mail className="h-4 w-4" />
      </span>
      <div className="verify-banner-body">
        <p className="verify-banner-title">{t('auth.verify.bannerTitle')}</p>
        <p className="verify-banner-sub">{t('auth.verify.bannerSub', { email: me.email })}</p>
      </div>
      <div className="verify-banner-actions">
        {status === 'sent' ? (
          <span className="verify-banner-ack">{t('auth.verify.resent')}</span>
        ) : (
          <button
            type="button"
            className="verify-banner-btn"
            onClick={resend}
            disabled={status === 'sending'}
          >
            <Send className="h-3.5 w-3.5" />
            {status === 'sending' ? t('auth.verify.resending') : t('auth.verify.bannerResend')}
          </button>
        )}
      </div>
    </div>
  );
}
