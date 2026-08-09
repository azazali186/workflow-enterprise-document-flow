import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Archive } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { TextInput } from '@/components/ui/Field';
import { login } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';
import { errorMessage } from '@/hooks/useToast';

export function LoginPage() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const status = useAppSelector((s) => s.auth.status);
  const error = useAppSelector((s) => s.auth.error);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  const loading = status === 'loading';

  useEffect(() => {
    if (status === 'authenticated') navigate('/app', { replace: true });
  }, [status, navigate]);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    if (!email.trim() || !password) {
      setFormError('Enter your email and password.');
      return;
    }
    try {
      await dispatch(login({ email: email.trim(), password })).unwrap();
    } catch (err) {
      setFormError(errorMessage(err, 'Login failed. Check your credentials.'));
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center bg-paper-50 px-4">
      <div className="grid w-full max-w-4xl overflow-hidden rounded-2xl border border-ink-200/70 bg-white shadow-pop lg:grid-cols-2">
        {/* Editorial panel */}
        <div className="relative hidden flex-col justify-between bg-ink-950 p-10 text-white lg:flex">
          <div className="flex items-center gap-2.5">
            <div className="flex size-9 items-center justify-center rounded-lg bg-primary-600">
              <Archive className="size-5" aria-hidden />
            </div>
            <span className="font-display text-lg font-semibold tracking-tight">DocuFlow</span>
          </div>
          <div>
            <p className="font-display text-3xl font-medium leading-snug tracking-tight">
              Every document,
              <br />
              <span className="text-primary-300">in its right place.</span>
            </p>
            <p className="mt-4 max-w-sm text-sm leading-relaxed text-ink-300">
              Create, verify, approve, and archive digital documents with a controlled workflow —
              from draft to done.
            </p>
          </div>
          <p className="text-xs text-ink-400">Secure · Audited · Enterprise-grade</p>
        </div>

        {/* Form panel */}
        <div className="p-8 sm:p-10">
          <div className="mb-8 lg:hidden">
            <div className="flex items-center gap-2.5">
              <div className="flex size-9 items-center justify-center rounded-lg bg-primary-600 text-white">
                <Archive className="size-5" aria-hidden />
              </div>
              <span className="font-display text-lg font-semibold tracking-tight text-ink-950">DocuFlow</span>
            </div>
          </div>

          <h1 className="font-display text-2xl font-semibold tracking-tight text-ink-950">Welcome back</h1>
          <p className="mt-1.5 text-sm text-ink-500">Sign in to the admin console to continue.</p>

          <form onSubmit={onSubmit} className="mt-8 space-y-5" noValidate>
            <TextInput
              label="Email address"
              type="email"
              autoComplete="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
            <TextInput
              label="Password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />

            {(formError || error) && (
              <div className="rounded-lg border border-danger-200 bg-danger-50 px-3.5 py-2.5 text-sm text-danger-600 animate-fade-in">
                {formError ?? error}
              </div>
            )}

            <Button type="submit" fullWidth size="lg" loading={loading}>
              {loading ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
