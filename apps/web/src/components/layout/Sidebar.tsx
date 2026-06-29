import { AlertCircle, Bell, FileText, Home, LogOut, User, Users } from 'lucide-react';
import { NavLink } from 'react-router-dom';

const navItems = [
  { to: '/community', label: 'Community', icon: Home },
  { to: '/issues', label: 'Issues', icon: AlertCircle },
  { to: '/petitions', label: 'Petitions', icon: FileText },
  { to: '/representatives', label: 'Representatives', icon: Users },
  { to: '/notifications', label: 'Notifications', icon: Bell },
];

export function Sidebar() {
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
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `dashboard-link ${isActive ? 'dashboard-link-active' : 'dashboard-link-idle'}`
            }
          >
            <Icon className="h-4 w-4" aria-hidden="true" />
            {label}
          </NavLink>
        ))}
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
