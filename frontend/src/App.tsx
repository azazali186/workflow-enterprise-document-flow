import { lazy, Suspense, useEffect } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { ProtectedRoute, PublicOnly } from '@/routes/guards';
import { PageFallback } from '@/components/ui/PageFallback';
import { restoreSession } from '@/store/authSlice';
import { useAppDispatch } from '@/store';

const LoginPage = lazy(() => import('@/features/auth/LoginPage').then((m) => ({ default: m.LoginPage })));
const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage })));
const UsersPage = lazy(() => import('@/features/users/UsersPage').then((m) => ({ default: m.UsersPage })));
const RolesPage = lazy(() => import('@/features/roles/RolesPage').then((m) => ({ default: m.RolesPage })));
const DocumentsPage = lazy(() => import('@/features/documents/DocumentsPage').then((m) => ({ default: m.DocumentsPage })));
const CategoriesPage = lazy(() => import('@/features/categories/CategoriesPage').then((m) => ({ default: m.CategoriesPage })));
const AuditPage = lazy(() => import('@/features/audit/AuditPage').then((m) => ({ default: m.AuditPage })));
const ApprovalsPage = lazy(() => import('@/features/approvals/ApprovalsPage').then((m) => ({ default: m.ApprovalsPage })));
const VerificationsPage = lazy(() => import('@/features/verifications/VerificationsPage').then((m) => ({ default: m.VerificationsPage })));
const TemplatesPage = lazy(() => import('@/features/templates/TemplatesPage').then((m) => ({ default: m.TemplatesPage })));
const StoragesPage = lazy(() => import('@/features/storages/StoragesPage').then((m) => ({ default: m.StoragesPage })));
const AccessesPage = lazy(() => import('@/features/accesses/AccessesPage').then((m) => ({ default: m.AccessesPage })));
const VersionsPage = lazy(() => import('@/features/versions/VersionsPage').then((m) => ({ default: m.VersionsPage })));
const LoginLogsPage = lazy(() => import('@/features/loginlogs/LoginLogsPage').then((m) => ({ default: m.LoginLogsPage })));

export default function App() {
  const dispatch = useAppDispatch();

  // Boot-time session restore: probe the HttpOnly session cookie via refresh
  // so reloads land on the dashboard without a login flash.
  useEffect(() => {
    void dispatch(restoreSession());
  }, [dispatch]);

  return (
    <Suspense fallback={<PageFallback />}>
      <Routes>
        <Route path="/login" element={<PublicOnly><LoginPage /></PublicOnly>} />

        <Route element={<ProtectedRoute />}>
          <Route index element={<DashboardPage />} />
          <Route path="documents" element={<DocumentsPage />} />
          <Route path="categories" element={<CategoriesPage />} />
          <Route path="roles" element={<RolesPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="approvals" element={<ApprovalsPage />} />
          <Route path="verifications" element={<VerificationsPage />} />
          <Route path="templates" element={<TemplatesPage />} />
          <Route path="storages" element={<StoragesPage />} />
          <Route path="accesses" element={<AccessesPage />} />
          <Route path="versions" element={<VersionsPage />} />
          <Route path="login-logs" element={<LoginLogsPage />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
