import { useTranslation } from 'react-i18next';
import { AlertTriangle, RefreshCw, WifiOff } from 'lucide-react';
import { useErrorMessage } from '../hooks/useErrorMessage';

/**
 * The standard way to render a failed request.
 *
 * Replaces the old one-liner (`<p class="text-red-600">{t('…loadError')}</p>`),
 * which told the user something failed but never why — the same red sentence
 * appeared whether their wifi was off, their session had expired, or the
 * service was down, each of which needs a different response from them.
 *
 * Structure is deliberate:
 *   context  — what failed ("Could not load projects")
 *   title    — the cause ("You're offline")
 *   detail   — what to do about it
 *   retry    — only when retrying could plausibly work
 *   code     — small, last, for quoting in a bug report
 */
export function ErrorState({
  error,
  context,
  onRetry,
  compact,
}: {
  error: unknown;
  context?: string;
  onRetry?: () => void;
  /** Inline variant for use inside forms and cards. */
  compact?: boolean;
}) {
  const { t } = useTranslation();
  const describe = useErrorMessage();
  const { title, detail, code, retryable, kind } = describe(error, context);

  const Icon = kind === 'offline' || kind === 'unreachable' ? WifiOff : AlertTriangle;

  if (compact) {
    return (
      <div role="alert" className="flex items-start gap-2 text-sm text-rose-700 dark:text-rose-300">
        <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <div>
          {context && <span className="font-medium">{context} </span>}
          <span>{detail}</span>
          {retryable && onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="ml-2 underline underline-offset-2 hover:no-underline"
            >
              {t('errors.retry')}
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div
      role="alert"
      className="rounded-2xl border border-rose-200 bg-rose-50 p-4 md:p-6 dark:border-rose-500/30 dark:bg-rose-500/10"
    >
      <div className="flex items-start gap-3">
        <Icon
          className="mt-0.5 h-5 w-5 shrink-0 text-rose-600 dark:text-rose-400"
          aria-hidden="true"
        />
        <div className="min-w-0 flex-1">
          {context && (
            <p className="text-sm font-semibold text-rose-900 dark:text-rose-100">{context}</p>
          )}
          <p className="mt-0.5 text-sm font-medium text-rose-800 dark:text-rose-200">{title}</p>
          <p className="mt-1 text-sm text-rose-700 dark:text-rose-300">{detail}</p>

          {retryable && onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-rose-300 bg-white px-3 py-1.5 text-sm font-medium text-rose-800 transition hover:bg-rose-100 dark:border-rose-500/40 dark:bg-transparent dark:text-rose-200 dark:hover:bg-rose-500/10"
            >
              <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
              {t('errors.retry')}
            </button>
          )}

          {code && (
            <p className="mt-3 text-xs text-rose-600/80 dark:text-rose-400/80">
              {t('errors.codeLabel', { code })}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
