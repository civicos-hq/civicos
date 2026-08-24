/**
 * Shared classification of failed HTTP requests.
 *
 * Lives here, not in either app, because the citizen app and the admin
 * console hit the same gateway and fail in exactly the same ways — while
 * needing completely different copy for it (web is translated into five
 * locales and speaks to citizens; admin is English-only and speaks to
 * operators). So the CAUSE is shared and the WORDS are not.
 *
 * Deliberately dependency-free — no axios import — so this package stays
 * installable from anywhere. Axios errors are detected by their own
 * `isAxiosError` marker, which is all `axios.isAxiosError()` checks.
 */

export type ApiErrorKind =
  | 'offline' // the device itself reports no network
  | 'unreachable' // request sent, no response — service down, DNS, CORS
  | 'timeout' // took too long
  | 'rateLimited' // 429
  | 'unauthorized' // 401 — session expired
  | 'forbidden' // 403 — signed in, not allowed
  | 'notFound' // 404
  | 'validation' // 400 / 422 — the input was rejected
  | 'conflict' // 409 — state changed underneath us
  | 'tooLarge' // 413
  | 'upstreamDown' // 502/503/504, or the gateway's UPSTREAM_ERROR
  | 'serverError' // other 5xx
  | 'unknown';

export interface ClassifiedError {
  kind: ApiErrorKind;
  /** Backend error code, e.g. ISSUE_NOT_FOUND. Absent when there was no response. */
  code?: string;
  /** Raw server message. Surface only where it is safe and useful. */
  serverMessage?: string;
  status?: number;
  /** Seconds until a rate-limited action may be retried. */
  retryAfter?: number;
  /** Whether offering a "Try again" control actually makes sense. */
  retryable: boolean;
}

/** 5xx codes that mean "a service behind the gateway is down", not "bad request". */
const UPSTREAM_STATUSES = new Set([502, 503, 504]);

/**
 * Sentinel for a query the client PAUSED rather than failed.
 *
 * Paused is a third state no page had a branch for: `status` stays 'pending'
 * and `fetchStatus` becomes 'paused', so `isLoading` and `isError` are BOTH
 * false and the render falls through to whatever comes last — usually the
 * empty state. The user is then told "nothing here yet" when the truth is
 * "we never managed to ask".
 */
export const QUERY_PAUSED = Symbol.for('civicos:query-paused');

interface HttpErrorLike {
  isAxiosError?: boolean;
  code?: string;
  response?: {
    status: number;
    data?: unknown;
    headers?: Record<string, string>;
  };
}

function asHttpError(error: unknown): HttpErrorLike | null {
  if (!error || typeof error !== 'object') return null;
  const e = error as HttpErrorLike;
  return e.isAxiosError === true ? e : null;
}

function isOffline(): boolean {
  return typeof navigator !== 'undefined' && navigator.onLine === false;
}

/**
 * Classifies any thrown value into a cause.
 *
 * Ordering matters: offline is checked first because when the device has no
 * connection every other signal (timeout, unreachable) is a symptom rather
 * than the cause, and telling someone "the server is unavailable" when their
 * wifi is off sends them debugging the wrong thing.
 */
export function classifyError(error: unknown): ClassifiedError {
  // A paused query is one the client declined to send. Clients pause when
  // they believe there is no connection — but this has been observed while
  // the client's own online check reports connected, so do NOT assume the
  // user is offline just because a query paused. Ask the browser, and if it
  // says we are connected, report the honest thing: it didn't get through.
  if (error === QUERY_PAUSED) {
    return { kind: isOffline() ? 'offline' : 'unreachable', retryable: true };
  }

  // navigator.onLine has a known weakness — it reports true for a connected
  // interface with no actual route — but a `false` reading is reliable, and
  // false is the only case acted on here.
  if (isOffline()) {
    return { kind: 'offline', retryable: true };
  }

  const err = asHttpError(error);
  if (!err) {
    return { kind: 'unknown', retryable: true };
  }

  if (err.code === 'ECONNABORTED' || err.code === 'ETIMEDOUT') {
    return { kind: 'timeout', retryable: true };
  }

  // No response object at all: the request never completed a round trip.
  if (!err.response) {
    return { kind: 'unreachable', retryable: true };
  }

  const status = err.response.status;
  const body = err.response.data as Partial<ApiErrorBody> | undefined;
  const code = typeof body?.code === 'string' ? body.code : undefined;
  const serverMessage = typeof body?.message === 'string' ? body.message : undefined;
  const base = { code, serverMessage, status };

  if (status === 429) {
    return { ...base, kind: 'rateLimited', retryAfter: retryAfterSeconds(err), retryable: true };
  }
  if (status === 401) return { ...base, kind: 'unauthorized', retryable: false };
  if (status === 403) return { ...base, kind: 'forbidden', retryable: false };
  if (status === 404) return { ...base, kind: 'notFound', retryable: false };
  if (status === 409) return { ...base, kind: 'conflict', retryable: true };
  if (status === 413) return { ...base, kind: 'tooLarge', retryable: false };
  if (status === 400 || status === 422) return { ...base, kind: 'validation', retryable: false };

  if (UPSTREAM_STATUSES.has(status) || code === 'UPSTREAM_ERROR') {
    return { ...base, kind: 'upstreamDown', retryable: true };
  }
  if (status >= 500) return { ...base, kind: 'serverError', retryable: true };

  return { ...base, kind: 'unknown', retryable: true };
}

/** The gateway's error envelope, as far as classification needs it. */
interface ApiErrorBody {
  code: string;
  message: string;
}

function retryAfterSeconds(err: HttpErrorLike): number | undefined {
  const header = err.response?.headers?.['retry-after'];
  const fromBody = (err.response?.data as { data?: { retryAfter?: number } } | undefined)?.data
    ?.retryAfter;
  const seconds = Number(header ?? fromBody ?? 0);
  return Number.isFinite(seconds) && seconds > 0 ? Math.ceil(seconds) : undefined;
}

/** Whether a failure is worth automatically retrying. */
export function isRetryableError(error: unknown): boolean {
  return classifyError(error).retryable;
}

/**
 * The error a surface should render for a query, or null when it is fine.
 *
 * Covers both ways a query becomes unusable: it errored, or it is paused
 * with nothing cached to show. Gate BOTH the error panel and the empty state
 * on this, so an empty state can only ever mean "we asked and the answer was
 * nothing".
 */
export function queryFailure(query: {
  isError: boolean;
  error: unknown;
  fetchStatus: string;
  data: unknown;
}): unknown | null {
  if (query.isError) return query.error;
  if (query.fetchStatus === 'paused' && query.data === undefined) return QUERY_PAUSED;
  return null;
}

/**
 * Backend error codes with a specific, user-meaningful meaning worth showing
 * instead of the generic per-kind copy. Anything not listed falls back to the
 * kind — an unknown code must never leak raw to a screen.
 */
export const KNOWN_ERROR_CODES = [
  'EMAIL_NOT_VERIFIED',
  'COMMUNITY_MEMBERSHIP_REQUIRED',
  'PRIMARY_COMMUNITY_MISMATCH',
  'FILE_TOO_LARGE',
  'INVALID_FILE_TYPE',
  'TOO_MANY_VIDEOS',
  'INVALID_VIDEO',
  'ISSUE_NOT_FOUND',
  'PETITION_NOT_FOUND',
  'ALREADY_SIGNED',
  'REPRESENTATIVE_UNCLAIMED',
  'VALIDATION_ERROR',
  'UPSTREAM_ERROR',
] as const;

export type KnownErrorCode = (typeof KNOWN_ERROR_CODES)[number];

export function isKnownErrorCode(code: string | undefined): code is KnownErrorCode {
  return !!code && (KNOWN_ERROR_CODES as readonly string[]).includes(code);
}
