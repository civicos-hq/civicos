import { AlertCircle, Bell, FileText, Home, LogOut, User, Users } from 'lucide-react';
import { NavLink } from 'react-router-dom';
import { useUnreadCount } from '../../hooks/useNotifications';

const navItems = [
  { to: '/community', label: 'Community', icon: Home },
  { to: '/issues', label: 'Issues', icon: AlertCircle },
  { to: '/petitions', label: 'Petitions', icon: FileText },
  { to: '/representatives', label: 'Representatives', icon: Users },
  { to: '/notifications', label: 'Notifications', icon: Bell },
];

export function Sidebar() {
  const { data: unread = 0 } = useUnreadCount();
  function logout() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    window.location.href = '/login';
  }

  return (
    <aside className="dashboard-sidebar">
      <div className="dashboard-brand">
        <span className="brand-mark" aria-hidden="true">
          C
        </span>
        <div>
          <p className="brand-title">CivicOS</p>
          <p className="brand-subtitle">Public Action Console</p>
        </div>
      </div>

      <nav className="dashboard-nav" aria-label="Primary">
        {navItems.map(({ to, label, icon: Icon }) => {
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
              <span className="dashboard-link-label flex-1">{label}</span>
              {showBadge && (
                <span className="dashboard-link-badge inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-rose-500 px-1.5 text-[10px] font-bold leading-none text-white">
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
          Profile
        </NavLink>

        <button type="button" className="dashboard-link dashboard-link-idle" onClick={logout}>
          <LogOut className="h-4 w-4" aria-hidden="true" />
          Sign out
        </button>
      </div>
    </aside>
  );
}
