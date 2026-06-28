import { Routes, Route, Navigate } from 'react-router-dom';
import { DashboardLayout } from './components/layout/DashboardLayout';
import { LoginPage } from './routes/LoginPage';
import { RegisterPage } from './routes/RegisterPage';
import { CommunityPage } from './routes/CommunityPage';
import { IssuesPage } from './routes/IssuesPage';
import { PetitionsPage } from './routes/PetitionsPage';
import { RepresentativesPage } from './routes/RepresentativesPage';
import { NotificationsPage } from './routes/NotificationsPage';
import { ProfilePage } from './routes/ProfilePage';

export default function App() {
  return (
    <Routes>
      {/* Public */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      {/* Authenticated */}
      <Route element={<DashboardLayout />}>
        <Route path="/community" element={<CommunityPage />} />
        <Route path="/issues" element={<IssuesPage />} />
        <Route path="/petitions" element={<PetitionsPage />} />
        <Route path="/representatives" element={<RepresentativesPage />} />
        <Route path="/notifications" element={<NotificationsPage />} />
        <Route path="/profile" element={<ProfilePage />} />
      </Route>

      {/* Default */}
      <Route path="/" element={<Navigate to="/community" replace />} />
      <Route path="*" element={<Navigate to="/community" replace />} />
    </Routes>
  );
}
