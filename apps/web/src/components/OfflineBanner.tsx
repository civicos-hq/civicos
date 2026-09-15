import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import { WifiOff, Wifi } from 'lucide-react';
import { useOnlineStatus } from '../hooks/useOnlineStatus';

/**
 * A single app-wide banner for lost connectivity.
 *
 * When the connection drops, every query on the page fails at once. Without
 * this the user gets a wall of identical red panels and no indication that
 * one shared cause explains all of them — so the banner states the cause once
 * and the individual panels stay quiet about diagnosis.
 *
 * On reconnect it briefly confirms and refetches, because the alternative is
 * a page full of stale error states that a user has to reload manually.
 */
export function OfflineBanner() {
  const { t } = useTranslation();
  const online = useOnlineStatus();
  const queryClient = useQueryClient();
  const [justReconnected, setJustReconnected] = useState(false);

  useEffect(() => {
    if (online) return;
    // Went offline — arm the reconnect confirmation for when we come back.
    setJustReconnected(false);
  }, [online]);

  useEffect(() => {
    if (!online) return;
    let cancelled = false;
    // Only announce a reconnect if something actually failed while we were
    // away; on a normal page load this must stay silent.
    const hadFailures = queryClient
      .getQueryCache()
      .getAll()
      .some((q) => q.state.status === 'error');
    if (!hadFailures) return;

    setJustReconnected(true);
    void queryClient.refetchQueries({ type: 'active' });
    const timer = window.setTimeout(() => {
      if (!cancelled) setJustReconnected(false);
    }, 4000);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [online, queryClient]);

  if (online && !justReconnected) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className={`fixed inset-x-0 top-0 z-50 flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium ${
        online ? 'bg-emerald-600 text-white' : 'bg-slate-900 text-white dark:bg-slate-700'
      }`}
    >
      {online ? (
        <>
          <Wifi className="h-4 w-4" aria-hidden="true" />
          {t('errors.offlineBanner.restored')}
        </>
      ) : (
        <>
          <WifiOff className="h-4 w-4" aria-hidden="true" />
          {t('errors.offlineBanner.offline')}
        </>
      )}
    </div>
  );
}
