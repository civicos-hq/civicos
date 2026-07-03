import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle } from 'lucide-react';
import { RATE_LIMIT_EVENT, type RateLimitEvent } from '../lib/api';

/**
 * Global toast that listens for the "rate limited" event dispatched by the
 * axios interceptor. Auto-dismisses after retryAfter seconds so the user
 * sees the message *and* has a natural cue to try again.
 *
 * Mount once at the layout level (both dashboard and auth shells). We
 * deliberately avoid a Portal — the fixed positioning + z-index is enough
 * and keeps the component leaf-simple.
 */
export function RateLimitToast() {
  const { t } = useTranslation();
  const [remaining, setRemaining] = useState(0);

  useEffect(() => {
    function onEvent(ev: Event) {
      const detail = (ev as CustomEvent<RateLimitEvent>).detail;
      // Clamp so a huge retry-after (misconfigured server) doesn't pin the
      // toast on screen for minutes. 30s is more than the user needs to read.
      setRemaining(Math.min(30, Math.max(1, Math.round(detail.retryAfter))));
    }
    window.addEventListener(RATE_LIMIT_EVENT, onEvent);
    return () => window.removeEventListener(RATE_LIMIT_EVENT, onEvent);
  }, []);

  useEffect(() => {
    if (remaining <= 0) return;
    const id = window.setInterval(() => {
      setRemaining((r) => Math.max(0, r - 1));
    }, 1000);
    return () => window.clearInterval(id);
  }, [remaining]);

  if (remaining <= 0) return null;

  return (
    <div className="rate-limit-toast" role="status" aria-live="polite">
      <AlertTriangle className="h-4 w-4 rate-limit-toast-icon" aria-hidden="true" />
      <div>
        <p className="rate-limit-toast-title">{t('rateLimit.title')}</p>
        <p className="rate-limit-toast-body">{t('rateLimit.body', { seconds: remaining })}</p>
      </div>
    </div>
  );
}
