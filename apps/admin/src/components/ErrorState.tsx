import { AlertTriangle, RefreshCw, WifiOff } from 'lucide-react';
import { describeError } from '../lib/errorMessage';

/**
 * Standard rendering for a failed request in the admin console.
 *
 * Replaces bare one-liners like `<p className="admin-error">Could not load
 * analytics.</p>`, which said something failed but never why — the same
 * sentence appeared whether the gateway was down, the session had expired,
 * or the operator's role was insufficient, each needing a different response.
 *
 * The reference code is shown deliberately: an operator debugging a live
 * outage wants the machine-readable code to grep logs with, not reassurance.
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
  compact?: boolean;
}) {
  const { title, detail, code, retryable, kind } = describeError(error, context);
  const Icon = kind === 'offline' || kind === 'unreachable' ? WifiOff : AlertTriangle;

  if (compact) {
    return (
      <p role="alert" className="admin-error">
        {context && <strong>{context} </strong>}
        {detail}
        {retryable && onRetry && (
          <button type="button" onClick={onRetry} className="admin-error-retry-inline">
            Retry
          </button>
        )}
      </p>
    );
  }

  return (
    <div role="alert" className="admin-error-panel">
      <Icon className="admin-error-panel-icon" aria-hidden="true" size={18} />
      <div>
        {context && <p className="admin-error-panel-context">{context}</p>}
        <p className="admin-error-panel-title">{title}</p>
        <p className="admin-error-panel-detail">{detail}</p>
        {retryable && onRetry && (
          <button type="button" onClick={onRetry} className="admin-btn admin-btn-secondary">
            <RefreshCw size={14} aria-hidden="true" /> Retry
          </button>
        )}
        {code && <p className="admin-error-panel-code">Reference: {code}</p>}
      </div>
    </div>
  );
}
