import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom';
import { InlineLoading } from '@carbon/react';
import { useAuth } from './contexts/AuthContext';
import { AppLayout } from './components/AppLayout';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { DashboardPage } from './pages/DashboardPage';
import { UsersPage } from './pages/UsersPage';
import { DevDocsPage } from './pages/DevDocsPage';
import { DevDocViewPage } from './pages/DevDocViewPage';
import { DevDocEditPage } from './pages/DevDocEditPage';
import { SettingsPage } from './pages/SettingsPage';
import { NotificationsPage } from './pages/NotificationsPage';
import { ProfilePage } from './pages/ProfilePage';
import { TasksPage } from './pages/modules/TasksPage';
import { AgendaPage } from './pages/modules/AgendaPage';
import { FollowUpsPage } from './pages/modules/FollowUpsPage';
import { StationsPage } from './pages/modules/StationsPage';
import { StationDetailPage } from './pages/modules/StationDetailPage';
import { NokiaPage } from './pages/modules/NokiaPage';
import { NokiaDetailPage } from './pages/modules/NokiaDetailPage';
import { NotFoundPage, ForbiddenPage } from './pages/StatusPages';

function Splash() {
  return (
    <div className="app-splash">
      <div className="app-splash__brand">Documentation Center</div>
      <InlineLoading description="Carregando sessão…" />
    </div>
  );
}

function ProtectedLayout() {
  const { user, loading } = useAuth();
  const location = useLocation();
  if (loading) return <Splash />;
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  );
}

function AdminRoute() {
  const { user } = useAuth();
  if (!user?.is_admin) return <ForbiddenPage />;
  return <Outlet />;
}

function PublicOnly({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  if (loading) return <Splash />;
  if (user) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<PublicOnly><LoginPage /></PublicOnly>} />
      <Route path="/register" element={<PublicOnly><RegisterPage /></PublicOnly>} />
      <Route element={<ProtectedLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/notifications" element={<NotificationsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/agenda" element={<AgendaPage />} />
        <Route path="/tasks" element={<TasksPage />} />
        <Route path="/follow-ups" element={<FollowUpsPage />} />
        <Route path="/stations" element={<StationsPage />} />
        <Route path="/stations/:id" element={<StationDetailPage />} />
        <Route path="/nokia" element={<NokiaPage />} />
        <Route path="/nokia/:id" element={<NokiaDetailPage />} />
        <Route element={<AdminRoute />}>
          <Route path="/users" element={<UsersPage />} />
          <Route path="/dev-docs" element={<DevDocsPage />} />
          <Route path="/dev-docs/new" element={<DevDocEditPage />} />
          <Route path="/dev-docs/:id" element={<DevDocViewPage />} />
          <Route path="/dev-docs/:id/edit" element={<DevDocEditPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
