import { NavLink } from 'react-router-dom';
import { Home, AlertCircle, FileText, Users, Bell, MessageSquare, User } from 'lucide-react';

const navItems = [
  { to: '/community', label: 'Community', icon: Home },
  { to: '/issues', label: 'Issues', icon: AlertCircle },
  { to: '/petitions', label: 'Petitions', icon: FileText },
  { to: '/representatives', label: 'Representatives', icon: Users },
  { to: '/notifications', label: 'Notifications', icon: Bell },
  { to: '/ai', label: 'AI Assistant', icon: MessageSquare },
];

export function Sidebar() {
  return (
    <aside className="flex w-60 flex-col border-r border-gray-100 bg-white">
      <div className="flex h-16 items-center border-b border-gray-100 px-6">
        <span className="text-xl font-bold text-civic-600">CivicOS</span>
      </div>

      <nav className="flex-1 space-y-1 px-3 py-4">
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors ${
                isActive
                  ? 'bg-civic-50 font-medium text-civic-700'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
              }`
            }
          >
            <Icon className="h-4 w-4" />
            {label}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-gray-100 px-3 py-4">
        <NavLink
          to="/profile"
          className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-gray-600 hover:bg-gray-50"
        >
          <User className="h-4 w-4" />
          Profile
        </NavLink>
      </div>
    </aside>
  );
}
