import axios, { type AxiosRequestConfig } from 'axios';

// VITE_API_URL is shared with the citizen app. When Render's Blueprint
// injects it via `fromService: property: host`, the value is a bare
// hostname like "civicos-gateway.onrender.com" — prepend https:// so
// axios treats it as a real URL. Local dev keeps the http://localhost
// default.
function resolveApiBase(): string {
  const raw = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:3000';
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw;
  return `https://${raw}`;
}
export const API_BASE = resolveApiBase();

export const api = axios.create({
  baseURL: API_BASE,
  timeout: 15_000,
});

const TOKEN_KEY = 'civicos-admin-token';
const REFRESH_KEY = 'civicos-admin-refresh';
const USER_KEY = 'civicos-admin-user';

export interface AdminSession {
  accessToken: string;
  refreshToken?: string;
  user: {
    id: string;
    email: string;
    name: string;
    role: string;
    emailVerified: boolean;
  };
}

export function getSession(): AdminSession | null {
  const token = localStorage.getItem(TOKEN_KEY);
  const userRaw = localStorage.getItem(USER_KEY);
  if (!token || !userRaw) return null;
  try {
    return {
      accessToken: token,
      refreshToken: localStorage.getItem(REFRESH_KEY) ?? undefined,
      user: JSON.parse(userRaw),
    };
  } catch {
    return null;
  }
}

export function setSession(s: AdminSession) {
  localStorage.setItem(TOKEN_KEY, s.accessToken);
  if (s.refreshToken) localStorage.setItem(REFRESH_KEY, s.refreshToken);
  localStorage.setItem(USER_KEY, JSON.stringify(s.user));
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(USER_KEY);
}

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) {
    config.headers = config.headers ?? {};
    config.headers['Authorization'] = `Bearer ${token}`;
  }
  return config;
});

// Access tokens live 15 minutes; without rotation the console bounced
// admins to the login page mid-task every quarter hour. On a 401 we spend
// the stored refresh token once (coalesced across concurrent 401s) and
// retry the original request; only a failed rotation ends the session.
let refreshInFlight: Promise<string | null> | null = null;

async function performRefresh(): Promise<string | null> {
  const refreshToken = localStorage.getItem(REFRESH_KEY);
  if (!refreshToken) return null;
  try {
    // Bare axios: the instance interceptors must not attach the expired
    // access token or recurse into this handler.
    const res = await axios.post(`${API_BASE}/api/v1/auth/refresh`, { refreshToken });
    const tokens = res.data?.data?.tokens as
      { accessToken?: string; refreshToken?: string } | undefined;
    if (!tokens?.accessToken) return null;
    localStorage.setItem(TOKEN_KEY, tokens.accessToken);
    if (tokens.refreshToken) localStorage.setItem(REFRESH_KEY, tokens.refreshToken);
    return tokens.accessToken;
  } catch {
    return null;
  }
}

function forceSignOut() {
  clearSession();
  const path = window.location.pathname;
  if (path !== '/login') {
    window.location.href = `/login?redirect=${encodeURIComponent(path)}`;
  }
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status;
    const original = error.config as (AxiosRequestConfig & { _retried?: boolean }) | undefined;
    if (status !== 401 || !original || original._retried) {
      if (status === 401) forceSignOut();
      return Promise.reject(error);
    }

    original._retried = true;
    if (!refreshInFlight) {
      refreshInFlight = performRefresh().finally(() => (refreshInFlight = null));
    }
    const newAccess = await refreshInFlight;
    if (!newAccess) {
      forceSignOut();
      return Promise.reject(error);
    }
    original.headers = { ...original.headers, Authorization: `Bearer ${newAccess}` };
    return api.request(original);
  },
);

// Convenience typed getter — returns .data.data unwrapped from the
// {success, data} envelope the Go services use.
export async function apiGet<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await api.get<{ success: boolean; data: T }>(url, config);
  return res.data.data;
}

export async function apiPost<T>(
  url: string,
  body?: unknown,
  config?: AxiosRequestConfig,
): Promise<T> {
  const res = await api.post<{ success: boolean; data: T }>(url, body, config);
  return res.data.data;
}

export async function apiPatch<T>(
  url: string,
  body?: unknown,
  config?: AxiosRequestConfig,
): Promise<T> {
  const res = await api.patch<{ success: boolean; data: T }>(url, body, config);
  return res.data.data;
}
