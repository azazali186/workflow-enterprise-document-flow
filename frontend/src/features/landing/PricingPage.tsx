import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Check, Sparkles } from 'lucide-react';
import { Container, Section, SectionHeader } from '@/components/landing/Section';
import { useReveal } from '@/hooks/useReveal';
import { cn } from '@/lib/cn';

interface Tier {
  name: string;
  tagline: string;
  monthly: number | null;
  annual: number | null;
  cta: string;
  highlighted?: boolean;
  features: string[];
}

const TIERS: Tier[] = [
  {
    name: 'Starter',
    tagline: 'For small teams getting documents under control.',
    monthly: 0,
    annual: 0,
    cta: 'Start free',
    features: [
      'Up to 10 users',
      'Documents & categories',
      'Single-level approvals',
      'Version history',
      'Standard audit trail',
    ],
  },
  {
    name: 'Growth',
    tagline: 'For teams that need real workflow control.',
    monthly: 29,
    annual: 24,
    cta: 'Start 14-day trial',
    highlighted: true,
    features: [
      'Unlimited users',
      'Multi-level approval chains',
      'Independent verification track',
      'Templates & roles',
      'Document-level access grants',
      'Full audit & login logs',
      'Real-time console updates',
    ],
  },
  {
    name: 'Enterprise',
    tagline: 'For organizations with compliance requirements.',
    monthly: null,
    annual: null,
    cta: 'Contact sales',
    features: [
      'Everything in Growth',
      'Custom roles & permissions',
      'S3-compatible storage options',
      'Extended retention policies',
      'Dedicated support & onboarding',
      'SSO-ready session control',
    ],
  },
];

export function PricingPage() {
  const revealRef = useReveal<HTMLDivElement>();
  const [annual, setAnnual] = useState(true);

  const priceFor = (tier: Tier) => {
    if (tier.monthly === null) return 'Custom';
    if (tier.monthly === 0) return '$0';
    return annual ? `$${tier.annual}` : `$${tier.monthly}`;
  };

  const periodFor = (tier: Tier) => {
    if (tier.monthly === null) return '';
    if (tier.monthly === 0) return 'forever';
    return annual ? 'per user / month, billed yearly' : 'per user / month';
  };

  return (
    <div ref={revealRef}>
      <Section className="pb-8 pt-16 sm:pt-20">
        <Container>
          <SectionHeader
            center
            eyebrow="Pricing"
            title="Simple pricing that scales with your workflow"
            description="Start free, upgrade when your approval chains need to. No hidden fees, no per-document charges."
          />

          {/* Billing toggle */}
          <div className="mt-10 flex items-center justify-center gap-3" data-reveal>
            <span className={cn('text-sm font-medium transition-colors', !annual ? 'text-ink-900' : 'text-ink-400')}>
              Monthly
            </span>
            <button
              onClick={() => setAnnual((v) => !v)}
              role="switch"
              aria-checked={annual}
              aria-label="Toggle annual billing"
              className={cn(
                'relative h-6.5 w-12 rounded-full transition-colors duration-200 cursor-pointer',
                annual ? 'bg-primary-600' : 'bg-ink-200',
              )}
            >
              <span
                className={cn(
                  'absolute top-0.5 size-5.5 rounded-full bg-white shadow-sm transition-all duration-200',
                  annual ? 'left-6' : 'left-0.5',
                )}
              />
            </button>
            <span className={cn('text-sm font-medium transition-colors', annual ? 'text-ink-900' : 'text-ink-400')}>
              Annual
              <span className="ml-1.5 rounded-full bg-success-50 px-2 py-0.5 text-[11px] font-semibold text-success-600">
                Save ~17%
              </span>
            </span>
          </div>
        </Container>
      </Section>

      <Section className="pt-10">
        <Container>
          <div className="grid gap-6 lg:grid-cols-3">
            {TIERS.map((tier, i) => (
              <div
                key={tier.name}
                data-reveal
                style={{ ['--reveal-delay' as string]: `${i * 90}ms` }}
                className={cn(
                  'relative flex flex-col rounded-3xl border p-7 transition-all duration-200',
                  tier.highlighted
                    ? 'border-primary-300 bg-ink-950 text-white shadow-pop lg:-translate-y-2'
                    : 'border-ink-200/70 bg-white',
                )}
              >
                {tier.highlighted && (
                  <span className="absolute -top-3 left-1/2 flex -translate-x-1/2 items-center gap-1.5 rounded-full bg-primary-600 px-3 py-1 text-[11px] font-semibold text-white shadow-sm">
                    <Sparkles className="size-3" aria-hidden />
                    Most popular
                  </span>
                )}

                <h2 className={cn('font-display text-xl font-semibold tracking-tight', tier.highlighted ? 'text-white' : 'text-ink-950')}>
                  {tier.name}
                </h2>
                <p className={cn('mt-1.5 text-[13px] leading-relaxed', tier.highlighted ? 'text-ink-300' : 'text-ink-500')}>
                  {tier.tagline}
                </p>

                <div className="mt-6 flex items-baseline gap-1.5">
                  <span className={cn('font-display text-4xl font-semibold tracking-tight', tier.highlighted ? 'text-white' : 'text-ink-950')}>
                    {priceFor(tier)}
                  </span>
                  {periodFor(tier) && (
                    <span className={cn('text-xs', tier.highlighted ? 'text-ink-400' : 'text-ink-400')}>
                      {periodFor(tier)}
                    </span>
                  )}
                </div>

                <ul className="mt-6 flex-1 space-y-3">
                  {tier.features.map((f) => (
                    <li key={f} className="flex items-start gap-2.5">
                      <Check
                        className={cn('mt-0.5 size-4 shrink-0', tier.highlighted ? 'text-primary-300' : 'text-primary-600')}
                        aria-hidden
                      />
                      <span className={cn('text-sm', tier.highlighted ? 'text-ink-200' : 'text-ink-700')}>{f}</span>
                    </li>
                  ))}
                </ul>

                <Link
                  to="/contact"
                  className={cn(
                    'mt-7 inline-flex h-10.5 items-center justify-center rounded-lg text-sm font-semibold transition-colors',
                    tier.highlighted
                      ? 'bg-primary-600 text-white hover:bg-primary-500'
                      : 'border border-ink-200 bg-white text-ink-800 hover:bg-paper-100',
                  )}
                >
                  {tier.cta}
                </Link>
              </div>
            ))}
          </div>

          <p className="mt-10 text-center text-sm text-ink-400" data-reveal>
            All plans include the full audit trail and secure HttpOnly session handling.
            Questions? <Link to="/contact" className="font-medium text-primary-600 hover:text-primary-700">Talk to us</Link>.
          </p>
        </Container>
      </Section>
    </div>
  );
}
