import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';

export function DashboardLayout() {
  return (
    <div className="dashboard-shell">
      <div className="dashboard-glow dashboard-glow-a" aria-hidden="true" />
      <div className="dashboard-glow dashboard-glow-b" aria-hidden="true" />

      <Sidebar />
      <div className="dashboard-main-pane">
        <Topbar />
        <main className="dashboard-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
