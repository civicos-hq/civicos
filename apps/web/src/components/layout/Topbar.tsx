import { Bell, Menu } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useUnreadCount } from '../../hooks/useNotifications';
import { useCommunitiesByID } from '../../hooks/useCommunities';
import { useMe } from '../../hooks/useMe';
import { api } from '../../lib/api';
import { SearchBar } from './SearchBar';
import { LanguageSwitcher } from '../LanguageSwitcher';
import { ThemeToggle } from '../ThemeToggle';
import { meInitials } from '../../lib/initials';

/**
 * `onOpenDrawer` is invoked when the hamburger button is tapped on
 * mobile. DashboardLayout owns the drawer-open state and passes this
 * setter down. The button itself only renders below the drawer
 * breakpoint (see `.dashboard-drawer-toggle` in index.css).
 */
export function Topbar({ onOpenDrawer }: { onOpenDrawer?: () => void } = {}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const meQuery = useMe();
  const membershipIds = (meQuery.data?.memberships ?? []).map((m) => m.communityId);
  const communitiesQuery = useCommunitiesByID(membershipIds);
  const { data: unread = 0 } = useUnreadCount();
  const hasUnread = unread > 0;
  const label = unread > 99 ? '99+' : String(unread);
  const activeCommunityId = meQuery.data?.activeCommunityId;
  // Already scoped to the user's memberships by the query itself — no
  // client-side filtering of a full community list any more.
  const joinedCommunities = communitiesQuery.data ?? [];

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
      {/* Hamburger — mobile-only. Below 860px the sidebar collapses
          into an off-canvas drawer; this button opens it. Hidden at
          desktop widths via CSS. */}
      <button
        type="button"
        className="dashboard-drawer-toggle"
        onClick={onOpenDrawer}
        aria-label={t('common.openMenu', 'Open menu')}
      >
        <Menu className="h-5 w-5" aria-hidden="true" />
      </button>

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
            pattern. Initials are derived from the user's name (up to
            two letters) so a real account shows "GN" for "Gino
            Osahon" rather than the hard-coded "CO" every prior
            version used. The aria-label spells out the destination
            for screen readers since the visible content is just the
            initials. Falls back to "?" if the name is missing so the
            circle never reads blank. */}
        <Link to="/profile" className="dashboard-avatar" aria-label={t('common.openProfile')}>
          {meInitials(meQuery.data?.name)}
        </Link>
      </div>
    </header>
  );
}
