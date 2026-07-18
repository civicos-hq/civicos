import { lazy } from 'react';
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';
import { DashboardLayout } from './components/layout/DashboardLayout';
import { RateLimitToast } from './components/RateLimitToast';

// Marketing + auth surfaces stay eagerly imported: fresh visitors land on
// one of them, so any lazy split just adds a paint delay to the first
// screen a user ever sees. The dashboard routes below are behind a login
// wall — nobody sees them without an authenticated round-trip — so
// splitting cuts the initial payload for the visitor traffic that matters
// most (marketing → sign-up conversion).
import { HomePage } from './routes/HomePage';
import { PrivacyPage } from './routes/PrivacyPage';
import { TermsPage } from './routes/TermsPage';
import { LoginPage } from './routes/LoginPage';
import { RegisterPage } from './routes/RegisterPage';
import { ForgotPasswordPage } from './routes/ForgotPasswordPage';
import { ResetPasswordPage } from './routes/ResetPasswordPage';
import { VerifyEmailPage } from './routes/VerifyEmailPage';
import { VerifyEmailSentPage } from './routes/VerifyEmailSentPage';
import { OnboardingPage } from './routes/OnboardingPage';

// Dashboard routes — lazy. Each becomes its own Vite chunk, fetched only
// when the user actually navigates to it. Suspense boundary lives inside
// DashboardLayout so the sidebar + topbar stay stable during the fetch;
// only the main content area shows the fallback.
const DiscoverPage = lazy(() =>
  import('./routes/DiscoverPage').then((m) => ({ default: m.DiscoverPage })),
);
const CommunityPage = lazy(() =>
  import('./routes/CommunityPage').then((m) => ({ default: m.CommunityPage })),
);
const IssuesPage = lazy(() =>
  import('./routes/IssuesPage').then((m) => ({ default: m.IssuesPage })),
);
const IssueDetailPage = lazy(() =>
  import('./routes/IssueDetailPage').then((m) => ({ default: m.IssueDetailPage })),
);
const PetitionsPage = lazy(() =>
  import('./routes/PetitionsPage').then((m) => ({ default: m.PetitionsPage })),
);
const PetitionDetailPage = lazy(() =>
  import('./routes/PetitionDetailPage').then((m) => ({ default: m.PetitionDetailPage })),
);
const RepresentativesPage = lazy(() =>
  import('./routes/RepresentativesPage').then((m) => ({ default: m.RepresentativesPage })),
);
const RepresentativeDetailPage = lazy(() =>
  import('./routes/RepresentativeDetailPage').then((m) => ({
    default: m.RepresentativeDetailPage,
  })),
);
const OrganizationsPage = lazy(() =>
  import('./routes/OrganizationsPage').then((m) => ({ default: m.OrganizationsPage })),
);
const OrganizationDetailPage = lazy(() =>
  import('./routes/OrganizationDetailPage').then((m) => ({ default: m.OrganizationDetailPage })),
);
const ConsultationsPage = lazy(() =>
  import('./routes/ConsultationsPage').then((m) => ({ default: m.ConsultationsPage })),
);
const ConsultationDetailPage = lazy(() =>
  import('./routes/ConsultationDetailPage').then((m) => ({ default: m.ConsultationDetailPage })),
);
const AnnouncementsPage = lazy(() =>
  import('./routes/AnnouncementsPage').then((m) => ({ default: m.AnnouncementsPage })),
);
const AnnouncementDetailPage = lazy(() =>
  import('./routes/AnnouncementDetailPage').then((m) => ({ default: m.AnnouncementDetailPage })),
);
const ProjectsPage = lazy(() =>
  import('./routes/ProjectsPage').then((m) => ({ default: m.ProjectsPage })),
);
const ProjectDetailPage = lazy(() =>
  import('./routes/ProjectDetailPage').then((m) => ({ default: m.ProjectDetailPage })),
);
const OrgLandingPage = lazy(() =>
  import('./routes/OrgLandingPage').then((m) => ({ default: m.OrgLandingPage })),
);
const OrgDashboardPage = lazy(() =>
  import('./routes/OrgDashboardPage').then((m) => ({ default: m.OrgDashboardPage })),
);
const OrgConsultationCreatePage = lazy(() =>
  import('./routes/OrgConsultationCreatePage').then((m) => ({
    default: m.OrgConsultationCreatePage,
  })),
);
const OrgConsultationDetailPage = lazy(() =>
  import('./routes/OrgConsultationDetailPage').then((m) => ({
    default: m.OrgConsultationDetailPage,
  })),
);
const OrgAnnouncementCreatePage = lazy(() =>
  import('./routes/OrgAnnouncementCreatePage').then((m) => ({
    default: m.OrgAnnouncementCreatePage,
  })),
);
const OrgAnnouncementDetailPage = lazy(() =>
  import('./routes/OrgAnnouncementDetailPage').then((m) => ({
    default: m.OrgAnnouncementDetailPage,
  })),
);
const OrgProjectCreatePage = lazy(() =>
  import('./routes/OrgProjectCreatePage').then((m) => ({ default: m.OrgProjectCreatePage })),
);
const OrgProjectDetailPage = lazy(() =>
  import('./routes/OrgProjectDetailPage').then((m) => ({ default: m.OrgProjectDetailPage })),
);
const NotificationsPage = lazy(() =>
  import('./routes/NotificationsPage').then((m) => ({ default: m.NotificationsPage })),
);
const ProfilePage = lazy(() =>
  import('./routes/ProfilePage').then((m) => ({ default: m.ProfilePage })),
);

export function hasAccessToken() {
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
    <>
      {/* Mounted at the root so 429 toasts appear on public + authed pages. */}
      <RateLimitToast />
      <AppRoutes />
    </>
  );
}

function AppRoutes() {
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

      {/* Privacy policy — always reachable regardless of auth state. */}
      <Route path="/privacy" element={<PrivacyPage />} />
      <Route path="/terms" element={<TermsPage />} />

      {/* Authenticated */}
      <Route element={<RequireAuth />}>
        {/* Onboarding sits outside the dashboard chrome so the wizard is
            distraction-free. It self-redirects to /discover once the user
            has a community, so revisiting the URL is safe. */}
        <Route path="/onboarding" element={<OnboardingPage />} />
        <Route element={<DashboardLayout />}>
          <Route path="/discover" element={<DiscoverPage />} />
          <Route path="/community" element={<CommunityPage />} />
          <Route path="/issues" element={<IssuesPage />} />
          <Route path="/issues/:id" element={<IssueDetailPage />} />
          <Route path="/petitions" element={<PetitionsPage />} />
          <Route path="/petitions/:id" element={<PetitionDetailPage />} />
          <Route path="/representatives" element={<RepresentativesPage />} />
          <Route path="/representatives/:id" element={<RepresentativeDetailPage />} />
          <Route path="/organizations" element={<OrganizationsPage />} />
          <Route path="/organizations/:id" element={<OrganizationDetailPage />} />
          <Route path="/consultations" element={<ConsultationsPage />} />
          <Route path="/consultations/:id" element={<ConsultationDetailPage />} />
          <Route path="/announcements" element={<AnnouncementsPage />} />
          <Route path="/announcements/:id" element={<AnnouncementDetailPage />} />
          <Route path="/projects" element={<ProjectsPage />} />
          <Route path="/projects/:id" element={<ProjectDetailPage />} />
          <Route path="/org" element={<OrgLandingPage />} />
          <Route path="/org/:orgId" element={<OrgDashboardPage />} />
          <Route path="/org/:orgId/consultations/new" element={<OrgConsultationCreatePage />} />
          <Route path="/org/:orgId/consultations/:id" element={<OrgConsultationDetailPage />} />
          <Route path="/org/:orgId/announcements/new" element={<OrgAnnouncementCreatePage />} />
          <Route path="/org/:orgId/announcements/:id" element={<OrgAnnouncementDetailPage />} />
          <Route path="/org/:orgId/projects/new" element={<OrgProjectCreatePage />} />
          <Route path="/org/:orgId/projects/:id" element={<OrgProjectDetailPage />} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Route>

      {/* Marketing homepage — visible to everyone, signed in or out.
          Sidebar brand link on the dashboard points here, so signed-in
          users need to be able to reach it. TopNav on the homepage is
          auth-aware and shows a "Dashboard" CTA when the visitor has
          an access token. */}
      <Route path="/" element={<HomePage />} />
      <Route path="*" element={<Navigate to={hasAccessToken() ? '/discover' : '/'} replace />} />
    </Routes>
  );
}
