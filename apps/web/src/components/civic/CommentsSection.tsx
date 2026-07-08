import { useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import type {
  ApiResponse,
  IssueComment,
  PetitionComment,
  RepresentativeComment,
} from '@civicos/types';
import { api, getApiError } from '../../lib/api';
import { useEnumLabels } from '../../hooks/useEnumLabels';
import { useRelativeTime } from '../../hooks/useRelativeTime';
import { EmptyState } from '../EmptyState';
import { MessageSquare } from 'lucide-react';
import { ReportButton } from './ReportButton';
import type { ReportableType } from './ReportModal';

type EntityType = 'issues' | 'petitions' | 'representatives';
type AnyComment = IssueComment | PetitionComment | RepresentativeComment;

// Maps the CommentsSection entityType to the moderation flag content
// type. Used by the per-comment Report button.
const REPORTABLE_BY_ENTITY: Record<EntityType, ReportableType> = {
  issues: 'ISSUE_COMMENT',
  petitions: 'PETITION_COMMENT',
  representatives: 'REPRESENTATIVE_COMMENT',
};

const COMMENT_MAX = 2000;

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
  const { t } = useTranslation();
  const enums = useEnumLabels();
  const relative = useRelativeTime();
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
      setError('');
      queryClient.invalidateQueries({ queryKey: ['comments', entityType, entityId] });
    },
    onError: (err) => {
      const apiError = getApiError(err);
      setError(
        apiError?.code === 'EMAIL_NOT_VERIFIED'
          ? t('auth.verify.actionRequired')
          : apiError?.code === 'COMMUNITY_MEMBERSHIP_REQUIRED'
            ? t('comments.postMembershipRequired')
            : t('comments.postError'),
      );
    },
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
        {t('comments.heading')}{' '}
        <span className="text-sm font-normal text-slate-600">({comments.length})</span>
      </h2>

      <form onSubmit={handleSubmit} className="mt-4 space-y-3">
        <textarea
          className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
          rows={3}
          maxLength={COMMENT_MAX}
          placeholder={t('comments.placeholder')}
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex items-center justify-between">
          <p className="text-xs text-slate-600">
            {t('comments.charCount', { count: content.length, max: COMMENT_MAX })}
          </p>
          <Button
            type="submit"
            size="sm"
            loading={addMutation.isPending}
            disabled={!content.trim()}
          >
            {t('comments.post')}
          </Button>
        </div>
      </form>

      <div className="mt-6 space-y-4">
        {commentsQuery.isLoading ? (
          <p className="text-sm text-slate-600">{t('common.loading')}</p>
        ) : comments.length === 0 ? (
          <EmptyState icon={<MessageSquare className="h-5 w-5" />} title={t('comments.empty')} />
        ) : (
          comments.map((c) =>
            c.isHidden ? (
              <div
                key={c.id}
                className="flex items-center gap-2 rounded-xl border border-dashed border-slate-300 bg-slate-50/40 p-3 text-xs italic text-slate-500"
                data-hidden="true"
              >
                <MessageSquare className="h-3.5 w-3.5" aria-hidden="true" />
                <span>{t('comments.removed')}</span>
                <span className="ml-auto text-[10px] text-slate-400">{relative(c.createdAt)}</span>
              </div>
            ) : (
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
                    <span className="text-xs text-slate-600">{enums.userRole(c.authorRole)}</span>
                    {c.isOfficialResponse && (
                      <span className="rounded-full bg-civic-700 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white">
                        {t('comments.officialResponse')}
                      </span>
                    )}
                    <span className="ml-auto text-xs text-slate-400">{relative(c.createdAt)}</span>
                  </div>
                  <p className="mt-1 whitespace-pre-wrap text-sm text-slate-700">{c.content}</p>
                  {!c.isOfficialResponse && (
                    <div className="mt-2 flex justify-end">
                      <ReportButton
                        contentType={REPORTABLE_BY_ENTITY[entityType]}
                        contentId={c.id}
                      />
                    </div>
                  )}
                </div>
              </div>
            ),
          )
        )}
      </div>
    </article>
  );
}
