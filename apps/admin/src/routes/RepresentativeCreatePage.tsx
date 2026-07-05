import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ArrowLeft, ShieldAlert, UserPlus } from 'lucide-react';
import { apiGet, apiPost } from '../lib/api';

interface RepresentativeResponse {
  representative: { id: string; name: string };
}

interface CommunityLite {
  id: string;
  name: string;
  state: string;
  lga: string;
}

const POSITIONS = [
  'President',
  'Vice President',
  'Governor',
  'Deputy Governor',
  'Senator',
  'House of Representatives Member',
  'State House of Assembly Member',
  'LGA Chairman',
  'Councillor',
  'Traditional Ruler',
] as const;

export function RepresentativeCreatePage() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [title, setTitle] = useState('Hon.');
  const [position, setPosition] = useState('');
  const [constituency, setConstituency] = useState('');
  const [communityId, setCommunityId] = useState('');
  const [party, setParty] = useState('');
  const [bio, setBio] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [website, setWebsite] = useState('');
  const [error, setError] = useState<string | null>(null);

  const communitiesQuery = useQuery({
    queryKey: ['admin-communities-lite'],
    queryFn: () => apiGet<{ communities: CommunityLite[] }>('/api/v1/communities'),
  });
  const communities = communitiesQuery.data?.communities ?? [];

  const create = useMutation({
    mutationFn: () =>
      apiPost<RepresentativeResponse>('/api/v1/representatives', {
        name: name.trim(),
        title: title.trim(),
        position: position.trim(),
        constituency: constituency.trim(),
        communityId,
        party: party.trim() || undefined,
        bio: bio.trim() || undefined,
        avatarUrl: avatarUrl.trim() || undefined,
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
        website: website.trim() || undefined,
      }),
    onSuccess: (data) => {
      navigate(`/representatives?created=${data.representative.id}`);
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string; code?: string } } }).response
        ?.data;
      setError(msg?.message ?? 'Failed to create representative.');
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!name.trim() || !title.trim() || !position || !constituency.trim() || !communityId) {
      setError('Name, title, position, constituency, and community are required.');
      return;
    }
    create.mutate();
  }

  return (
    <>
      <Link
        to="/representatives"
        className="inline-flex items-center gap-1 text-sm text-civic-700 hover:underline mb-3"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to Representatives
      </Link>

      <header className="admin-page-header">
        <p className="admin-page-eyebrow">Section — Representatives · New</p>
        <h1 className="admin-page-title">Add a representative</h1>
        <p className="admin-page-sub">
          Once created, this representative is discoverable on the citizen app under{' '}
          <span className="mono">/representatives</span>. Citizens in the linked community will see
          them at the top of their list.
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

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-title">
                Title <span className="text-red-600">*</span>
              </label>
              <input
                id="r-title"
                className="admin-table-search"
                placeholder="Hon., Sen., HRH…"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
              />
            </div>
            <div className="flex flex-col gap-1 md:col-span-2">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-name">
                Full name <span className="text-red-600">*</span>
              </label>
              <input
                id="r-name"
                className="admin-table-search"
                placeholder="e.g. Adebayo Ogunlesi"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                minLength={2}
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-position">
                Position <span className="text-red-600">*</span>
              </label>
              <select
                id="r-position"
                className="admin-table-search"
                value={position}
                onChange={(e) => setPosition(e.target.value)}
                required
              >
                <option value="">Select a position…</option>
                {POSITIONS.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-party">
                Party <span className="text-xs text-slate-500">(optional)</span>
              </label>
              <input
                id="r-party"
                className="admin-table-search mono"
                placeholder="APC, PDP, LP, NNPP…"
                value={party}
                onChange={(e) => setParty(e.target.value)}
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-constituency">
                Constituency <span className="text-red-600">*</span>
              </label>
              <input
                id="r-constituency"
                className="admin-table-search"
                placeholder="e.g. Lagos Central Senatorial District"
                value={constituency}
                onChange={(e) => setConstituency(e.target.value)}
                required
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-community">
                Home community <span className="text-red-600">*</span>
              </label>
              <select
                id="r-community"
                className="admin-table-search"
                value={communityId}
                onChange={(e) => setCommunityId(e.target.value)}
                required
                disabled={communitiesQuery.isLoading}
              >
                <option value="">
                  {communitiesQuery.isLoading ? 'Loading communities…' : 'Select a community…'}
                </option>
                {communities.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name} · {c.state} / {c.lga}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-500">
                Citizens in this community see this rep pinned to the top of their list.
              </p>
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="r-bio">
              Biography <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <textarea
              id="r-bio"
              rows={4}
              className="admin-table-search"
              placeholder="Public-facing biography. Shown on the representative detail page."
              value={bio}
              onChange={(e) => setBio(e.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-slate-700" htmlFor="r-avatar">
              Avatar URL <span className="text-xs text-slate-500">(optional)</span>
            </label>
            <input
              id="r-avatar"
              type="url"
              className="admin-table-search mono"
              placeholder="https://…"
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
            />
          </div>

          <h3
            className="text-xs font-semibold text-slate-500 mono pt-2"
            style={{ letterSpacing: '0.14em' }}
          >
            OFFICIAL CONTACT
          </h3>
          <p className="text-xs text-slate-500 -mt-2">
            Public. Shown to citizens as contact channels. Leave blank if not published.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-email">
                Email
              </label>
              <input
                id="r-email"
                type="email"
                className="admin-table-search mono"
                placeholder="office@example.gov.ng"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-phone">
                Phone
              </label>
              <input
                id="r-phone"
                type="tel"
                className="admin-table-search mono"
                placeholder="+234 xxx xxx xxxx"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-sm font-medium text-slate-700" htmlFor="r-website">
                Website
              </label>
              <input
                id="r-website"
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
            <Link to="/representatives" className="admin-btn admin-btn-secondary">
              Cancel
            </Link>
            <button
              type="submit"
              className="admin-btn admin-btn-primary"
              disabled={create.isPending}
            >
              <UserPlus className="h-4 w-4" aria-hidden="true" />
              {create.isPending ? 'Creating…' : 'Create representative'}
            </button>
          </div>
        </div>
      </form>
    </>
  );
}
