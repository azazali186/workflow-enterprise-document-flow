import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { ChevronDown, Mail, MessageSquare, Send } from 'lucide-react';
import { Container, Section, SectionHeader } from '@/components/landing/Section';
import { TextInput, Textarea } from '@/components/ui/Field';
import { Button } from '@/components/ui/Button';
import { useToast } from '@/hooks/useToast';
import { useReveal } from '@/hooks/useReveal';
import { cn } from '@/lib/cn';

const FAQS = [
  {
    q: 'How is DocuFlow different from a shared folder?',
    a: 'A shared folder has no order: no lifecycle, no approvals, no audit trail. DocuFlow gives every document a controlled path — create, verify, approve, archive — where every step is recorded.',
  },
  {
    q: 'Can we define our own approval chains?',
    a: 'Yes. Approvals support multi-level chains with named approvers per level. Each level must decide before the document moves on.',
  },
  {
    q: 'Is there an audit trail we can export?',
    a: 'Every meaningful action — logins, permission changes, verifications, approvals — is written to an append-only audit log that you can browse and query from the console.',
  },
  {
    q: 'How are sessions and documents protected?',
    a: 'Sessions use HttpOnly cookies with CSRF protection, and document access is controlled per user and per role. See the Security page for the full model.',
  },
  {
    q: 'Can we self-host?',
    a: 'DocuFlow ships as containers with a Kubernetes deployment manifest, so you can run it in your own environment. Talk to us for the details.',
  },
];

export function ContactPage() {
  const revealRef = useReveal<HTMLDivElement>();
  const toast = useToast();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [company, setCompany] = useState('');
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [openFaq, setOpenFaq] = useState<number | null>(0);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !email.trim() || !message.trim()) {
      toast.error('Please fill in your name, email, and message.');
      return;
    }
    setSending(true);
    // Frontend demo submission — wire to a real endpoint (e.g. /contact) when
    // the backend exposes one. Simulated delay keeps the UX honest.
    await new Promise((r) => setTimeout(r, 700));
    setSending(false);
    toast.success('Message sent', 'Our team will get back to you within one business day.');
    setName('');
    setEmail('');
    setCompany('');
    setMessage('');
  };

  return (
    <div ref={revealRef}>
      <Section className="pb-8 pt-16 sm:pt-20">
        <Container>
          <SectionHeader
            center
            eyebrow="Contact"
            title="Talk to a human about your documents"
            description="Whether you are evaluating DocuFlow or want a deeper look at the security model, we would love to hear from you."
          />
        </Container>
      </Section>

      <Section className="pt-8">
        <Container>
          <div className="grid gap-10 lg:grid-cols-[1.1fr_1fr] lg:gap-16">
            {/* Form */}
            <div data-reveal className="rounded-3xl border border-ink-200/70 bg-white p-7 sm:p-9">
              <h2 className="flex items-center gap-2.5 font-display text-xl font-semibold tracking-tight text-ink-950">
                <MessageSquare className="size-5 text-primary-600" aria-hidden />
                Send us a message
              </h2>
              <form onSubmit={onSubmit} className="mt-7 space-y-5" noValidate>
                <div className="grid gap-5 sm:grid-cols-2">
                  <TextInput
                    label="Name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Ada Lovelace"
                    required
                  />
                  <TextInput
                    label="Work email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="you@company.com"
                    required
                  />
                </div>
                <TextInput
                  label="Company (optional)"
                  value={company}
                  onChange={(e) => setCompany(e.target.value)}
                  placeholder="Acme Inc."
                />
                <Textarea
                  label="Message"
                  rows={5}
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder="Tell us about your workflow and what you need…"
                  required
                />
                <Button type="submit" size="lg" loading={sending} icon={<Send className="size-4" />}>
                  {sending ? 'Sending…' : 'Send message'}
                </Button>
              </form>
            </div>

            {/* FAQ */}
            <div data-reveal style={{ ['--reveal-delay' as string]: '120ms' }}>
              <h2 className="flex items-center gap-2.5 font-display text-xl font-semibold tracking-tight text-ink-950">
                <Mail className="size-5 text-primary-600" aria-hidden />
                Frequently asked
              </h2>
              <div className="mt-6 space-y-3">
                {FAQS.map((faq, i) => {
                  const open = openFaq === i;
                  return (
                    <div
                      key={faq.q}
                      className={cn(
                        'overflow-hidden rounded-2xl border transition-colors duration-200',
                        open ? 'border-primary-200 bg-white' : 'border-ink-200/70 bg-white/60',
                      )}
                    >
                      <button
                        onClick={() => setOpenFaq(open ? null : i)}
                        aria-expanded={open}
                        className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left cursor-pointer"
                      >
                        <span className="text-sm font-semibold text-ink-900">{faq.q}</span>
                        <ChevronDown
                          className={cn('size-4 shrink-0 text-ink-400 transition-transform duration-200', open && 'rotate-180')}
                          aria-hidden
                        />
                      </button>
                      {open && (
                        <p className="px-5 pb-5 text-sm leading-relaxed text-ink-500 animate-fade-in">
                          {faq.a}
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>

              <p className="mt-6 text-sm text-ink-500">
                Prefer to try it first?{' '}
                <Link to="/login" className="font-medium text-primary-600 transition-colors hover:text-primary-700">
                  Open the console
                </Link>
                .
              </p>
            </div>
          </div>
        </Container>
      </Section>
    </div>
  );
}
