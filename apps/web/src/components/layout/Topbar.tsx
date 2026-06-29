import { Bell, Search, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useUnreadCount } from '../../hooks/useNotifications';

export function Topbar() {
  const { data: unread = 0 } = useUnreadCount();
  const hasUnread = unread > 0;
  const label = unread > 99 ? '99+' : String(unread);

  return (
    <header className="dashboard-topbar">
      <div className="relative max-w-xl flex-1">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
        <input
          type="search"
          placeholder="Search issues, petitions, representatives…"
          className="dashboard-search"
        />
      </div>

      <div className="ml-auto hidden items-center gap-3 rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600 shadow-sm md:flex">
        <Sparkles className="h-3.5 w-3.5 text-amber-500" />
        Participation Pulse: Rising
      </div>

      <div className="ml-auto flex items-center gap-3 md:ml-0">
        <Link
          to="/notifications"
          className="dashboard-icon-btn relative"
          aria-label={hasUnread ? `Open notifications (${unread} unread)` : 'Open notifications'}
        >
          <Bell className="h-5 w-5" />
          {hasUnread && (
            <span className="absolute -right-1 -top-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold leading-none text-white">
              {label}
            </span>
          )}
        </Link>

        <div className="dashboard-avatar" aria-hidden="true">
          CO
        </div>
      </div>
    </header>
  );
}
