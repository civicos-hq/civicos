import { Bell } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useUnreadCount } from '../../hooks/useNotifications';
import { useCommunities } from '../../hooks/useCommunities';
import { useMe } from '../../hooks/useMe';
import { api } from '../../lib/api';
import { SearchBar } from './SearchBar';
import { LanguageSwitcher } from '../LanguageSwitcher';
import { ThemeToggle } from '../ThemeToggle';

export function Topbar() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const meQuery = useMe();
  const communitiesQuery = useCommunities();
  const { data: unread = 0 } = useUnreadCount();
  const hasUnread = unread > 0;
  const label = unread > 99 ? '99+' : String(unread);
  const memberships = meQuery.data?.memberships ?? [];
  const activeCommunityId = meQuery.data?.activeCommunityId;
  const joinedCommunityIDs = new Set(memberships.map((membership) => membership.communityId));
  const joinedCommunities = (communitiesQuery.data ?? []).filter((community) =>
    joinedCommunityIDs.has(community.id),
  );

  const switchMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.patch('/api/v1/auth/me/active-community', { communityId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries();
    },
  });

  return (
    <header className="dashboard-topbar" aria-label={t('common.topbar')}>
      <SearchBar />

      <div className="ml-auto flex items-center gap-3">
        {joinedCommunities.length > 0 && (
          <label
            className="hidden items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium shadow-sm lg:flex"
            style={{
              borderColor: 'var(--dash-line)',
              background: 'var(--dash-surface)',
              color: 'var(--dash-muted)',
            }}
          >
            <span>{t('communityPage.activeSelect')}</span>
            <select
              value={activeCommunityId ?? ''}
              onChange={(e) => switchMutation.mutate(e.target.value)}
              disabled={switchMutation.isPending}
              className="bg-transparent text-sm font-semibold focus:outline-none"
              style={{ color: 'var(--dash-ink)' }}
            >
              {joinedCommunities.map((community) => (
                <option key={community.id} value={community.id}>
                  {community.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <ThemeToggle />
        <LanguageSwitcher />
        <Link
          to="/notifications"
          className="dashboard-icon-btn relative"
          aria-label={hasUnread ? `Open notifications (${unread} unread)` : 'Open notifications'}
        >
          <Bell className="h-5 w-5" />
          {hasUnread && (
            <span className="absolute -right-1 -top-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold leading-none text-white">
              {label}
            </span>
          )}
        </Link>

        {/* Avatar is the universal shortcut to Profile — users click
            here reflexively expecting account settings, so honour the
            pattern. Aria-label spells out the destination for screen
            readers since the visible content is just initials. */}
        <Link to="/profile" className="dashboard-avatar" aria-label={t('common.openProfile')}>
          CO
        </Link>
      </div>
    </header>
  );
}
