import axios from 'axios';
import type { ApiError } from '@civicos/types';

/**
 * Error classification for every failed request in the app.
 *
 * The problem this solves: `getApiError()` only ever sees errors that came
 * back WITH an HTTP response. When the request never reached a server at all
 * — the phone lost signal, DNS failed, the service is down, CORS rejected it
 * — `error.response` is undefined, so every call site fell through to a
 * generic "Could not load X. Try again in a moment."
 *
 * That message is actively misleading when the user is offline: it points at
 * CivicOS when the fix is on their end, and "try again in a moment" is wrong
 * advice when the answer is "reconnect first".
 *
 * So we classify the CAUSE, and each surface pairs it with its own context.
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
  | 'upstreamDown' // 502/503/504, or gateway's UPSTREAM_ERROR
  | 'serverError' // other 5xx
  | 'unknown';

export interface ClassifiedError {
  kind: ApiErrorKind;
  /** Backend error code, e.g. ISSUE_NOT_FOUND. Absent when there was no response. */
  code?: string;
  /** Raw server message. Shown only when it is safe and useful to surface. */
  serverMessage?: string;
  status?: number;
  /** Seconds until a rate-limited action may be retried. */
  retryAfter?: number;
  /** Whether offering a "Try again" button actually makes sense. */
  retryable: boolean;
}

/** 5xx codes that mean "a service behind the gateway is down", not "bad request". */
const UPSTREAM_STATUSES = new Set([502, 503, 504]);

/**
 * Sentinel for a query TanStack has PAUSED rather than failed.
 *
 * Paused is a third state that no page had a branch for: `status` stays
 * 'pending' and `fetchStatus` becomes 'paused', so `isLoading` and `isError`
 * are BOTH false and the render falls through to whatever comes last —
 * usually the empty state. The user is then told "No projects here yet" when
 * the truth is "we never managed to ask".
 *
 * queryFailure() converts that state into this sentinel so it travels the
 * same path as a real error and gets the offline message it deserves.
 */
export const QUERY_PAUSED = Symbol('civicos:query-paused');

/**
 * The error a surface should render for a query, or null when it is fine.
 *
 * Covers the two ways a query can be unusable: it errored, or it is paused
 * with nothing cached to show. Call sites gate BOTH their error panel and
 * their empty state on this, so an empty state can only ever mean "we asked
 * and the answer was nothing".
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
 * Classifies any thrown value into a cause.
 *
 * Ordering matters: the offline check comes first because when the device is
 * offline EVERY other signal (timeout, unreachable) is a symptom rather than
 * the cause, and telling someone "the server is unavailable" when their wifi
 * is off sends them debugging the wrong thing.
 */
export function classifyError(error: unknown): ClassifiedError {
  // A paused query is one the client declined to send. TanStack pauses when
  // it believes there is no connection — but it has been observed pausing
  // while its own onlineManager reports online, so we must NOT assume the
  // user is offline just because a query is paused. Ask the browser instead,
  // and if the browser says we are connected, report the honest thing: the
  // request didn't get through, cause unknown.
  if (error === QUERY_PAUSED) {
    const offline = typeof navigator !== 'undefined' && navigator.onLine === false;
    return { kind: offline ? 'offline' : 'unreachable', retryable: true };
  }

  // navigator.onLine has a known weakness — it reports true for a connected
  // interface with no actual route — but a `false` reading is reliable, and
  // false is the only case we act on here.
  if (typeof navigator !== 'undefined' && navigator.onLine === false) {
    return { kind: 'offline', retryable: true };
  }

  if (!axios.isAxiosError(error)) {
    return { kind: 'unknown', retryable: true };
  }

  if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
    return { kind: 'timeout', retryable: true };
  }

  // No response object at all: the request never completed a round trip.
  if (!error.response) {
    return { kind: 'unreachable', retryable: true };
  }

  const status = error.response.status;
  const body = error.response.data as Partial<ApiError> | undefined;
  const code = typeof body?.code === 'string' ? body.code : undefined;
  const serverMessage = typeof body?.message === 'string' ? body.message : undefined;
  const base = { code, serverMessage, status };

  if (status === 429) {
    return { ...base, kind: 'rateLimited', retryAfter: retryAfterSeconds(error), retryable: true };
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

function retryAfterSeconds(error: unknown): number | undefined {
  if (!axios.isAxiosError(error)) return undefined;
  const header = (error.response?.headers as Record<string, string> | undefined)?.['retry-after'];
  const fromBody = (error.response?.data as { data?: { retryAfter?: number } } | undefined)?.data
    ?.retryAfter;
  const seconds = Number(header ?? fromBody ?? 0);
  return Number.isFinite(seconds) && seconds > 0 ? Math.ceil(seconds) : undefined;
}

/**
 * Whether a failure is worth automatically retrying.
 *
 * Used by the QueryClient: retrying a 403 or a validation error just burns
 * requests and delays the moment the user is told what is actually wrong.
 */
export function isRetryableError(error: unknown): boolean {
  return classifyError(error).retryable;
}

/**
 * Backend error codes that carry a specific, user-meaningful meaning we would
 * rather show than the generic per-kind copy. Anything not listed falls back
 * to the kind-level message — an unknown code must never leak raw to the UI.
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
