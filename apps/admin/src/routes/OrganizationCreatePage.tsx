import { useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { ArrowLeft, Building2, ShieldAlert } from 'lucide-react';
import { apiPost } from '../lib/api';
import { NIGERIAN_STATES } from '../data/nigeria';

interface OrganizationResponse {
  organization: { id: string; name: string; slug: string };
}

const KINDS = ['GOVERNMENT', 'AGENCY', 'NGO', 'UTILITY', 'OTHER'] as const;
const JURISDICTIONS = ['NATIONAL', 'STATE', 'LGA', 'COMMUNITY'] as const;

function toSlug(input: string): string {
  return input
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');
}

export function OrganizationCreatePage() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [slugTouched, setSlugTouched] = useState(false);
  const [kind, setKind] = useState<(typeof KINDS)[number]>('GOVERNMENT');
  const [jurisdiction, setJurisdiction] = useState<(typeof JURISDICTIONS)[number]>('LGA');
  const [state, setState] = useState('');
  const [lga, setLga] = useState('');
  const [description, setDescription] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [website, setWebsite] = useState('');
  const [logoUrl, setLogoUrl] = useState('');
  const [error, setError] = useState<string | null>(null);

  const needsState =
    jurisdiction === 'STATE' || jurisdiction === 'LGA' || jurisdiction === 'COMMUNITY';
  const needsLga = jurisdiction === 'LGA' || jurisdiction === 'COMMUNITY';
  const lgaOptions = useMemo(
    () => NIGERIAN_STATES.find((s) => s.name === state)?.lgas ?? [],
    [state],
  );

  const create = useMutation({
    mutationFn: () =>
      apiPost<OrganizationResponse>('/api/v1/organizations', {
        name: name.trim(),
        slug: slug.trim() || toSlug(name),
        kind,
        jurisdiction,
        state: needsState && state ? state : undefined,
        lga: needsLga && lga ? lga : undefined,
        description: description.trim() || undefined,
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
        website: website.trim() || undefined,
        logoUrl: logoUrl.trim() || undefined,
      }),
    onSuccess: (data) => {
      navigate(`/organizations/${data.organization.id}`);
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string; code?: string } } }).response
        ?.data;
      setError(msg?.message ?? 'Failed to create organization.');
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!name.trim()) {
      setError('Name is required.');
      return;
    }
    if (needsState && !state) {
      setError(`This jurisdiction (${jurisdiction}) requires a state.`);
      return;
    }
    if (needsLga && !lga) {
      setError(`This jurisdiction (${jurisdiction}) requires an LGA.`);
      return;
    }
    create.mutate();
  }

  return (
    <>
      <Link
        to="/organizations"
        className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to Organizations
      </Link>

      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Organizations · New</p>
        <h1 className="admin-page-title">Create an organization</h1>
        <p className="admin-page-sub">
          Ministries, LGAs, agencies, utilities, NGOs. Appears in the citizen app under{' '}
          <span className="mono">/organizations</span>. You can grant the verified badge later from
          the org detail page.
        </p>
      </header>

      <form
        onSubmit={handleSubmit}
        className="admin-table-shell"
        style={{ padding: '1.5rem', maxWidth: '720px' }}
      >
        <div className="space-y-4">
          <h3
            className="text-xs font-semibold text-slate-500 mono"
            style={{ letterSpacing: '0.14em' }}
          >
            IDENTITY
          </h3>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="o-name">
              Organization name <span className="text-red-600">*</span>
            </label>
            <input
              id="o-name"
              className="admin-table-search"
              placeholder="e.g. Lagos State Waste Management Authority"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                if (!slugTouched) setSlug(toSlug(e.target.value));
              }}
              required
              minLength={2}
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="o-slug">
              Slug <span className="text-red-600">*</span>{' '}
              <span className="text-xs text-slate-500">(auto-generated; edit if needed)</span>
            </label>
            <input
              id="o-slug"
              className="admin-table-search mono"
              placeholder="lagos-state-waste-management-authority"
              value={slug}
              onChange={(e) => {
                setSlug(toSlug(e.target.value));
                setSlugTouched(true);
              }}
              required
              minLength={2}
              pattern="^[a-z0-9-]+$"
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="o-kind">
                Kind <span className="text-red-600">*</span>
              </label>
              <select
                id="o-kind"
                className="admin-table-search"
                value={kind}
                onChange={(e) => setKind(e.target.value as (typeof KINDS)[number])}
                required
              >
                {KINDS.map((k) => (
                  <option key={k} value={k}>
                    {k}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="o-jur">
                Jurisdiction <span className="text-red-600">*</span>
              </label>
              <select
                id="o-jur"
                className="admin-table-search"
                value={jurisdiction}
                onChange={(e) => {
                  setJurisdiction(e.target.value as (typeof JURISDICTIONS)[number]);
                  if (e.target.value === 'NATIONAL') {
                    setState('');
                    setLga('');
                  }
                }}
                required
              >
                {JURISDICTIONS.map((j) => (
                  <option key={j} value={j}>
                    {j}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-500">
                {jurisdiction === 'NATIONAL'
                  ? 'Federal-level organization. No state required.'
                  : jurisdiction === 'STATE'
                    ? 'State-level organization. Choose a state.'
                    : jurisdiction === 'LGA'
                      ? 'LGA-level organization. Choose state and LGA.'
                      : 'Community-level organization. Choose state and LGA.'}
              </p>
            </div>
          </div>

          {needsState && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <label className="text-sm font-medium text-slate-700" htmlFor="o-state">
                  State <span className="text-red-600">*</span>
                </label>
                <select
                  id="o-state"
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
              {needsLga && (
                <div className="flex flex-col gap-1">
                  <label className="text-sm font-medium text-slate-700" htmlFor="o-lga">
                    LGA <span className="text-red-600">*</span>
                  </label>
                  <select
                    id="o-lga"
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
              )}
            </div>
          )}

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="o-desc">
              Description <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <textarea
              id="o-desc"
              rows={3}
              className="admin-table-search"
              placeholder="What this organization does. Shown on its citizen-facing page."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="o-logo">
              Logo URL <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <input
              id="o-logo"
              type="url"
              className="admin-table-search mono"
              placeholder="https://…"
              value={logoUrl}
              onChange={(e) => setLogoUrl(e.target.value)}
            />
          </div>

          <h3
            className="text-xs font-semibold text-slate-500 mono pt-2"
            style={{ letterSpacing: '0.14em' }}
          >
            OFFICIAL CONTACT
          </h3>
          <p className="text-xs text-slate-500 -mt-2">
            Public. Citizens use these to reach the organization directly.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="o-email">
                Email
              </label>
              <input
                id="o-email"
                type="email"
                className="admin-table-search mono"
                placeholder="info@example.gov.ng"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="o-phone">
                Phone
              </label>
              <input
                id="o-phone"
                type="tel"
                className="admin-table-search mono"
                placeholder="+234 xxx xxx xxxx"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="o-website">
                Website
              </label>
              <input
                id="o-website"
                type="url"
                className="admin-table-search mono"
                placeholder="https://…"
                value={website}
                onChange={(e) => setWebsite(e.target.value)}
              />
            </div>
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
            <Link to="/organizations" className="admin-btn admin-btn-secondary">
              Cancel
            </Link>
            <button
              type="submit"
              className="admin-btn admin-btn-primary"
              disabled={create.isPending}
            >
              <Building2 className="h-4 w-4" aria-hidden="true" />
              {create.isPending ? 'Creating…' : 'Create organization'}
            </button>
          </div>
        </div>
      </form>
    </>
  );
}
