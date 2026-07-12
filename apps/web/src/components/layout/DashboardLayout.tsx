import { Suspense } from 'react';
import { useTranslation } from 'react-i18next';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';
import { UnverifiedBanner } from '../UnverifiedBanner';
import { useNotificationStream } from '../../hooks/useNotifications';

export function DashboardLayout() {
  const { t } = useTranslation();
  useNotificationStream();

  return (
    <div className="dashboard-shell">
      <div className="dashboard-glow dashboard-glow-a" aria-hidden="true" />
      <div className="dashboard-glow dashboard-glow-b" aria-hidden="true" />

      <Sidebar />
      <div className="dashboard-main-pane">
        <Topbar />
        {/* Chrome strip — sits between the topbar and content, not inside
            the scrollable content area, so page real estate isn't eaten. */}
        <UnverifiedBanner />
        <main className="dashboard-content">
          {/* Suspense boundary catches lazy() route chunks. Placing it
              here (not at the app root) means sidebar + topbar stay
              stable while the next route's chunk is being fetched —
              users see a small text placeholder in the content area
              instead of the entire dashboard chrome flashing away. */}
          <Suspense fallback={<p className="text-sm text-slate-500">{t('common.loading')}</p>}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
