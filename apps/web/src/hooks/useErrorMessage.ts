import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { classifyError, isKnownErrorCode, type ClassifiedError } from '../lib/apiError';

export interface ErrorDescription {
  /** One short line naming the CAUSE. "You're offline", not "Something went wrong". */
  title: string;
  /** What it means and what to do about it. */
  detail: string;
  /** Caller-supplied context: which action failed. Rendered above the cause. */
  context?: string;
  /** Backend error code, shown small so a user can quote it in a bug report. */
  code?: string;
  retryable: boolean;
  kind: ClassifiedError['kind'];
}

/**
 * Turns any thrown error into copy that names the cause and the next step.
 *
 * Two-part message by design. The CONTEXT ("Could not post your comment")
 * comes from the call site, because only it knows what the user was doing.
 * The CAUSE ("You're offline — check your connection") comes from the error.
 * Showing only the first is the bug this fixes; showing only the second
 * leaves the user unsure what they lost.
 */
export function useErrorMessage() {
  const { t } = useTranslation();
  return useDescribe(t);
}

/**
 * Same classification, flattened to one string.
 *
 * For the many call sites that already hold an error *string* in state and
 * render it in a `<p>`. Converting those to hold an error *object* would be a
 * far larger change for the same result, so this meets them where they are —
 * they gain the cause without restructuring.
 */
export function useErrorText() {
  const { t } = useTranslation();
  const describe = useDescribe(t);
  return useCallback(
    (error: unknown, context?: string): string => {
      const { context: ctx, detail } = describe(error, context);
      return ctx ? `${ctx} ${detail}` : detail;
    },
    [describe],
  );
}

function useDescribe(t: ReturnType<typeof useTranslation>['t']) {
  return useCallback(
    (error: unknown, context?: string): ErrorDescription => {
      const classified = classifyError(error);
      const { kind, code, retryAfter } = classified;

      // A recognised backend code beats the generic per-kind copy — it can say
      // "verify your email first" where the kind can only say "not allowed".
      if (isKnownErrorCode(code)) {
        const key = `errors.codes.${code}`;
        const detail = t(key, { defaultValue: '' });
        if (detail) {
          return {
            title: t(`errors.kinds.${kind}.title`),
            detail,
            context,
            code,
            retryable: classified.retryable,
            kind,
          };
        }
      }

      return {
        title: t(`errors.kinds.${kind}.title`),
        detail:
          kind === 'rateLimited' && retryAfter
            ? t('errors.kinds.rateLimited.detailCountdown', { seconds: retryAfter })
            : t(`errors.kinds.${kind}.detail`),
        context,
        code,
        retryable: classified.retryable,
        kind,
      };
    },
    [t],
  );
}
