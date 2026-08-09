import { Link } from 'react-router-dom';
import {
  ArrowRight,
  BadgeCheck,
  Bell,
  CheckCircle2,
  FileText,
  FolderTree,
  History,
  KeyRound,
  ScrollText,
  ShieldCheck,
  Table2,
  Users,
  Workflow,
} from 'lucide-react';
import { Container, Section, SectionHeader } from '@/components/landing/Section';
import { useReveal } from '@/hooks/useReveal';
import { cn } from '@/lib/cn';

interface FeatureRow {
  eyebrow: string;
  title: string;
  body: string;
  points: string[];
  icon: typeof FileText;
  dark?: boolean;
}

const ROWS: FeatureRow[] = [
  {
    eyebrow: 'Documents',
    title: 'Every document has a number, a place, and a path',
    body: 'Documents are first-class objects — numbered, categorized, tagged, and tracked through a lifecycle you control. The pipeline starts the moment a document is registered.',
    points: [
      'Document numbers and metadata from day one',
      'Categories and tags for organization that scales',
      'Ownership, status, and size tracked automatically',
      'Search that covers titles, tags, and metadata',
    ],
    icon: FileText,
  },
  {
    eyebrow: 'Approvals',
    title: 'Approval chains that actually mean something',
    body: 'Build multi-level approval chains with explicit approvers. Each level gates the next, decisions are recorded the moment they are made, and nothing moves without the right sign-off.',
    points: [
      'Multi-level chains with named approvers',
      'Independent approve / reject decisions with comments',
      'Per-level status visible at a glance',
      'Real-time events push updates to the console',
    ],
    icon: Workflow,
    dark: true,
  },
  {
    eyebrow: 'Verification',
    title: 'A verification step that is not a rubber stamp',
    body: 'Verification runs as a separate track from approval, so the person who approves never silently verifies. Documents are checked before they can be approved.',
    points: [
      'Independent verification before approval',
      'Method and notes recorded per verification',
      'Rejected documents return to draft with full context',
      'Verification history preserved in the audit trail',
    ],
    icon: BadgeCheck,
  },
  {
    eyebrow: 'Templates',
    title: 'Start from a standard, not a blank page',
    body: 'Templates carry the structure your organization already agreed on — slugs, descriptions, and content — so every new document starts consistent.',
    points: [
      'Slug-based templating for predictable references',
      'Standardized structure and metadata',
      'Teams can ship their own templates',
      'No drift between documents of the same kind',
    ],
    icon: Table2,
    dark: true,
  },
  {
    eyebrow: 'Access & permissions',
    title: 'Fine-grained control over who sees what',
    body: 'Role-based permissions gate every action in the console, and document-level access grants go further — read, write, or approve, per user or per role.',
    points: [
      'RBAC across every console action',
      'Document-level grants for read, write, and approve',
      'Grant to a single user or an entire role',
      'Revoke instantly, fully audited',
    ],
    icon: KeyRound,
  },
  {
    eyebrow: 'Audit & records',
    title: 'An append-only trail your compliance team will love',
    body: 'Logins, logouts, permissions, verifications, approvals, access changes — every meaningful action is written to an immutable audit log with actor, action, and timestamp.',
    points: [
      'Full login and logout history',
      'Entity-level audit records with actor and action',
      'Traceable from document to decision',
      'Export-ready, queryable records',
    ],
    icon: ScrollText,
    dark: true,
  },
];

const MINI_GRID = [
  { icon: FolderTree, title: 'Categories', body: 'Organize documents into a hierarchy that matches how your teams work.' },
  { icon: History, title: 'Versions', body: 'Every save is a version. Browse, compare, and restore any snapshot.' },
  { icon: Users, title: 'Users & roles', body: 'Manage people, roles, and permissions from one screen with zero drift.' },
  { icon: Bell, title: 'Live updates', body: 'The console updates in real time as decisions and verifications land.' },
  { icon: ShieldCheck, title: 'Secure sessions', body: 'HttpOnly session cookies with CSRF protection on every state change.' },
  { icon: CheckCircle2, title: 'Storage', body: 'Files stored in S3-compatible object storage, tracked per document.' },
];

export function FeaturesPage() {
  const revealRef = useReveal<HTMLDivElement>();

  return (
    <div ref={revealRef}>
      <Section className="pb-8 pt-16 sm:pt-20">
        <Container>
          <SectionHeader
            center
            eyebrow="Features"
            title="Built around the document, not the inbox"
            description="DocuFlow gives every document a controlled lifecycle with accountability at each step — so your team agrees on what is true, and when."
          />
        </Container>
      </Section>

      {ROWS.map((row) => (
        <section
          key={row.title}
          className={cn(
            'py-16 sm:py-20',
            row.dark ? 'bg-ink-950 text-white' : 'border-y border-ink-200/60 bg-white',
          )}
        >
          <Container>
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-16">
              <div data-reveal className={cn(row.dark && 'lg:order-2')}>
                <span
                  className={cn(
                    'text-[11px] font-semibold uppercase tracking-[0.18em]',
                    row.dark ? 'text-primary-300' : 'text-primary-600',
                  )}
                >
                  {row.eyebrow}
                </span>
                <h2
                  className={cn(
                    'mt-3 font-display text-3xl font-semibold tracking-tight text-balance',
                    row.dark ? 'text-white' : 'text-ink-950',
                  )}
                >
                  {row.title}
                </h2>
                <p
                  className={cn(
                    'mt-4 text-[15px] leading-relaxed',
                    row.dark ? 'text-ink-300' : 'text-ink-500',
                  )}
                >
                  {row.body}
                </p>
                <ul className="mt-6 space-y-3">
                  {row.points.map((p) => (
                    <li key={p} className="flex items-start gap-3">
                      <CheckCircle2
                        className={cn('mt-0.5 size-4.5 shrink-0', row.dark ? 'text-primary-300' : 'text-primary-600')}
                        aria-hidden
                      />
                      <span className={cn('text-sm', row.dark ? 'text-ink-200' : 'text-ink-700')}>{p}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Visual panel */}
              <div
                data-reveal
                style={{ ['--reveal-delay' as string]: '120ms' }}
                className={cn(
                  'rounded-3xl border p-8 sm:p-10',
                  row.dark ? 'lg:order-1' : '',
                  row.dark
                    ? 'border-white/10 bg-white/5 backdrop-blur-sm'
                    : 'border-ink-200/70 bg-paper-50',
                )}
              >
                <div
                  className={cn(
                    'flex size-14 items-center justify-center rounded-2xl',
                    row.dark ? 'bg-primary-500/15 text-primary-300' : 'bg-primary-600 text-white',
                  )}
                >
                  <row.icon className="size-7" aria-hidden />
                </div>
                <h3
                  className={cn(
                    'mt-6 font-display text-xl font-semibold tracking-tight',
                    row.dark ? 'text-white' : 'text-ink-900',
                  )}
                >
                  {row.eyebrow} in practice
                </h3>
                <p
                  className={cn(
                    'mt-3 text-sm leading-relaxed',
                    row.dark ? 'text-ink-300' : 'text-ink-500',
                  )}
                >
                  The console keeps every stage visible — who did what, when, and with which
                  document. No more “where is this?” emails.
                </p>
                <div
                  className={cn(
                    'mt-6 h-1.5 w-full overflow-hidden rounded-full',
                    row.dark ? 'bg-white/10' : 'bg-ink-100',
                  )}
                  aria-hidden
                >
                  <div className="h-full w-3/4 rounded-full bg-primary-500" />
                </div>
                <div
                  className={cn(
                    'mt-3 flex items-center justify-between text-xs tabular',
                    row.dark ? 'text-ink-400' : 'text-ink-400',
                  )}
                >
                  <span>Draft</span>
                  <span>Verified</span>
                  <span>Approved</span>
                </div>
              </div>
            </div>
          </Container>
        </section>
      ))}

      {/* Mini capability grid */}
      <Section className="bg-white">
        <Container>
          <SectionHeader
            center
            eyebrow="And more"
            title="The details that keep a workflow honest"
          />
          <div className="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {MINI_GRID.map((f, i) => (
              <div
                key={f.title}
                data-reveal
                style={{ ['--reveal-delay' as string]: `${(i % 3) * 80}ms` }}
                className="rounded-2xl border border-ink-200/70 bg-paper-50 p-6 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-card"
              >
                <f.icon className="size-5 text-primary-600" aria-hidden />
                <h3 className="mt-3.5 text-[15px] font-semibold text-ink-900">{f.title}</h3>
                <p className="mt-1.5 text-sm leading-relaxed text-ink-500">{f.body}</p>
              </div>
            ))}
          </div>

          <div className="mt-12 text-center" data-reveal>
            <Link
              to="/login"
              className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary-600 px-6 text-[15px] font-medium text-white shadow-sm transition-colors hover:bg-primary-700"
            >
              See it in the console
              <ArrowRight className="size-4" aria-hidden />
            </Link>
          </div>
        </Container>
      </Section>
    </div>
  );
}
