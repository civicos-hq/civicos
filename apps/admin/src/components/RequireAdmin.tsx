import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { getSession } from '../lib/api';

// Bounces any request without a stored PLATFORM_ADMIN session back to
// /login (preserving where they were trying to go via ?redirect=).
export function RequireAdmin() {
  const session = getSession();
  const location = useLocation();
  if (!session || session.user.role !== 'PLATFORM_ADMIN') {
    const redirect = encodeURIComponent(location.pathname + location.search);
    return <Navigate to={`/login?redirect=${redirect}`} replace />;
  }
  return <Outlet />;
}
