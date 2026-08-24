import { classifyError, isKnownErrorCode, type ApiErrorKind } from '@civicos/types';

/**
 * Error copy for the admin console.
 *
 * Classification is shared with the citizen app (@civicos/types); the wording
 * is not. This audience is operators, not citizens, so the copy names the
 * component that failed and says what to check, rather than reassuring. There
 * is no i18n in this app — these are the strings.
 */

interface Copy {
  title: string;
  detail: string;
}

const KIND_COPY: Record<ApiErrorKind, Copy> = {
  offline: {
    title: "You're offline",
    detail: 'This machine has no network connection. The console will work again once it returns.',
  },
  unreachable: {
    title: 'Cannot reach the API gateway',
    detail:
      'The request never got a response. Check that the gateway is running and that VITE_API_URL points at it.',
  },
  timeout: {
    title: 'The request timed out',
    detail:
      'The gateway did not answer within 15 seconds. It may be overloaded or blocked on a slow upstream query.',
  },
  rateLimited: {
    title: 'Rate limited',
    detail: 'This action hit the gateway rate limit. Wait a moment before retrying.',
  },
  unauthorized: {
    title: 'Your admin session expired',
    detail: 'Sign in again to continue. No changes you already saved are affected.',
  },
  forbidden: {
    title: 'Your role does not permit this',
    detail:
      'This action needs a higher-privileged role than your account holds. Ask a platform admin.',
  },
  notFound: {
    title: 'Not found',
    detail: 'This record no longer exists. It may have been deleted since the page loaded.',
  },
  validation: {
    title: 'The server rejected these details',
    detail: 'Check the highlighted fields and submit again.',
  },
  conflict: {
    title: 'This record changed while you were editing',
    detail: 'Someone else updated it. Reload to see the current state before retrying.',
  },
  tooLarge: {
    title: 'That upload is too large',
    detail: 'The file exceeded the size limit. Try a smaller one.',
  },
  upstreamDown: {
    title: 'A backend service is down',
    detail:
      'The gateway is up but the service behind this page is not responding. Check the service health panel on the Overview page.',
  },
  serverError: {
    title: 'The server errored',
    detail: 'This is a fault in CivicOS. Check the service logs; quote the reference code below.',
  },
  unknown: {
    title: 'Something went wrong',
    detail: 'The request failed for an unrecognised reason. Check the browser console.',
  },
};

/**
 * Recognised backend codes worth naming precisely. Unlisted codes fall back
 * to the per-kind copy so an unknown code never reaches the screen raw.
 */
const CODE_COPY: Record<string, string> = {
  EMAIL_NOT_VERIFIED: 'This account has not verified its email address yet.',
  VALIDATION_ERROR: 'The server rejected these details. Check the fields and submit again.',
  UPSTREAM_ERROR:
    'The service behind this page is not responding. Check the service health panel on the Overview page.',
};

export interface DescribedError {
  title: string;
  detail: string;
  /** What the operator was doing, supplied by the call site. */
  context?: string;
  code?: string;
  retryable: boolean;
  kind: ApiErrorKind;
}

/**
 * Turns any thrown value into copy naming the cause and the next step.
 *
 * Two parts by design: the CONTEXT ("Could not resolve this flag") comes from
 * the call site, which is the only thing that knows what was attempted; the
 * CAUSE comes from the error.
 */
export function describeError(error: unknown, context?: string): DescribedError {
  const classified = classifyError(error);
  const kindCopy = KIND_COPY[classified.kind];
  const codeCopy = isKnownErrorCode(classified.code)
    ? CODE_COPY[classified.code]
    : classified.code
      ? CODE_COPY[classified.code]
      : undefined;

  // Precedence, most specific first:
  //   1. curated copy for a recognised code
  //   2. the server's own message — kept because these call sites already
  //      showed it and an operator debugging a live problem wants the
  //      backend's actual words, not a paraphrase
  //   3. the per-kind copy, which is the only thing available when there was
  //      no response at all (offline, gateway down, timeout) — precisely the
  //      cases that previously fell through to a generic string
  const detail =
    classified.kind === 'rateLimited' && classified.retryAfter
      ? `This action hit the gateway rate limit. Try again in ${classified.retryAfter} seconds.`
      : (codeCopy ?? classified.serverMessage ?? kindCopy.detail);

  return {
    title: kindCopy.title,
    detail,
    context,
    code: classified.code,
    retryable: classified.retryable,
    kind: classified.kind,
  };
}

/** Flattened to one string, for the many call sites holding an error string. */
export function errorText(error: unknown, context?: string): string {
  const { context: ctx, detail } = describeError(error, context);
  return ctx ? `${ctx} ${detail}` : detail;
}
