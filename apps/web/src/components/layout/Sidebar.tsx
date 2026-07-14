import {
  AlertCircle,
  Bell,
  Briefcase,
  Building2,
  Compass,
  FileText,
  Home,
  LogOut,
  Megaphone,
  MessageSquare,
  User,
  Users,
  Wrench,
} from 'lucide-react';
import { Link, NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useUnreadCount } from '../../hooks/useNotifications';
import { useMyOrganizations } from '../../hooks/useConsultations';
import { signOut } from '../../lib/api';

const navItems = [
  { to: '/discover', i18n: 'sidebar.discover', icon: Compass },
  { to: '/community', i18n: 'sidebar.community', icon: Home },
  { to: '/issues', i18n: 'sidebar.issues', icon: AlertCircle },
  { to: '/petitions', i18n: 'sidebar.petitions', icon: FileText },
  { to: '/representatives', i18n: 'sidebar.representatives', icon: Users },
  { to: '/organizations', i18n: 'sidebar.organizations', icon: Building2 },
  { to: '/consultations', i18n: 'sidebar.consultations', icon: MessageSquare },
  { to: '/announcements', i18n: 'sidebar.announcements', icon: Megaphone },
  { to: '/projects', i18n: 'sidebar.projects', icon: Wrench },
  { to: '/notifications', i18n: 'sidebar.notifications', icon: Bell },
];

export function Sidebar() {
  const { t } = useTranslation();
  const { data: unread = 0 } = useUnreadCount();
  const { data: myOrgs = [] } = useMyOrganizations();
  // The "My organization" link only shows when the caller can actually
  // act on behalf of an org. Ordinary citizens don't see it at all.
  const canActAsOrg = myOrgs.some(
    (m) => m.membership.role === 'OWNER' || m.membership.role === 'ADMIN',
  );
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
      <Link to="/" className="dashboard-brand" aria-label={t('common.backToHome')}>
        <span className="brand-mark" aria-hidden="true">
          <img src="/civicos-mark.png" alt="" />
        </span>
        <div>
          <p className="brand-title">CivicOS</p>
          <p className="brand-subtitle">{t('nav.brandSubtitle')}</p>
        </div>
      </Link>

      <nav className="dashboard-nav" aria-label={t('common.mainNav')}>
        {canActAsOrg && (
          <NavLink
            to="/org"
            className={({ isActive }) =>
              `dashboard-link ${isActive ? 'dashboard-link-active' : 'dashboard-link-idle'}`
            }
          >
            <Briefcase className="h-4 w-4" aria-hidden="true" />
            <span className="dashboard-link-label flex-1">{t('sidebar.myOrg')}</span>
          </NavLink>
        )}
        {navItems.map(({ to, i18n: i18nKey, icon: Icon }) => {
          const showBadge = to === '/notifications' && unread > 0;
          return (
            <NavLink
              key={to}
              to={to}
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
      </nav>

      <div className="dashboard-footer-nav">
        <NavLink to="/profile" className="dashboard-link dashboard-link-idle">
          <User className="h-4 w-4" />
          {t('sidebar.profile')}
        </NavLink>

        <button type="button" className="dashboard-link dashboard-link-idle" onClick={logout}>
          <LogOut className="h-4 w-4" aria-hidden="true" />
          {t('sidebar.signOut')}
        </button>
      </div>
    </aside>
  );
}
