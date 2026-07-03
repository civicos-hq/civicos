import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Mail, Send, X } from 'lucide-react';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';

const DISMISS_KEY = 'civicos.unverifiedBannerDismissed';

/**
 * Slim chrome strip that nags the user to verify their email. Previously
 * a full-width card in the content flow eating 20% of every page's
 * viewport — now a one-line bar between Topbar and content so real
 * content isn't pushed below the fold.
 *
 * Dismissable per session (sessionStorage) so the user isn't harangued
 * across every navigation. It comes back on a fresh tab / next login,
 * which is the right nag cadence — persistent enough to see, not
 * obnoxious.
 */
export function UnverifiedBanner() {
  const { t } = useTranslation();
  const { data: me } = useMe();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent' | 'error'>('idle');
  const [dismissed, setDismissed] = useState(false);

  // Hydrate the session-scoped dismiss state after mount so SSR / hydration
  // never has to touch sessionStorage.
  useEffect(() => {
    if (sessionStorage.getItem(DISMISS_KEY) === '1') setDismissed(true);
  }, []);

  if (!me || me.emailVerified || dismissed) return null;

  async function resend() {
    setStatus('sending');
    try {
      await api.post('/api/v1/auth/resend-verification');
      setStatus('sent');
      queryClient.invalidateQueries({ queryKey: ['me'] });
    } catch {
      setStatus('error');
    }
  }

  function dismiss() {
    sessionStorage.setItem(DISMISS_KEY, '1');
    setDismissed(true);
  }

  return (
    <div className="verify-strip" role="status">
      <span className="verify-strip-icon" aria-hidden="true">
        <Mail className="h-3.5 w-3.5" />
      </span>
      <p className="verify-strip-body">{t('auth.verify.bannerCompact')}</p>
      {status === 'sent' ? (
        <span className="verify-strip-ack">{t('auth.verify.resent')}</span>
      ) : (
        <button
          type="button"
          className="verify-strip-btn"
          onClick={resend}
          disabled={status === 'sending'}
        >
          <Send className="h-3 w-3" />
          {status === 'sending' ? t('auth.verify.resending') : t('auth.verify.bannerResend')}
        </button>
      )}
      <button
        type="button"
        className="verify-strip-dismiss"
        onClick={dismiss}
        aria-label={t('auth.verify.bannerDismiss')}
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
