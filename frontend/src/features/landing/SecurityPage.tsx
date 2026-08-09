import { Link } from 'react-router-dom';
import {
  Cookie,
  Fingerprint,
  Gauge,
  KeyRound,
  Lock,
  ScrollText,
  ShieldCheck,
  Workflow,
} from 'lucide-react';
import { Container, Section, SectionHeader } from '@/components/landing/Section';
import { useReveal } from '@/hooks/useReveal';

const PILLARS = [
  {
    icon: Cookie,
    title: 'Secure sessions',
    body: 'Sessions ride in HttpOnly, SameSite cookies — never in JavaScript-readable storage. Cross-site scripting cannot steal your session token.',
  },
  {
    icon: ShieldCheck,
    title: 'CSRF protection',
    body: 'Every state-changing request carries a double-submit CSRF token bound to your session and verified in constant time. Cross-site request forgery is rejected before it reaches a handler.',
  },
  {
    icon: Lock,
    title: 'Password security',
    body: 'Passwords are hashed with a deliberately slow, salted algorithm. Failed logins are counted, accounts are locked, and every attempt is logged.',
  },
  {
    icon: KeyRound,
    title: 'Role-based access control',
    body: 'Permissions gate every console action, and document-level grants refine access further — read, write, or approve, per user or per role.',
  },
  {
    icon: ScrollText,
    title: 'Immutable audit trail',
    body: 'Logins, logouts, permissions changes, verifications, and approvals are written to an append-only audit log. Every decision has an actor and a timestamp.',
  },
  {
    icon: Gauge,
    title: 'Rate limiting',
    body: 'Authentication endpoints are rate-limited to slow credential stuffing and brute force. The abuse signal is visible in the logs.',
  },
  {
    icon: Workflow,
    title: 'Single active session',
    body: 'Each user holds one active session. Logging in elsewhere rotates the old session — so a leaked session cannot coexist with the real user.',
  },
  {
    icon: Fingerprint,
    title: 'Secrets by environment',
    body: 'The configuration is twelve-factor: keys are injected per environment and never committed. Known weak development defaults are rejected in production.',
  },
];

export function SecurityPage() {
  const revealRef = useReveal<HTMLDivElement>();

  return (
    <div ref={revealRef}>
      <Section className="relative overflow-hidden pb-8 pt-16 sm:pt-20">
        <div
          className="pointer-events-none absolute inset-x-0 top-0 h-80 bg-[radial-gradient(55%_60%_at_50%_-10%,rgb(92_102_230/0.12),transparent)]"
          aria-hidden
        />
        <Container className="relative">
          <SectionHeader
            center
            eyebrow="Security"
            title="Trustworthy by construction, not by promise"
            description="DocuFlow is built around the idea that a document is only as valuable as the proof of its journey. Here is how we protect both."
          />
        </Container>
      </Section>

      <Section className="pt-12">
        <Container>
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {PILLARS.map((p, i) => (
              <div
                key={p.title}
                data-reveal
                style={{ ['--reveal-delay' as string]: `${(i % 4) * 80}ms` }}
                className="flex flex-col rounded-2xl border border-ink-200/70 bg-white p-6 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-card"
              >
                <div className="flex size-10 items-center justify-center rounded-xl bg-primary-50 text-primary-600">
                  <p.icon className="size-5" aria-hidden />
                </div>
                <h3 className="mt-4 font-display text-lg font-semibold tracking-tight text-ink-900">{p.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-ink-500">{p.body}</p>
              </div>
            ))}
          </div>
        </Container>
      </Section>

      {/* Deep-dive band */}
      <section className="relative overflow-hidden bg-ink-950 text-white">
        <div className="ink-grid absolute inset-0" aria-hidden />
        <Container className="relative py-16 sm:py-20">
          <div className="grid gap-10 lg:grid-cols-2 lg:items-center">
            <div data-reveal>
              <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary-300">
                How sessions work
              </span>
              <h2 className="mt-3 font-display text-3xl font-semibold tracking-tight text-balance">
                A session your front end never touches
              </h2>
              <p className="mt-4 text-[15px] leading-relaxed text-ink-300">
                The session token is delivered as an HttpOnly cookie — JavaScript cannot read
                it, so XSS cannot steal it. The CSRF value needed for state changes is derived
                from that same session and exchanged through the login response, keeping the
                two concerns separate.
              </p>
              <Link
                to="/contact"
                className="mt-6 inline-flex h-10.5 items-center justify-center gap-2 rounded-lg bg-primary-600 px-5 text-sm font-semibold text-white transition-colors hover:bg-primary-500"
              >
                Ask us about your requirements
              </Link>
            </div>

            <div data-reveal style={{ ['--reveal-delay' as string]: '120ms' }} className="rounded-3xl border border-white/10 bg-white/5 p-8 backdrop-blur-sm">
              <h3 className="text-sm font-semibold text-white">The request lifecycle</h3>
              <ol className="mt-5 space-y-4">
                {[
                  ['01', 'Login', 'Credentials checked, session minted, HttpOnly cookie set.'],
                  ['02', 'Every action', 'CSRF token verified against the session before any handler runs.'],
                  ['03', 'Every change', 'Actor, action, entity, and timestamp appended to the audit log.'],
                  ['04', 'Logout', 'Session invalidated server-side; cookie expired immediately.'],
                ].map(([n, title, body]) => (
                  <li key={n} className="flex gap-4">
                    <span className="font-display text-lg font-semibold text-primary-300">{n}</span>
                    <div>
                      <p className="text-sm font-medium text-white">{title}</p>
                      <p className="mt-0.5 text-[13px] leading-relaxed text-ink-300">{body}</p>
                    </div>
                  </li>
                ))}
              </ol>
            </div>
          </div>
        </Container>
      </section>

      <Section className="pb-24 text-center">
        <Container>
          <h2 className="mx-auto max-w-xl font-display text-3xl font-semibold tracking-tight text-balance text-ink-950" data-reveal>
            Compliance starts with a workflow you can prove
          </h2>
          <p className="mx-auto mt-4 max-w-lg text-[15px] leading-relaxed text-ink-500" data-reveal>
            Every approval, verification, and access change is a record. If your auditors ask
            “show me what happened,” DocuFlow shows them.
          </p>
          <div className="mt-8" data-reveal>
            <Link
              to="/login"
              className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary-600 px-6 text-[15px] font-medium text-white shadow-sm transition-colors hover:bg-primary-700"
            >
              Open the console
            </Link>
          </div>
        </Container>
      </Section>
    </div>
  );
}
