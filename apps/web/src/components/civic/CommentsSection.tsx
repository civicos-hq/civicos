import { useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import type { ApiResponse, IssueComment, PetitionComment, UserRole } from '@civicos/types';
import { api } from '../../lib/api';

type EntityType = 'issues' | 'petitions';
type AnyComment = IssueComment | PetitionComment;

const ROLE_LABEL: Partial<Record<UserRole, string>> = {
  CITIZEN: 'Citizen',
  REPRESENTATIVE: 'Representative',
  GOVERNMENT_ADMIN: 'Government Admin',
  NGO: 'NGO',
  MODERATOR: 'Moderator',
  PLATFORM_ADMIN: 'Platform Admin',
} as Partial<Record<UserRole, string>>;

function formatTimeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diff / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

function initials(name: string): string {
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((p) => p[0]?.toUpperCase())
      .join('') || '?'
  );
}

export function CommentsSection({
  entityType,
  entityId,
}: {
  entityType: EntityType;
  entityId: string;
}) {
  const queryClient = useQueryClient();
  const [content, setContent] = useState('');
  const [error, setError] = useState('');

  const commentsQuery = useQuery({
    queryKey: ['comments', entityType, entityId],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ comments: AnyComment[] }>>(
        `/api/v1/${entityType}/${entityId}/comments`,
      );
      return res.data.data.comments;
    },
  });

  const addMutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/${entityType}/${entityId}/comments`, { content });
    },
    onSuccess: () => {
      setContent('');
      queryClient.invalidateQueries({ queryKey: ['comments', entityType, entityId] });
    },
    onError: () => setError('Could not post your comment. Try again.'),
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    if (!content.trim()) return;
    addMutation.mutate();
  }

  const comments = commentsQuery.data ?? [];

  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-slate-900">
        Discussion <span className="text-sm font-normal text-slate-500">({comments.length})</span>
      </h2>

      <form onSubmit={handleSubmit} className="mt-4 space-y-3">
        <textarea
          className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
          rows={3}
          maxLength={2000}
          placeholder="Add to the discussion…"
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex items-center justify-between">
          <p className="text-xs text-slate-500">{content.length}/2000</p>
          <Button
            type="submit"
            size="sm"
            loading={addMutation.isPending}
            disabled={!content.trim()}
          >
            Post comment
          </Button>
        </div>
      </form>

      <div className="mt-6 space-y-4">
        {commentsQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading…</p>
        ) : comments.length === 0 ? (
          <p className="text-sm text-slate-500">No comments yet — start the conversation.</p>
        ) : (
          comments.map((c) => (
            <div
              key={c.id}
              className={`flex gap-3 rounded-xl border p-3 ${c.isOfficialResponse ? 'border-civic-200 bg-civic-50/40' : 'border-slate-200 bg-slate-50/60'}`}
            >
              <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-civic-100 text-xs font-semibold text-civic-700 ring-1 ring-civic-200">
                {initials(c.authorName)}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-2">
                  <p className="text-sm font-semibold text-slate-900">{c.authorName}</p>
                  <span className="text-xs text-slate-500">
                    {ROLE_LABEL[c.authorRole] ?? c.authorRole}
                  </span>
                  {c.isOfficialResponse && (
                    <span className="rounded-full bg-civic-700 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white">
                      Official response
                    </span>
                  )}
                  <span className="ml-auto text-xs text-slate-400">
                    {formatTimeAgo(c.createdAt)}
                  </span>
                </div>
                <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">{c.content}</p>
              </div>
            </div>
          ))
        )}
      </div>
    </article>
  );
}
