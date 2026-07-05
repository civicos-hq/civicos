import { useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { ArrowLeft, MapPin, ShieldAlert } from 'lucide-react';
import { apiPost } from '../lib/api';
import { NIGERIAN_STATES } from '../data/nigeria';

interface CommunityResponse {
  community: { id: string; name: string; slug: string };
}

function toSlug(input: string): string {
  return input
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');
}

export function CommunityCreatePage() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [slugTouched, setSlugTouched] = useState(false);
  const [state, setState] = useState('');
  const [lga, setLga] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState<string | null>(null);

  const lgaOptions = useMemo(
    () => NIGERIAN_STATES.find((s) => s.name === state)?.lgas ?? [],
    [state],
  );

  const create = useMutation({
    mutationFn: () =>
      apiPost<CommunityResponse>('/api/v1/communities', {
        name: name.trim(),
        slug: slug.trim() || toSlug(name),
        state,
        lga,
        description: description.trim() || undefined,
      }),
    onSuccess: (data) => {
      navigate(`/communities/${data.community.id}`);
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string; code?: string } } }).response
        ?.data;
      setError(msg?.message ?? 'Failed to create community.');
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!name.trim() || !state || !lga) {
      setError('Name, state, and LGA are required.');
      return;
    }
    create.mutate();
  }

  return (
    <>
      <Link
        to="/communities"
        className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to Communities
      </Link>

      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Communities · New</p>
        <h1 className="admin-page-title">Create a community</h1>
        <p className="admin-page-sub">
          Every citizen belongs to one community — a state + LGA pair. Add a new one here to make it
          selectable during onboarding on the citizen app.
        </p>
      </header>

      <form
        onSubmit={handleSubmit}
        className="admin-table-shell"
        style={{ padding: '1.5rem', maxWidth: '640px' }}
      >
        <div className="space-y-4">
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="c-name">
              Community name <span className="text-red-600">*</span>
            </label>
            <input
              id="c-name"
              className="admin-table-search"
              placeholder="e.g. Ikeja Central"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                if (!slugTouched) setSlug(toSlug(e.target.value));
              }}
              required
              minLength={2}
            />
            <p className="text-xs text-slate-500">
              Shown to citizens when they join and when they browse issues.
            </p>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="c-slug">
              Slug <span className="text-red-600">*</span>{' '}
              <span className="text-xs text-slate-500">(auto-generated; edit if needed)</span>
            </label>
            <input
              id="c-slug"
              className="admin-table-search mono"
              placeholder="ikeja-central"
              value={slug}
              onChange={(e) => {
                setSlug(toSlug(e.target.value));
                setSlugTouched(true);
              }}
              required
              minLength={2}
              pattern="^[a-z0-9-]+$"
            />
            <p className="text-xs text-slate-500">
              Used in URLs. Lowercase letters, numbers, and hyphens only.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="c-state">
                State <span className="text-red-600">*</span>
              </label>
              <select
                id="c-state"
                className="admin-table-search"
                value={state}
                onChange={(e) => {
                  setState(e.target.value);
                  setLga('');
                }}
                required
              >
                <option value="">Select a state…</option>
                {NIGERIAN_STATES.map((s) => (
                  <option key={s.code} value={s.name}>
                    {s.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="c-lga">
                LGA <span className="text-red-600">*</span>
              </label>
              <select
                id="c-lga"
                className="admin-table-search"
                value={lga}
                onChange={(e) => setLga(e.target.value)}
                required
                disabled={!state}
              >
                <option value="">{state ? 'Select an LGA…' : 'Pick a state first'}</option>
                {lgaOptions.map((l) => (
                  <option key={l} value={l}>
                    {l}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="c-desc">
              Description <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <textarea
              id="c-desc"
              rows={3}
              className="admin-table-search"
              placeholder="A short introduction for citizens joining this community."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          {error && (
            <div
              className="flex items-start gap-2 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900"
              role="alert"
            >
              <ShieldAlert className="h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <span>{error}</span>
            </div>
          )}

          <div className="flex justify-end gap-2">
            <Link to="/communities" className="admin-btn admin-btn-secondary">
              Cancel
            </Link>
            <button
              type="submit"
              className="admin-btn admin-btn-primary"
              disabled={create.isPending}
            >
              <MapPin className="h-4 w-4" aria-hidden="true" />
              {create.isPending ? 'Creating…' : 'Create community'}
            </button>
          </div>
        </div>
      </form>
    </>
  );
}
