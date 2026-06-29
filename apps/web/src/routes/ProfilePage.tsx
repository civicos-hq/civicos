import { useState, type FormEvent } from 'react';
import axios from 'axios';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Input } from '@civicos/ui';
import type { ApiResponse, Community, User } from '@civicos/types';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';

function useCommunity(id: string | undefined) {
  return useQuery({
    queryKey: ['community', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ community: Community }>>(`/api/v1/communities/${id}`);
      return res.data.data.community;
    },
  });
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[parts.length - 1]![0]!).toUpperCase();
}

function formatJoined(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

const ROLE_TONE: Record<string, string> = {
  CITIZEN: 'bg-slate-100 text-slate-700',
  REPRESENTATIVE: 'bg-amber-100 text-amber-700',
  GOVERNMENT_ADMIN: 'bg-sky-100 text-sky-700',
  PLATFORM_ADMIN: 'bg-civic-100 text-civic-700',
  NGO: 'bg-emerald-100 text-emerald-700',
  MODERATOR: 'bg-purple-100 text-purple-700',
};

export function ProfilePage() {
  const navigate = useNavigate();
  const meQuery = useMe();
  const me = meQuery.data;
  const communityQuery = useCommunity(me?.communityId);

  function signOut() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    navigate('/login');
  }

  if (meQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading your profile…</p>;
  }

  if (!me) {
    return (
      <p className="text-sm text-red-600">Could not load your profile. Try signing in again.</p>
    );
  }

  const roleTone = ROLE_TONE[me.role] ?? 'bg-slate-100 text-slate-700';
  const community = communityQuery.data;

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-civic-700 text-xl font-bold text-white">
            {initials(me.name)}
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Civic Profile
            </p>
            <h1 className="mt-1 text-2xl font-semibold text-slate-900">{me.name}</h1>
            <p className="text-sm text-slate-500">{me.email}</p>
          </div>
          <span
            className={`rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wide ${roleTone}`}
          >
            {me.role.replace('_', ' ')}
          </span>
        </div>
      </header>

      <div className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
        <AccountSection user={me} />

        <aside className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Community</h2>
          {communityQuery.isLoading ? (
            <p className="mt-3 text-sm text-slate-500">Loading…</p>
          ) : community ? (
            <div className="mt-3 space-y-2 text-sm text-slate-700">
              <p className="text-base font-semibold text-slate-900">{community.name}</p>
              <p className="text-slate-500">
                {community.lga}, {community.state}
              </p>
              {community.description && (
                <p className="pt-1 text-slate-600">{community.description}</p>
              )}
              <Link
                to="/community"
                className="mt-2 inline-block text-sm font-semibold text-civic-700 hover:text-civic-800"
              >
                Change community →
              </Link>
            </div>
          ) : (
            <div className="mt-3 space-y-2 text-sm text-slate-600">
              <p>You haven't joined a community yet.</p>
              <Link
                to="/community"
                className="inline-block text-sm font-semibold text-civic-700 hover:text-civic-800"
              >
                Find your community →
              </Link>
            </div>
          )}
        </aside>
      </div>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Session</h2>
        <p className="mt-2 text-sm text-slate-600">Sign out of CivicOS on this device.</p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={signOut}>
          Sign out
        </Button>
      </section>
    </section>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">{label}</dt>
      <dd className="mt-1 text-sm text-slate-900">{value}</dd>
    </div>
  );
}

function AccountSection({ user }: { user: User }) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);
  const [error, setError] = useState('');

  function startEdit() {
    setName(user.name);
    setEmail(user.email);
    setError('');
    setEditing(true);
  }

  function cancel() {
    setEditing(false);
    setError('');
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const payload: { name?: string; email?: string } = {};
      if (name.trim() !== user.name) payload.name = name.trim();
      if (email.trim() !== user.email) payload.email = email.trim();
      if (Object.keys(payload).length === 0) return user;
      const res = await api.patch<ApiResponse<{ user: User }>>('/api/v1/auth/me', payload);
      return res.data.data.user;
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated);
      setEditing(false);
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err)) {
        // Service responses are flat { code }; the gateway wraps as { error: { code } }.
        const data = err.response?.data;
        const code: string | undefined = data?.error?.code ?? data?.code;
        if (code === 'EMAIL_ALREADY_IN_USE') {
          setError('That email is already registered to another account.');
          return;
        }
        if (code === 'VALIDATION_ERROR') {
          setError('Please double-check the values you entered.');
          return;
        }
      }
      setError('Could not save changes. Please try again.');
    },
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    mutation.mutate();
  }

  if (!editing) {
    return (
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex items-start justify-between gap-3">
          <h2 className="text-lg font-semibold text-slate-900">Account</h2>
          <Button size="sm" variant="secondary" onClick={startEdit}>
            Edit
          </Button>
        </div>
        <dl className="mt-4 grid gap-3 sm:grid-cols-2">
          <Field label="Full name" value={user.name} />
          <Field label="Email" value={user.email} />
          <Field label="Role" value={user.role.replace('_', ' ')} />
          <Field label="Member since" value={formatJoined(user.createdAt)} />
        </dl>
      </section>
    );
  }

  const dirty = name.trim() !== user.name || email.trim() !== user.email;

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-slate-900">Edit account</h2>
      <form className="mt-4 grid gap-3 sm:grid-cols-2" onSubmit={submit}>
        <label className="text-sm text-slate-700">
          Full name
          <Input
            className="mt-1.5"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
            maxLength={100}
          />
        </label>
        <label className="text-sm text-slate-700">
          Email
          <Input
            className="mt-1.5"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>

        {error && <p className="sm:col-span-2 text-sm text-red-600">{error}</p>}

        <div className="sm:col-span-2 mt-2 flex items-center justify-end gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={cancel}>
            Cancel
          </Button>
          <Button type="submit" size="sm" loading={mutation.isPending} disabled={!dirty}>
            Save changes
          </Button>
        </div>
      </form>
    </section>
  );
}
