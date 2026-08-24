import { useEffect, useState } from 'react';

/**
 * Tracks whether the browser believes it has a network connection.
 *
 * Caveat worth knowing: `navigator.onLine` is true for any connected
 * interface, even one with no route to the internet (captive portals, a
 * router with no upstream). So `false` is trustworthy and `true` is not —
 * which is why this only drives the offline banner, and per-request failures
 * are still classified individually by classifyError().
 */
export function useOnlineStatus(): boolean {
  const [online, setOnline] = useState(() =>
    typeof navigator === 'undefined' ? true : navigator.onLine,
  );

  useEffect(() => {
    const goOnline = () => setOnline(true);
    const goOffline = () => setOnline(false);
    window.addEventListener('online', goOnline);
    window.addEventListener('offline', goOffline);
    return () => {
      window.removeEventListener('online', goOnline);
      window.removeEventListener('offline', goOffline);
    };
  }, []);

  return online;
}
