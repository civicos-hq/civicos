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
const USER_KEY = 'civicos-admin-user';

export interface AdminSession {
  accessToken: string;
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
    return { accessToken: token, user: JSON.parse(userRaw) };
  } catch {
    return null;
  }
}

export function setSession(s: AdminSession) {
  localStorage.setItem(TOKEN_KEY, s.accessToken);
  localStorage.setItem(USER_KEY, JSON.stringify(s.user));
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
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

// When the token expires the whole admin session goes back to the login
// page — no auto-refresh yet. Admin sessions are short-lived by design;
// re-authenticating is a low-cost trust checkpoint.
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status;
    if (status === 401) {
      clearSession();
      const path = window.location.pathname;
      if (path !== '/login') {
        window.location.href = `/login?redirect=${encodeURIComponent(path)}`;
      }
    }
    return Promise.reject(error);
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
