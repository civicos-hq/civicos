import {
  AlertCircle,
  Briefcase,
  Building2,
  Compass,
  FileText,
  LogOut,
  Megaphone,
  MessageSquare,
  User,
  Users,
  HandCoins,
  Home,
  Wrench,
  type LucideIcon,
} from 'lucide-react';
import { Link, NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { UserRole } from '@civicos/types';
import { useUnreadCount } from '../../hooks/useNotifications';
import { useMyOrganizations } from '../../hooks/useConsultations';
import { useMe } from '../../hooks/useMe';
import { signOut } from '../../lib/api';

/**
 * The sidebar nav is split into named groups so the user can scan to
 * the kind of work they want to do, not to a flat alphabetical list of
 * 13 destinations. Discover and Community stay at the top as the
 * orientation surfaces; the rest cluster into:
 *
 *   - Speak up     — what the citizen posts (issues, petitions, fundraising)
 *   - Connect      — who they reach (representatives, organizations)
 *   - Stay informed — what they're told about (consultations, announcements,
 *                     projects)
 *   - Account      — notifications, profile, sign out
 *
 * Each group renders with a small uppercase label above the links, in
 * muted ink, so the structure is visible without competing with the
 * active link. Previously the whole list was 13 equal-weight links
 * stacked top-to-bottom, which read as a long menu rather than four
 * short ones.
 */
type NavLinkDef = { to: string; i18n: string; icon: LucideIcon };

const navGroups: Array<{ label: string; items: NavLinkDef[] }> = [
  {
    label: 'sidebar.speakUp',
    items: [
      { to: '/issues', i18n: 'sidebar.issues', icon: AlertCircle },
      { to: '/petitions', i18n: 'sidebar.petitions', icon: FileText },
      { to: '/campaigns', i18n: 'sidebar.campaigns', icon: HandCoins },
    ],
  },
  {
    label: 'sidebar.connect',
    items: [
      { to: '/representatives', i18n: 'sidebar.representatives', icon: Users },
      { to: '/organizations', i18n: 'sidebar.organizations', icon: Building2 },
    ],
  },
  {
    label: 'sidebar.stayInformed',
    items: [
      { to: '/consultations', i18n: 'sidebar.consultations', icon: MessageSquare },
      { to: '/announcements', i18n: 'sidebar.announcements', icon: Megaphone },
      { to: '/projects', i18n: 'sidebar.projects', icon: Wrench },
    ],
  },
];

// Orientation surfaces — Discover and Community stay at the top, outside
// the groups, because they're the "where am I / where do I belong"
// surfaces rather than task destinations.
const orientationItems: NavLinkDef[] = [
  { to: '/discover', i18n: 'sidebar.discover', icon: Compass },
  { to: '/community', i18n: 'sidebar.community', icon: Home },
];

/**
 * `onNavigate` fires whenever the user activates a nav link inside
 * the sidebar. DashboardLayout uses it to close the mobile drawer
 * immediately on tap (the pathname-change effect also closes it as
 * a safety, but this gives instant visual feedback).
 */
export function Sidebar({ onNavigate }: { onNavigate?: () => void } = {}) {
  const { t } = useTranslation();
  const { data: unread = 0 } = useUnreadCount();
  const { data: myOrgs = [] } = useMyOrganizations();
  const { data: me } = useMe();
  // The "My organization" link only shows when the caller can actually
  // act on behalf of an org. Ordinary citizens don't see it at all.
  //
  // Representatives are included even before they own anything: their
  // constituency office is created on first visit to /org, and without the
  // link there is no way to reach the page that creates it.
  const canActAsOrg =
    myOrgs.some((m) => m.membership.role === 'OWNER' || m.membership.role === 'ADMIN') ||
    me?.role === UserRole.REPRESENTATIVE;
  async function logout() {
    // Best-effort server revoke of the refresh family, then wipe local state.
    // signOut() itself never throws so the sign-out flow is idempotent.
    await signOut();
    window.location.href = '/login';
  }

  return (
    <aside className="dashboard-sidebar" aria-label={t('common.mainNav')}>
      {/* Brand block doubles as a shortcut back to the public landing
          page — standard "click the logo to go home" pattern that
          users reach for reflexively. */}
      <Link
        to="/"
        className="dashboard-brand"
        aria-label={t('common.backToHome')}
        onClick={onNavigate}
      >
        <span className="brand-mark" aria-hidden="true">
          <img src="/civicos-mark.png" alt="" />
        </span>
        <div>
          <p className="brand-title">CivicOS</p>
          <p className="brand-subtitle">{t('nav.brandSubtitle')}</p>
        </div>
      </Link>

      <nav className="dashboard-nav" aria-label={t('common.mainNav')}>
        {/* Orientation surfaces — Discover and Community first, with no
            group label, because they're the "where am I" pages rather
            than tasks. */}
        {orientationItems.map(({ to, i18n: i18nKey, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={onNavigate}
            className={({ isActive }) =>
              `dashboard-link ${isActive ? 'dashboard-link-active' : 'dashboard-link-idle'}`
            }
          >
            <Icon className="h-4 w-4" aria-hidden="true" />
            <span className="dashboard-link-label flex-1">{t(i18nKey)}</span>
          </NavLink>
        ))}

        {canActAsOrg && (
          <NavLink
            to="/org"
            onClick={onNavigate}
            className={({ isActive }) =>
              `dashboard-link ${isActive ? 'dashboard-link-active' : 'dashboard-link-idle'}`
            }
          >
            <Briefcase className="h-4 w-4" aria-hidden="true" />
            <span className="dashboard-link-label flex-1">
              {/* A representative's is their constituency office, not an
                  "organization" — same destination, honest label. */}
              {me?.role === UserRole.REPRESENTATIVE ? t('sidebar.myOffice') : t('sidebar.myOrg')}
            </span>
          </NavLink>
        )}

        {/* Grouped nav. Each group has a small uppercase label that sets
            context for the links underneath without competing with the
            active link's color or weight. */}
        {navGroups.map((group) => (
          <div key={group.label} className="dashboard-nav-group">
            <p className="dashboard-nav-group-label">{t(group.label)}</p>
            {group.items.map(({ to, i18n: i18nKey, icon: Icon }) => {
              const showBadge = to === '/notifications' && unread > 0;
              return (
                <NavLink
                  key={to}
                  to={to}
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    `dashboard-link ${isActive ? 'dashboard-link-active' : 'dashboard-link-idle'}`
                  }
                >
                  <Icon className="h-4 w-4" aria-hidden="true" />
                  <span className="dashboard-link-label flex-1">{t(i18nKey)}</span>
                  {showBadge && (
                    <span className="dashboard-link-badge inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 text-[10px] font-bold leading-none text-white">
                      {unread > 99 ? '99+' : unread}
                    </span>
                  )}
                </NavLink>
              );
            })}
          </div>
        ))}
      </nav>

      <div className="dashboard-footer-nav">
        <p className="dashboard-nav-group-label">{t('sidebar.account')}</p>
        <NavLink to="/profile" onClick={onNavigate} className="dashboard-link dashboard-link-idle">
          <User className="h-4 w-4" aria-hidden="true" />
          <span className="dashboard-link-label flex-1">{t('sidebar.profile')}</span>
        </NavLink>

        <button type="button" className="dashboard-link dashboard-link-idle" onClick={logout}>
          <LogOut className="h-4 w-4" aria-hidden="true" />
          <span className="dashboard-link-label flex-1">{t('sidebar.signOut')}</span>
        </button>
      </div>
    </aside>
  );
}
