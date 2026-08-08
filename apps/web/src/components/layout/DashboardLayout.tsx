import { Suspense, useEffect, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';
import { UnverifiedBanner } from '../UnverifiedBanner';
import { useNotificationStream } from '../../hooks/useNotifications';

/**
 * The dashboard chrome: sidebar + topbar + drawer + scroll lock. Renders
 * its children inside the main content area.
 *
 * Two ways to use it:
 *
 *   1. As a route parent — declare `<Route element={<DashboardLayout />}>`
 *      and let nested routes render via `<Outlet />`. The standard pattern
 *      for every other dashboard page.
 *
 *   2. With explicit children — pass a single child element and it
 *      renders inside the main content area instead of the outlet. Used
 *      by routes that share a URL with a public surface (see
 *      CampaignRoute in App.tsx), where declaring a nested route would
 *      shadow the public variant.
 *
 * Both work because `<Outlet />` is rendered only when no children were
 * passed — React Router gives us the rendered children of the route when
 * used as an element, and a null Outlet when used imperatively.
 */
export function DashboardLayout({ children }: { children?: ReactNode } = {}) {
  const { t } = useTranslation();
  useNotificationStream();

  // Mobile drawer state — sidebar is off-canvas below 860px and gets
  // toggled by the hamburger in the Topbar. Close automatically on
  // route change so tapping a nav link doesn't leave the drawer open
  // over the destination content.
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { pathname } = useLocation();
  useEffect(() => {
    setDrawerOpen(false);
  }, [pathname]);

  // Lock body scroll while the drawer is open so background content
  // doesn't scroll under the overlay.
  useEffect(() => {
    if (!drawerOpen) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, [drawerOpen]);

  return (
    <div className={`dashboard-shell${drawerOpen ? ' dashboard-shell--drawer-open' : ''}`}>
      <div className="dashboard-glow dashboard-glow-a" aria-hidden="true" />
      <div className="dashboard-glow dashboard-glow-b" aria-hidden="true" />

      <Sidebar onNavigate={() => setDrawerOpen(false)} />

      {/* Backdrop — only visible on mobile when drawer is open. Tapping
          it closes the drawer. aria-hidden because the sidebar itself
          holds keyboard focus. */}
      {drawerOpen && (
        <button
          type="button"
          className="dashboard-drawer-backdrop"
          aria-label={t('common.closeMenu', 'Close menu')}
          onClick={() => setDrawerOpen(false)}
        />
      )}

      <div className="dashboard-main-pane">
        <Topbar onOpenDrawer={() => setDrawerOpen(true)} />
        {/* Chrome strip — sits between the topbar and content, not inside
            the scrollable content area, so page real estate isn't eaten. */}
        <UnverifiedBanner />
        <main className="dashboard-content">
          {/* Suspense boundary catches lazy() route chunks. Placing it
              here (not at the app root) means sidebar + topbar stay
              stable while the next route's chunk is being fetched —
              users see a small text placeholder in the content area
              instead of the entire dashboard chrome flashing away. */}
          <Suspense
            fallback={
              <p className="text-sm" style={{ color: 'var(--dash-muted)' }}>
                {t('common.loading')}
              </p>
            }
          >
            {/* When used as a route parent, the nested route renders here.
                When used with explicit children, those render instead. */}
            {children ?? <Outlet />}
          </Suspense>
        </main>
      </div>
    </div>
  );
}
