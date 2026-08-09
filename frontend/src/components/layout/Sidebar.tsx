import { NavLink, useNavigate } from 'react-router-dom';
import {
  Archive,
  BadgeCheck,
  CheckCircle2,
  ChevronLeft,
  FileText,
  FolderTree,
  History,
  KeyRound,
  LayoutDashboard,
  LogOut,
  ScrollText,
  ShieldCheck,
  Table2,
  Users,
} from 'lucide-react';
import { cn } from '@/lib/cn';
import { useAppDispatch, useAppSelector } from '@/store';
import { toggleSidebar, setMobileSidebar } from '@/store/uiSlice';
import { logout } from '@/store/authSlice';

interface NavItem {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
}

const navGroups: Array<{ label: string; items: NavItem[] }> = [
  {
    label: 'Overview',
    items: [{ to: '/app', label: 'Dashboard', icon: LayoutDashboard, end: true }],
  },
  {
    label: 'Workspace',
    items: [
      { to: '/app/documents', label: 'Documents', icon: FileText },
      { to: '/app/categories', label: 'Categories', icon: FolderTree },
      { to: '/app/roles', label: 'Roles & Permissions', icon: ShieldCheck },
      { to: '/app/users', label: 'Users', icon: Users },
    ],
  },
  {
    label: 'Workflow',
    items: [
      { to: '/app/approvals', label: 'Approvals', icon: CheckCircle2 },
      { to: '/app/verifications', label: 'Verifications', icon: BadgeCheck },
      { to: '/app/templates', label: 'Templates', icon: Table2 },
      { to: '/app/versions', label: 'Versions', icon: History },
      { to: '/app/accesses', label: 'Access Grants', icon: KeyRound },
    ],
  },
  {
    label: 'Records',
    items: [
      { to: '/app/audit', label: 'Audit Logs', icon: ScrollText },
      { to: '/app/login-logs', label: 'Login Logs', icon: LogOut },
    ],
  },
];

export function Sidebar() {
  const dispatch = useAppDispatch();
  const collapsed = useAppSelector((s) => s.ui.sidebarCollapsed);
  const mobileOpen = useAppSelector((s) => s.ui.mobileSidebarOpen);
  const navigate = useNavigate();

  const closeMobile = () => dispatch(setMobileSidebar(false));

  const handleLogout = async () => {
    closeMobile();
    await dispatch(logout());
    navigate('/login');
  };

  return (
    <>
      {/* Mobile backdrop */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-ink-950/50 backdrop-blur-[2px] lg:hidden animate-fade-in"
          onClick={closeMobile}
          aria-hidden
        />
      )}

      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex flex-col bg-ink-950 text-ink-200 transition-[width,transform] duration-200',
          'lg:static lg:z-auto',
          collapsed ? 'lg:w-18' : 'lg:w-60',
          'w-60',
          mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        )}
      >
        {/* Brand */}
        <div className={cn('flex h-16 items-center gap-2.5 border-b border-white/5 px-5', collapsed && 'lg:justify-center lg:px-0')}>
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary-600 text-white">
            <Archive className="size-4.5" aria-hidden />
          </div>
          {!collapsed && (
            <div className="min-w-0">
              <p className="truncate font-display text-[15px] font-semibold tracking-tight text-white">DocuFlow</p>
              <p className="text-[10px] uppercase tracking-widest text-ink-400">Admin Console</p>
            </div>
          )}
        </div>

        {/* Nav */}
        <nav className="nice-scroll flex-1 overflow-y-auto px-3 py-4">
          {navGroups.map((group) => (
            <div key={group.label} className="mb-5">
              {!collapsed && (
                <p className="mb-1.5 px-2.5 text-[10px] font-semibold uppercase tracking-widest text-ink-500">
                  {group.label}
                </p>
              )}
              <ul className="space-y-0.5">
                {group.items.map((item) => (
                  <li key={item.to}>
                    <NavLink
                      to={item.to}
                      end={item.end}
                      onClick={closeMobile}
                      title={collapsed ? item.label : undefined}
                      className={({ isActive }) =>
                        cn(
                          'group flex items-center gap-3 rounded-lg px-2.5 py-2 text-[13px] font-medium transition-colors duration-150',
                          collapsed && 'lg:justify-center lg:px-0',
                          isActive
                            ? 'bg-primary-600/15 text-primary-300'
                            : 'text-ink-300 hover:bg-white/5 hover:text-white',
                        )
                      }
                    >
                      <item.icon className="size-4.5 shrink-0" aria-hidden />
                      {!collapsed && <span className="truncate">{item.label}</span>}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>

        {/* Footer: collapse toggle + logout */}
        <div className="border-t border-white/5 p-3">
          <button
            onClick={() => dispatch(toggleSidebar())}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-[13px] font-medium text-ink-400 transition-colors hover:bg-white/5 hover:text-white cursor-pointer',
              collapsed && 'lg:justify-center lg:px-0',
            )}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <ChevronLeft className={cn('size-4.5 shrink-0 transition-transform duration-200', collapsed && 'lg:rotate-180')} aria-hidden />
            {!collapsed && <span>Collapse</span>}
          </button>
          <button
            onClick={handleLogout}
            className={cn(
              'flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-[13px] font-medium text-ink-400 transition-colors hover:bg-danger-500/10 hover:text-danger-400 cursor-pointer',
              collapsed && 'lg:justify-center lg:px-0',
            )}
          >
            <LogOut className="size-4.5 shrink-0" aria-hidden />
            {!collapsed && <span>Sign out</span>}
          </button>
        </div>
      </aside>
    </>
  );
}
