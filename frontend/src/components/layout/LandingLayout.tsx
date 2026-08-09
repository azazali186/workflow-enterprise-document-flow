import { useEffect } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { LandingHeader } from '@/components/landing/LandingHeader';
import { LandingFooter } from '@/components/landing/LandingFooter';

export function LandingLayout() {
  const { pathname } = useLocation();

  // New page → start at the top (otherwise navigation lands mid-scroll).
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: 'instant' });
  }, [pathname]);

  return (
    <div className="flex min-h-dvh flex-col bg-paper-50">
      <LandingHeader />
      <main className="flex-1">
        <Outlet />
      </main>
      <LandingFooter />
    </div>
  );
}
