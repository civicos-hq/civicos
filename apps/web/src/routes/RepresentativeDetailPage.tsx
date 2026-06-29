import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Mail, Phone, Globe } from 'lucide-react';
import type { ApiResponse, Community, Representative } from '@civicos/types';
import { api } from '../lib/api';
import { useFollowedReps } from '../hooks/useFollowedReps';
import { Avatar, FollowButton } from './RepresentativesPage';
import { CommentsSection } from '../components/civic/CommentsSection';

function useRepresentative(id: string) {
  return useQuery({
    queryKey: ['representative', id],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ representative: Representative }>>(
        `/api/v1/representatives/${id}`,
      );
      return res.data.data.representative;
    },
    enabled: Boolean(id),
  });
}

function useCommunities() {
  return useQuery({
    queryKey: ['communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });
}

export function RepresentativeDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const repQuery = useRepresentative(id);
  const communitiesQuery = useCommunities();
  const followsQuery = useFollowedReps();

  if (repQuery.isLoading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }
  if (repQuery.isError || !repQuery.data) {
    return (
      <section className="space-y-4">
        <Link
          to="/representatives"
          className="text-sm font-semibold text-civic-700 hover:underline"
        >
          ← Back to representatives
        </Link>
        <p className="text-sm text-red-600">Couldn't load this representative.</p>
      </section>
    );
  }

  const rep = repQuery.data;
  const community = communitiesQuery.data?.find((c) => c.id === rep.communityId);
  const isFollowing = followsQuery.data?.has(rep.id) ?? false;

  return (
    <section className="space-y-6">
      <Link to="/representatives" className="text-sm font-semibold text-civic-700 hover:underline">
        ← Back to representatives
      </Link>

      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-5">
          <Avatar name={rep.name} src={rep.avatarUrl} size={96} />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Representative
            </p>
            <h1 className="mt-1 text-3xl font-semibold text-slate-900">
              {rep.title} {rep.name}
            </h1>
            <p className="mt-1 text-sm text-slate-600">
              {rep.position} · {rep.constituency}
              {community ? ` · ${community.name}` : ''}
            </p>
            {rep.party && (
              <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-civic-700">
                {rep.party}
              </p>
            )}
          </div>
          <FollowButton representativeId={rep.id} isFollowing={isFollowing} />
        </div>
      </header>

      <div className="grid gap-4 md:grid-cols-2">
        <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Response rate
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">{rep.responseRate}%</p>
          <p className="mt-1 text-sm text-slate-500">Of citizen messages answered.</p>
        </article>
        <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            Followers
          </p>
          <p className="mt-2 text-3xl font-semibold text-slate-900">
            {rep.followerCount.toLocaleString()}
          </p>
          <p className="mt-1 text-sm text-slate-500">Citizens subscribed to updates.</p>
        </article>
      </div>

      {rep.bio && (
        <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">About</h2>
          <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{rep.bio}</p>
        </article>
      )}

      <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Contact</h2>
        <p className="mt-1 text-sm text-slate-500">
          For private matters. For public questions, post below — replies are visible to everyone.
        </p>
        {rep.email || rep.phone || rep.website ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {rep.email && (
              <ContactLink
                href={`mailto:${rep.email}`}
                icon={<Mail className="h-4 w-4" />}
                label={rep.email}
              />
            )}
            {rep.phone && (
              <ContactLink
                href={`tel:${rep.phone.replace(/\s+/g, '')}`}
                icon={<Phone className="h-4 w-4" />}
                label={rep.phone}
              />
            )}
            {rep.website && (
              <ContactLink
                href={rep.website}
                icon={<Globe className="h-4 w-4" />}
                label={rep.website.replace(/^https?:\/\//, '')}
                external
              />
            )}
          </div>
        ) : (
          <p className="mt-4 text-sm italic text-slate-500">
            No direct contact details on file. Post a question below to reach this representative
            publicly.
          </p>
        )}
      </article>

      <CommentsSection entityType="representatives" entityId={rep.id} />
    </section>
  );
}

function ContactLink({
  href,
  icon,
  label,
  external,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  external?: boolean;
}) {
  return (
    <a
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
      className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-700 hover:border-civic-300 hover:bg-civic-50 hover:text-civic-700"
    >
      <span className="text-slate-400">{icon}</span>
      {label}
    </a>
  );
}
