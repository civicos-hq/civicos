import axios from 'axios';
import type { ApiResponse, ApiError } from '@civicos/types';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? 'http://localhost:3000',
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

// Attach JWT on every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// On 401 → redirect to login
api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('accessToken');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  },
);

export type { ApiResponse, ApiError };

const apiBase = import.meta.env.VITE_API_URL ?? 'http://localhost:3000';

export function uploadUrl(filenameOrUrl: string): string {
  if (/^https?:\/\//i.test(filenameOrUrl)) return filenameOrUrl;
  return `${apiBase}/api/v1/uploads/${filenameOrUrl}`;
}

export async function uploadImage(file: File): Promise<string> {
  const form = new FormData();
  form.append('file', file);
  const res = await api.post<ApiResponse<{ filename: string }>>('/api/v1/uploads', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return res.data.data.filename;
}
