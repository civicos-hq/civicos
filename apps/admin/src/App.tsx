import { Navigate, Route, Routes } from 'react-router-dom';
import { LoginPage } from './routes/LoginPage';
import { OverviewPage } from './routes/OverviewPage';
import { UsersPage } from './routes/UsersPage';
import { OrganizationsPage } from './routes/OrganizationsPage';
import { FlagsPage } from './routes/FlagsPage';
import { AuditPage } from './routes/AuditPage';
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
          <Route path="/organizations" element={<OrganizationsPage />} />
          <Route path="/flags" element={<FlagsPage />} />
          <Route path="/audit" element={<AuditPage />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
