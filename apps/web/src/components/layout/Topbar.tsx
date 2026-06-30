import { Bell, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useUnreadCount } from '../../hooks/useNotifications';
import { SearchBar } from './SearchBar';
import { LanguageSwitcher } from '../LanguageSwitcher';

export function Topbar() {
  const { data: unread = 0 } = useUnreadCount();
  const hasUnread = unread > 0;
  const label = unread > 99 ? '99+' : String(unread);

  return (
    <header className="dashboard-topbar">
      <SearchBar />

      <div className="ml-auto hidden items-center gap-3 rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600 shadow-sm md:flex">
        <Sparkles className="h-3.5 w-3.5 text-amber-500" />
        Participation Pulse: Rising
      </div>

      <div className="ml-auto flex items-center gap-3 md:ml-0">
        <LanguageSwitcher />
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
