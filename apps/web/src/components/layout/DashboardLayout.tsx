import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';
import { UnverifiedBanner } from '../UnverifiedBanner';
import { useNotificationStream } from '../../hooks/useNotifications';

export function DashboardLayout() {
  useNotificationStream();

  return (
    <div className="dashboard-shell">
      <div className="dashboard-glow dashboard-glow-a" aria-hidden="true" />
      <div className="dashboard-glow dashboard-glow-b" aria-hidden="true" />

      <Sidebar />
      <div className="dashboard-main-pane">
        <Topbar />
        <main className="dashboard-content">
          <UnverifiedBanner />
          <Outlet />
        </main>
      </div>
    </div>
  );
}
