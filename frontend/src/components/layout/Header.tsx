import { useLocation } from 'react-router-dom';
import { Menu } from 'lucide-react';
import { useAppDispatch, useAppSelector } from '@/store';
import { setMobileSidebar } from '@/store/uiSlice';
import { Avatar } from './Avatar';

/** Derives a breadcrumb title from the current route. */
export function usePageTitle(): string {
  const { pathname } = useLocation();
  if (pathname === '/app') return 'Dashboard';
  // Routes under the admin console are /app/<page>.
  const segment = pathname.split('/').filter(Boolean)[1];
  return segment
    ? segment.charAt(0).toUpperCase() + segment.slice(1).replace(/-/g, ' ')
    : 'Overview';
}

export function Header() {
  const dispatch = useAppDispatch();
  const user = useAppSelector((s) => s.auth.user);
  const title = usePageTitle();

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-ink-200/70 bg-paper-50/85 px-4 backdrop-blur-md sm:px-6">
      <button
        onClick={() => dispatch(setMobileSidebar(true))}
        aria-label="Open navigation"
        className="rounded-lg p-2 text-ink-500 transition-colors hover:bg-ink-100 hover:text-ink-900 lg:hidden cursor-pointer"
      >
        <Menu className="size-5" />
      </button>

      <div className="min-w-0 flex-1">
        <h2 className="truncate font-display text-lg font-semibold tracking-tight text-ink-950">{title}</h2>
      </div>

      <div className="flex items-center gap-3">
        <div className="hidden text-right sm:block">
          <p className="text-[13px] font-medium leading-tight text-ink-800">{user?.name || 'User'}</p>
          <p className="text-xs leading-tight text-ink-400">{user?.email}</p>
        </div>
        <Avatar name={user?.name || 'U'} size="md" />
      </div>
    </header>
  );
}
