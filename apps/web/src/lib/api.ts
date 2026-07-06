import type { AxiosError } from 'axios';
import axios, { type InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse, ApiError } from '@civicos/types';

// Prepend https:// when VITE_API_URL is a bare host (as Render's
// Blueprint fromService.host property returns). Local dev keeps the
// http://localhost default.
function resolveApiBase(): string {
  const raw = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:3000';
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw;
  return `https://${raw}`;
}
const API_BASE = resolveApiBase();

export const api = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

// Attach JWT on every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Refresh-token rotation ------------------------------------------------
//
// On any 401 we try the refresh endpoint exactly once, then retry the
// original request. Concurrent 401s share a single in-flight refresh so
// we don't burn multiple rotation slots on one page load.
//
// If refresh itself fails, we treat the session as dead: wipe local
// tokens and bounce to /login. Same for endpoints that shouldn't recurse
// (e.g. /refresh itself, or /logout).

const NON_RETRIABLE_URLS = ['/api/v1/auth/refresh', '/api/v1/auth/logout', '/api/v1/auth/login'];

// The single in-flight refresh promise so parallel 401s coalesce. Exported
// via refreshAccessToken() so long-lived non-axios connections (SSE) can
// share the same coalescing — no risk of two callers both burning a rotation
// slot on the same page load.
let refreshInFlight: Promise<string | null> | null = null;

async function performRefresh(): Promise<string | null> {
  const refreshToken = localStorage.getItem('refreshToken');
  if (!refreshToken) return null;
  try {
    // Use a bare axios instance so the request-interceptor doesn't attach
    // the (probably-expired) access token, and so a failure here doesn't
    // trigger this same handler recursively.
    const res = await axios.post<
      ApiResponse<{ tokens?: { accessToken: string; refreshToken: string } }>
    >(
      `${api.defaults.baseURL}/api/v1/auth/refresh`,
      { refreshToken },
      { headers: { 'Content-Type': 'application/json' } },
    );
    const tokens = res.data?.data?.tokens;
    if (!tokens?.accessToken) return null;
    localStorage.setItem('accessToken', tokens.accessToken);
    if (tokens.refreshToken) localStorage.setItem('refreshToken', tokens.refreshToken);
    return tokens.accessToken;
  } catch {
    return null;
  }
}

/**
 * Force a refresh-token rotation. Returns the new access token, or null if
 * refresh isn't possible (no refresh token, or the server rejected). Callers
 * are coalesced — multiple concurrent invocations share the same in-flight
 * request.
 */
export function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) refreshInFlight = performRefresh().finally(() => (refreshInFlight = null));
  return refreshInFlight;
}

function forceSignOut() {
  localStorage.removeItem('accessToken');
  localStorage.removeItem('refreshToken');
  // Avoid redirect loops during initial anonymous requests.
  if (!location.pathname.startsWith('/login')) {
    window.location.href = '/login';
  }
}

/**
 * The rate-limit event a UI-side listener (RateLimitToast) subscribes to.
 * We dispatch on window rather than plumbing a callback through every axios
 * call — this keeps the toast decoupled from mutation code.
 */
export type RateLimitEvent = { retryAfter: number };
export const RATE_LIMIT_EVENT = 'civicos:ratelimited';

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as InternalAxiosRequestConfig & { _retried?: boolean };
    const status = error.response?.status;

    // 429s never auto-retry — retrying would just burn more of the budget
    // and delay the reset. Broadcast so a global toast can nag the user.
    if (status === 429) {
      const retryHeader = (error.response?.headers as Record<string, string> | undefined)?.[
        'retry-after'
      ];
      const bodyRetry = (error.response?.data as { data?: { retryAfter?: number } } | undefined)
        ?.data?.retryAfter;
      const retryAfter = Number(retryHeader ?? bodyRetry ?? 0) || 5;
      window.dispatchEvent(
        new CustomEvent<RateLimitEvent>(RATE_LIMIT_EVENT, { detail: { retryAfter } }),
      );
      return Promise.reject(error);
    }

    if (
      status !== 401 ||
      !original ||
      original._retried ||
      NON_RETRIABLE_URLS.some((u) => (original.url ?? '').includes(u))
    ) {
      // 401 on refresh itself, or a second 401 after we already retried,
      // means the session is truly dead.
      if (status === 401) forceSignOut();
      return Promise.reject(error);
    }

    original._retried = true;
    const newAccess = await refreshAccessToken();
    if (!newAccess) {
      forceSignOut();
      return Promise.reject(error);
    }

    original.headers = original.headers ?? {};
    original.headers.Authorization = `Bearer ${newAccess}`;
    return api.request(original);
  },
);

/**
 * Best-effort revoke of the current refresh-token family server-side, then
 * clear local tokens. Called from the sidebar "Sign out" button. Never
 * throws — a failed revoke shouldn't prevent the user from signing out
 * locally.
 */
export async function signOut(): Promise<void> {
  const refreshToken = localStorage.getItem('refreshToken');
  try {
    if (refreshToken) {
      await axios.post(
        `${api.defaults.baseURL}/api/v1/auth/logout`,
        { refreshToken },
        { headers: { 'Content-Type': 'application/json' } },
      );
    }
  } catch {
    // ignore — we still wipe local state below
  }
  localStorage.removeItem('accessToken');
  localStorage.removeItem('refreshToken');
}

export type { ApiResponse, ApiError };

export function uploadUrl(filenameOrUrl: string): string {
  if (/^https?:\/\//i.test(filenameOrUrl)) return filenameOrUrl;
  return `${API_BASE}/api/v1/uploads/${filenameOrUrl}`;
}

export async function uploadImage(file: File): Promise<string> {
  const form = new FormData();
  form.append('file', file);
  const res = await api.post<ApiResponse<{ filename: string }>>('/api/v1/uploads', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return res.data.data.filename;
}
