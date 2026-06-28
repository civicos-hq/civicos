import { Bell, Search } from 'lucide-react';
import { Link } from 'react-router-dom';

export function Topbar() {
  return (
    <header className="flex h-16 items-center gap-4 border-b border-gray-100 bg-white px-6">
      <div className="relative max-w-md flex-1">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <input
          type="search"
          placeholder="Search issues, petitions, representatives…"
          className="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-civic-500"
        />
      </div>

      <div className="ml-auto flex items-center gap-3">
        <Link
          to="/notifications"
          className="relative rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-50 hover:text-gray-600"
        >
          <Bell className="h-5 w-5" />
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />
        </Link>

        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-civic-600 text-sm font-semibold text-white">
          G
        </div>
      </div>
    </header>
  );
}
