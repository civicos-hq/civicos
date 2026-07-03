import { useTranslation } from 'react-i18next';

/**
 * Returns a formatter that turns an ISO timestamp into a localised
 * "just now / 5m ago / 3 days ago" string. Uses the `time.*` namespace,
 * so the plural rules for each language are honoured by i18next.
 *
 * For anything older than 30 days we fall back to the browser's own
 * locale-aware date formatting (`Intl.DateTimeFormat`) so we don't have
 * to invent our own month names.
 */
export function useRelativeTime() {
  const { t, i18n } = useTranslation();

  return function relative(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const minutes = Math.round(diff / 60_000);
    if (minutes < 1) return t('time.justNow');
    if (minutes < 60) return t('time.minutesAgo', { count: minutes });
    const hours = Math.round(minutes / 60);
    if (hours < 24) return t('time.hoursAgo', { count: hours });
    const days = Math.round(hours / 24);
    if (days < 7) return t('time.daysAgo', { count: days });
    if (days < 30) return t('time.weeksAgo', { count: Math.round(days / 7) });
    if (days < 365) return t('time.monthsAgo', { count: Math.round(days / 30) });
    // For >1 year old, the exact date is more useful than "N years ago".
    return new Date(iso).toLocaleDateString(i18n.language);
  };
}
