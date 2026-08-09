import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { Menu, X } from 'lucide-react';
import { cn } from '@/lib/cn';
import { Container } from './Section';
import { Logo } from './Logo';

const NAV_LINKS = [
  { to: '/features', label: 'Features' },
  { to: '/pricing', label: 'Pricing' },
  { to: '/security', label: 'Security' },
  { to: '/contact', label: 'Contact' },
];

export function LandingHeader() {
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const { pathname } = useLocation();

  // Elevate the header with a hairline + shadow once the page scrolls.
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  // Close the mobile menu on navigation.
  useEffect(() => setMenuOpen(false), [pathname]);

  return (
    <header
      className={cn(
        'sticky top-0 z-50 border-b transition-all duration-200',
        scrolled || menuOpen
          ? 'border-ink-200/70 bg-paper-50/90 backdrop-blur-md'
          : 'border-transparent bg-paper-50/60 backdrop-blur-sm',
      )}
    >
      <Container>
        <div className="flex h-16 items-center justify-between gap-4">
          <Logo />

          {/* Desktop nav */}
          <nav className="hidden items-center gap-1 md:flex" aria-label="Primary">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                className={({ isActive }) =>
                  cn(
                    'rounded-lg px-3 py-2 text-[13.5px] font-medium transition-colors duration-150',
                    isActive
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-ink-600 hover:bg-ink-100/60 hover:text-ink-950',
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
          </nav>

          <div className="hidden items-center gap-2.5 md:flex">
            <Link
              to="/login"
              className="inline-flex h-8 items-center rounded-lg px-3 text-[13.5px] font-medium text-ink-600 transition-colors hover:bg-ink-100/60 hover:text-ink-950"
            >
              Sign in
            </Link>
            <Link
              to="/login"
              className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-primary-600 px-3.5 text-[13.5px] font-medium text-white shadow-sm transition-colors hover:bg-primary-700"
            >
              Open console
            </Link>
          </div>

          {/* Mobile toggle */}
          <button
            onClick={() => setMenuOpen((v) => !v)}
            aria-expanded={menuOpen}
            aria-label={menuOpen ? 'Close menu' : 'Open menu'}
            className="rounded-lg p-2 text-ink-600 transition-colors hover:bg-ink-100 hover:text-ink-950 md:hidden cursor-pointer"
          >
            {menuOpen ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>
        </div>
      </Container>

      {/* Mobile menu */}
      {menuOpen && (
        <div className="border-t border-ink-200/70 bg-paper-50 md:hidden animate-fade-in">
          <Container className="flex flex-col gap-1 py-4">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                className={({ isActive }) =>
                  cn(
                    'rounded-lg px-3 py-2.5 text-[15px] font-medium transition-colors',
                    isActive ? 'bg-primary-50 text-primary-700' : 'text-ink-700 hover:bg-ink-100/60',
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <div className="mt-3 flex flex-col gap-2.5 border-t border-ink-200/70 pt-4">
              <Link
                to="/login"
                className="inline-flex h-9.5 w-full items-center justify-center rounded-lg border border-ink-200 bg-white text-sm font-medium text-ink-700 transition-colors hover:bg-paper-100"
              >
                Sign in
              </Link>
              <Link
                to="/login"
                className="inline-flex h-9.5 w-full items-center justify-center rounded-lg bg-primary-600 text-sm font-medium text-white transition-colors hover:bg-primary-700"
              >
                Open console
              </Link>
            </div>
          </Container>
        </div>
      )}
    </header>
  );
}
