import { Navigate, Route, Routes } from 'react-router-dom';
import { LoginPage } from './routes/LoginPage';
import { OverviewPage } from './routes/OverviewPage';
import { UsersPage } from './routes/UsersPage';
import { UserDetailPage } from './routes/UserDetailPage';
import { OrganizationsPage } from './routes/OrganizationsPage';
import { OrganizationCreatePage } from './routes/OrganizationCreatePage';
import { CommunitiesPage } from './routes/CommunitiesPage';
import { CommunityCreatePage } from './routes/CommunityCreatePage';
import { CommunityDetailPage } from './routes/CommunityDetailPage';
import { RepresentativesPage } from './routes/RepresentativesPage';
import { RepresentativeCreatePage } from './routes/RepresentativeCreatePage';
import { FlagsPage } from './routes/FlagsPage';
import { DirectHidePage } from './routes/DirectHidePage';
import { AuditPage } from './routes/AuditPage';
import { OrganizationDetailPage } from './routes/OrganizationDetailPage';
import { AdminShell } from './components/AdminShell';
import { RequireAdmin } from './components/RequireAdmin';

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAdmin />}>
        <Route element={<AdminShell />}>
          <Route index element={<OverviewPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/users/:id" element={<UserDetailPage />} />
          <Route path="/organizations" element={<OrganizationsPage />} />
          <Route path="/organizations/new" element={<OrganizationCreatePage />} />
          <Route path="/organizations/:id" element={<OrganizationDetailPage />} />
          <Route path="/communities" element={<CommunitiesPage />} />
          <Route path="/communities/new" element={<CommunityCreatePage />} />
          <Route path="/communities/:id" element={<CommunityDetailPage />} />
          <Route path="/representatives" element={<RepresentativesPage />} />
          <Route path="/representatives/new" element={<RepresentativeCreatePage />} />
          <Route path="/moderation/direct-hide" element={<DirectHidePage />} />
          <Route path="/flags" element={<FlagsPage />} />
          <Route path="/audit" element={<AuditPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
