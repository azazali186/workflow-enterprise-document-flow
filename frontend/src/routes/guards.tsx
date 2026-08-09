import { useEffect, type ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '@/store';
import { clearSession } from '@/store/authSlice';
import { setUnauthorizedHandler } from '@/services/http';
import { AppLayout } from '@/components/layout/AppLayout';

function SessionSplash() {
  return (
    <div className="flex min-h-dvh items-center justify-center bg-paper-50">
      <div
        className="h-8 w-8 animate-spin rounded-full border-2 border-ink-200 border-t-primary-600"
        aria-label="Loading session"
      />
    </div>
  );
}

/** Wraps protected routes; wires the 401 handler and gates on session status. */
export function ProtectedRoute() {
  const dispatch = useAppDispatch();
  const location = useLocation();
  const status = useAppSelector((s) => s.auth.status);

  useEffect(() => {
    setUnauthorizedHandler(() => dispatch(clearSession()));
    return () => setUnauthorizedHandler(null);
  }, [dispatch]);

  // Boot restore (cookie probe) in flight → splash, not a login flash.
  if (status === 'idle' || status === 'loading') {
    return <SessionSplash />;
  }
  if (status !== 'authenticated') {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return <AppLayout />;
}

/** Blocks the login page for already-authenticated users (splash during boot). */
export function PublicOnly({ children }: { children: ReactNode }) {
  const { status } = useAppSelector((s) => s.auth);

  if (status === 'idle' || status === 'loading') {
    return <SessionSplash />;
  }
  if (status === 'authenticated') {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
