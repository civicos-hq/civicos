import { Navigate, Outlet, Route, Routes } from 'react-router-dom';
import { DashboardLayout } from './components/layout/DashboardLayout';
import { CommunityPage } from './routes/CommunityPage';
import { DiscoverPage } from './routes/DiscoverPage';
import { HomePage } from './routes/HomePage';
import { IssueDetailPage } from './routes/IssueDetailPage';
import { IssuesPage } from './routes/IssuesPage';
import { LoginPage } from './routes/LoginPage';
import { NotificationsPage } from './routes/NotificationsPage';
import { PetitionDetailPage } from './routes/PetitionDetailPage';
import { PetitionsPage } from './routes/PetitionsPage';
import { ProfilePage } from './routes/ProfilePage';
import { RegisterPage } from './routes/RegisterPage';
import { RepresentativeDetailPage } from './routes/RepresentativeDetailPage';
import { RepresentativesPage } from './routes/RepresentativesPage';
import { VerifyEmailPage } from './routes/VerifyEmailPage';
import { VerifyEmailSentPage } from './routes/VerifyEmailSentPage';
import { ForgotPasswordPage } from './routes/ForgotPasswordPage';
import { ResetPasswordPage } from './routes/ResetPasswordPage';

function hasAccessToken() {
  return Boolean(localStorage.getItem('accessToken'));
}

function RequireAuth() {
  if (!hasAccessToken()) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}

function PublicOnly() {
  if (hasAccessToken()) {
    return <Navigate to="/discover" replace />;
  }
  return <Outlet />;
}

export default function App() {
  return (
    <Routes>
      {/* Public */}
      <Route element={<PublicOnly />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>

      {/* Email verification — reachable whether or not signed in (the user
          may click the link from another browser/device). */}
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route path="/verify-email-sent" element={<VerifyEmailSentPage />} />

      {/* Password recovery — reachable while logged-out (the whole point) and
          intentionally left reachable while logged-in too so a user who is
          worried their session was hijacked can still trigger a reset. */}
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />

      {/* Authenticated */}
      <Route element={<RequireAuth />}>
        <Route element={<DashboardLayout />}>
          <Route path="/discover" element={<DiscoverPage />} />
          <Route path="/community" element={<CommunityPage />} />
          <Route path="/issues" element={<IssuesPage />} />
          <Route path="/issues/:id" element={<IssueDetailPage />} />
          <Route path="/petitions" element={<PetitionsPage />} />
          <Route path="/petitions/:id" element={<PetitionDetailPage />} />
          <Route path="/representatives" element={<RepresentativesPage />} />
          <Route path="/representatives/:id" element={<RepresentativeDetailPage />} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Route>

      {/* Default — marketing homepage for logged-out visitors, dashboard for logged-in */}
      <Route
        path="/"
        element={hasAccessToken() ? <Navigate to="/discover" replace /> : <HomePage />}
      />
      <Route path="*" element={<Navigate to={hasAccessToken() ? '/discover' : '/'} replace />} />
    </Routes>
  );
}
