import { Link } from 'react-router-dom';
import { Container } from './Section';
import { LogoMark } from './Logo';

const COLUMNS: Array<{ heading: string; links: Array<{ label: string; to: string }> }> = [
  {
    heading: 'Product',
    links: [
      { label: 'Features', to: '/features' },
      { label: 'Pricing', to: '/pricing' },
      { label: 'Security', to: '/security' },
      { label: 'Admin console', to: '/login' },
    ],
  },
  {
    heading: 'Workflow',
    links: [
      { label: 'Documents', to: '/features' },
      { label: 'Approvals', to: '/features' },
      { label: 'Verifications', to: '/features' },
      { label: 'Templates', to: '/features' },
    ],
  },
  {
    heading: 'Company',
    links: [
      { label: 'Contact', to: '/contact' },
      { label: 'About', to: '/contact' },
      { label: 'Careers', to: '/contact' },
    ],
  },
];

export function LandingFooter() {
  return (
    <footer className="border-t border-ink-200/70 bg-paper-50">
      <Container className="py-14">
        <div className="grid gap-10 md:grid-cols-[1.4fr_1fr_1fr_1fr]">
          {/* Brand */}
          <div>
            <div className="flex items-center gap-2.5">
              <LogoMark />
              <span className="font-display text-lg font-semibold tracking-tight text-ink-950">DocuFlow</span>
            </div>
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-ink-500">
              The controlled document workflow. Create, verify, approve, and archive digital
              documents — every step signed, tracked, and auditable.
            </p>
            <div className="mt-5 flex items-center gap-2 text-[13px] text-ink-400">
              <span className="size-2 rounded-full bg-success-500" aria-hidden />
              All systems operational
            </div>
          </div>

          {COLUMNS.map((col) => (
            <nav key={col.heading} aria-label={col.heading}>
              <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-ink-400">{col.heading}</p>
              <ul className="mt-4 space-y-2.5">
                {col.links.map((link) => (
                  <li key={link.label}>
                    <Link
                      to={link.to}
                      className="text-sm text-ink-600 transition-colors hover:text-primary-600"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>

        <div className="mt-12 flex flex-col items-center justify-between gap-3 border-t border-ink-200/70 pt-6 sm:flex-row">
          <p className="text-xs text-ink-400">
            © {new Date().getFullYear()} DocuFlow. Every document, in its right place.
          </p>
          <div className="flex items-center gap-5 text-xs text-ink-400">
            <Link to="/security" className="transition-colors hover:text-ink-600">Privacy</Link>
            <Link to="/security" className="transition-colors hover:text-ink-600">Terms</Link>
            <Link to="/security" className="transition-colors hover:text-ink-600">Compliance</Link>
          </div>
        </div>
      </Container>
    </footer>
  );
}
