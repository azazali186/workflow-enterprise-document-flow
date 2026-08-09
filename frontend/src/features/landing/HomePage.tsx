import { Link } from 'react-router-dom';
import {
  ArrowRight,
  BadgeCheck,
  FileText,
  Fingerprint,
  History,
  KeyRound,
  ShieldCheck,
  Table2,
  Workflow,
} from 'lucide-react';
import { Container, Eyebrow, Section, SectionHeader } from '@/components/landing/Section';
import { HeroPipeline } from './HeroPipeline';
import { useReveal } from '@/hooks/useReveal';

const FEATURES = [
  {
    icon: FileText,
    title: 'Controlled documents',
    body: 'Register every document with its own number, metadata, and tags — then let the workflow drive it from draft to done.',
  },
  {
    icon: Workflow,
    title: 'Multi-step approvals',
    body: 'Chain approvals across levels and users. Each approver sees exactly what is pending for them, nothing else.',
  },
  {
    icon: BadgeCheck,
    title: 'Independent verification',
    body: 'A separate verification track confirms authenticity before a document is approved — no rubber-stamping.',
  },
  {
    icon: Table2,
    title: 'Reusable templates',
    body: 'Start from a template instead of a blank page. Standardize slugs, structure, and metadata across teams.',
  },
  {
    icon: History,
    title: 'Full version history',
    body: 'Every saved change is a version. Compare, restore, and see exactly who changed what and when.',
  },
  {
    icon: KeyRound,
    title: 'Granular access control',
    body: 'Grant read, write, or approve access per user or per role. Nothing is visible that should not be.',
  },
];

const STEPS = [
  {
    n: '01',
    title: 'Create',
    body: 'Upload or register a document — title, category, tags, and the workflow starts automatically.',
  },
  {
    n: '02',
    title: 'Verify',
    body: 'An independent verifier confirms the document is authentic and complete before it moves on.',
  },
  {
    n: '03',
    title: 'Approve',
    body: 'Approvers review and decide at each level of the chain. Every decision is recorded instantly.',
  },
  {
    n: '04',
    title: 'Archive',
    body: 'Approved documents are sealed and stored with their full audit trail, ready for retrieval.',
  },
];

const STATS = [
  { value: '4', label: 'lifecycle stages — create, verify, approve, archive' },
  { value: '100%', label: 'of decisions written to the audit log' },
  { value: '24h', label: 'rolling session — no constant re-login' },
];

export function HomePage() {
  const revealRef = useReveal<HTMLDivElement>();

  return (
    <div ref={revealRef}>
      {/* ============ Hero ============ */}
      <section className="relative overflow-hidden">
        <div
          className="pointer-events-none absolute inset-x-0 top-0 h-[26rem] bg-[radial-gradient(60%_60%_at_50%_-10%,rgb(92_102_230/0.10),transparent)]"
          aria-hidden
        />
        <Container className="relative pt-16 pb-20 text-center sm:pt-24 sm:pb-28">
          <div data-reveal className="mx-auto max-w-3xl">
            <span className="inline-flex items-center gap-2 rounded-full border border-primary-200 bg-primary-50 px-3.5 py-1.5 text-[12.5px] font-medium text-primary-700">
              <ShieldCheck className="size-3.5" aria-hidden />
              Controlled document workflow for modern teams
            </span>
            <h1 className="mt-6 font-display text-4xl font-semibold tracking-tight text-balance text-ink-950 sm:text-6xl">
              Every document,
              <br />
              <span className="hero-flourish">in its right place.</span>
            </h1>
            <p className="mx-auto mt-6 max-w-xl text-base leading-relaxed text-ink-500 sm:text-lg">
              Create, verify, approve, and archive digital documents with a controlled
              workflow — every step signed, tracked, and fully auditable.
            </p>
            <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Link
                to="/login"
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-primary-600 px-6 text-[15px] font-medium text-white shadow-sm transition-all hover:bg-primary-700 sm:w-auto"
              >
                Open the console
                <ArrowRight className="size-4" aria-hidden />
              </Link>
              <Link
                to="/features"
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg border border-ink-200 bg-white px-6 text-[15px] font-medium text-ink-700 transition-colors hover:bg-paper-100 sm:w-auto"
              >
                Explore features
              </Link>
            </div>
          </div>

          <div className="mt-16 sm:mt-20" data-reveal>
            <HeroPipeline />
          </div>
        </Container>
      </section>

      {/* ============ Feature grid ============ */}
      <Section className="border-y border-ink-200/60 bg-white">
        <Container>
          <SectionHeader
            center
            eyebrow="Capabilities"
            title="Everything a document needs to be trusted"
            description="From the first draft to the sealed archive, DocuFlow keeps control, context, and accountability at every step."
          />
          <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((f, i) => (
              <div
                key={f.title}
                data-reveal
                style={{ ['--reveal-delay' as string]: `${(i % 3) * 90}ms` }}
                className="group rounded-2xl border border-ink-200/70 bg-paper-50 p-6 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-card"
              >
                <div className="flex size-10 items-center justify-center rounded-xl bg-primary-50 text-primary-600 transition-colors group-hover:bg-primary-600 group-hover:text-white">
                  <f.icon className="size-5" aria-hidden />
                </div>
                <h3 className="mt-4 font-display text-lg font-semibold tracking-tight text-ink-900">{f.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-ink-500">{f.body}</p>
              </div>
            ))}
          </div>
        </Container>
      </Section>

      {/* ============ How it works ============ */}
      <Section>
        <Container>
          <SectionHeader
            center
            eyebrow="How it works"
            title="A lifecycle, not a folder"
            description="Four stages, one controlled path. Each stage gates the next — nothing skips the queue."
          />
          <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {STEPS.map((s, i) => (
              <div
                key={s.n}
                data-reveal
                style={{ ['--reveal-delay' as string]: `${i * 90}ms` }}
                className="relative rounded-2xl border border-ink-200/70 bg-white p-6"
              >
                <span className="font-display text-3xl font-semibold text-primary-200">{s.n}</span>
                <h3 className="mt-3 font-display text-lg font-semibold tracking-tight text-ink-900">{s.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-ink-500">{s.body}</p>
                {i < STEPS.length - 1 && (
                  <ArrowRight
                    className="absolute right-4 top-6 hidden size-4 text-ink-300 lg:block"
                    aria-hidden
                  />
                )}
              </div>
            ))}
          </div>
        </Container>
      </Section>

      {/* ============ Stats / trust ============ */}
      <section className="relative overflow-hidden bg-ink-950 text-white">
        <div className="ink-grid absolute inset-0" aria-hidden />
        <Container className="relative py-16 sm:py-20">
          <div className="grid gap-10 md:grid-cols-[1fr_1.6fr] md:items-center">
            <div data-reveal>
              <Eyebrow tone="light">Built for accountability</Eyebrow>
              <h2 className="mt-3 font-display text-3xl font-semibold tracking-tight text-balance">
                Every action leaves a trace
              </h2>
              <p className="mt-4 text-[15px] leading-relaxed text-ink-300">
                Login attempts, permissions, verifications, and approvals — all written to an
                append-only audit log your compliance team will love.
              </p>
              <Link
                to="/security"
                className="mt-6 inline-flex items-center gap-1.5 text-sm font-medium text-primary-300 transition-colors hover:text-primary-200"
              >
                Read the security model
                <ArrowRight className="size-4" aria-hidden />
              </Link>
            </div>
            <dl className="grid gap-6 sm:grid-cols-3" data-reveal>
              {STATS.map((s) => (
                <div key={s.label} className="flex flex-col rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur-sm">
                  <dt className="order-2 mt-2 text-[13px] leading-snug text-ink-300">{s.label}</dt>
                  <dd className="order-1 font-display text-4xl font-semibold tracking-tight text-white">{s.value}</dd>
                </div>
              ))}
            </dl>
          </div>
        </Container>
      </section>

      {/* ============ Final CTA ============ */}
      <Section className="pb-24">
        <Container>
          <div
            data-reveal
            className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary-700 via-primary-600 to-primary-800 px-6 py-16 text-center text-white sm:px-12"
          >
            <div className="pointer-events-none absolute inset-0 ink-grid opacity-40" aria-hidden />
            <div className="relative mx-auto max-w-xl">
              <Fingerprint className="mx-auto size-8 text-primary-200" aria-hidden />
              <h2 className="mt-5 font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
                Bring order to your documents
              </h2>
              <p className="mt-4 text-[15px] leading-relaxed text-primary-100">
                Sign in to the console and start a controlled workflow in minutes. No setup
                calls, no paperwork.
              </p>
              <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <Link
                  to="/login"
                  className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-white px-6 text-[15px] font-semibold text-primary-800 shadow-sm transition-colors hover:bg-primary-50 sm:w-auto"
                >
                  Open the console
                  <ArrowRight className="size-4" aria-hidden />
                </Link>
                <Link
                  to="/contact"
                  className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg border border-white/30 px-6 text-[15px] font-medium text-white transition-colors hover:bg-white/10 sm:w-auto"
                >
                  Talk to us
                </Link>
              </div>
            </div>
          </div>
        </Container>
      </Section>

    </div>
  );
}
