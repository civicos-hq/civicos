import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { Building2, ClipboardList, Flag, LayoutDashboard, LogOut, Users } from 'lucide-react';
import { clearSession, getSession } from '../lib/api';

export function AdminShell() {
  const session = getSession();
  const navigate = useNavigate();

  function handleSignOut() {
    clearSession();
    navigate('/login', { replace: true });
  }

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar" aria-label="Admin navigation">
        <div className="admin-brand">
          <p className="admin-brand-title">CivicOS</p>
          <p className="admin-brand-app">Admin Console</p>
        </div>

        <NavLinkItem to="/" end icon={LayoutDashboard} label="Overview" />

        <p className="admin-nav-section">People</p>
        <NavLinkItem to="/users" icon={Users} label="Users" />
        <NavLinkItem to="/organizations" icon={Building2} label="Organizations" />

        <p className="admin-nav-section">Trust</p>
        <NavLinkItem to="/flags" icon={Flag} label="Moderation queue" />
        <NavLinkItem to="/audit" icon={ClipboardList} label="Audit log" />
      </aside>

      <div className="admin-main">
        <header className="admin-topbar">
          <div className="admin-topbar-actor">
            <span className="admin-topbar-role">{session?.user.role ?? '—'}</span>
            <span className="mono text-slate-600">{session?.user.email}</span>
          </div>
          <button
            type="button"
            onClick={handleSignOut}
            className="admin-btn admin-btn-secondary admin-btn-sm"
          >
            <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
            Sign out
          </button>
        </header>

        <main className="admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

function NavLinkItem({
  to,
  label,
  icon: Icon,
  end,
}: {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) => `admin-nav-link ${isActive ? 'active' : ''}`}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      {label}
    </NavLink>
  );
}
