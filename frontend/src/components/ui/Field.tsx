import {
  forwardRef,
  useId,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react';
import { cn } from '@/lib/cn';

interface FieldShellProps {
  label?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
  htmlFor?: string;
}

function FieldShell({ label, hint, error, required, children, htmlFor }: FieldShellProps) {
  const id = useId();
  const targetId = htmlFor ?? id;
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label htmlFor={targetId} className="text-[13px] font-medium text-ink-700">
          {label}
          {required && <span className="ml-0.5 text-danger-500">*</span>}
        </label>
      )}
      {children}
      {error ? (
        <p className="text-xs text-danger-600">{error}</p>
      ) : hint ? (
        <p className="text-xs text-ink-400">{hint}</p>
      ) : null}
    </div>
  );
}

const baseField =
  'w-full rounded-lg border bg-white px-3 text-sm text-ink-900 placeholder:text-ink-300 ' +
  'transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary-500/30 ' +
  'disabled:cursor-not-allowed disabled:bg-paper-100 disabled:text-ink-400';

interface TextInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  invalid?: boolean;
}

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(function TextInput(
  { label, hint, error, invalid, required, className, ...rest },
  ref,
) {
  // The same id drives the <label htmlFor> and the control so the label is
  // associated (accessibility + getByLabel selectors) even when no id prop
  // is passed by the caller.
  const fallbackId = useId();
  const inputId = rest.id ?? fallbackId;
  return (
    <FieldShell label={label} hint={hint} error={error} required={required} htmlFor={inputId}>
      <input
        ref={ref}
        id={inputId}
        aria-invalid={invalid || Boolean(error)}
        className={cn(
          baseField,
          'h-9.5',
          invalid || error
            ? 'border-danger-400 focus:border-danger-400 focus:ring-danger-500/30'
            : 'border-ink-200 focus:border-primary-400',
          className,
        )}
        required={required}
        {...rest}
      />
    </FieldShell>
  );
});

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  hint?: string;
  error?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, hint, error, required, className, children, ...rest },
  ref,
) {
  const fallbackId = useId();
  const selectId = rest.id ?? fallbackId;
  return (
    <FieldShell label={label} hint={hint} error={error} required={required} htmlFor={selectId}>
      <select
        ref={ref}
        id={selectId}
        className={cn(
          baseField,
          'h-9.5 cursor-pointer appearance-none bg-[url("data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns=%22http://www.w3.org/2000/svg%22%20width=%2216%22%20height=%2216%22%20viewBox=%220%200%2024%2024%22%20fill=%22none%22%20stroke=%22%2364748b%22%20stroke-width=%222%22%20stroke-linecap=%22round%22%20stroke-linejoin=%22round%22%3E%3Cpath%20d=%22m6%209%206%206%206-6%22/%3E%3C/svg%3E")] bg-[position:right_10px_center] bg-no-repeat pr-9',
          error ? 'border-danger-400' : 'border-ink-200 focus:border-primary-400',
          className,
        )}
        required={required}
        {...rest}
      >
        {children}
      </select>
    </FieldShell>
  );
});

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  hint?: string;
  error?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { label, hint, error, required, className, ...rest },
  ref,
) {
  const fallbackId = useId();
  const textareaId = rest.id ?? fallbackId;
  return (
    <FieldShell label={label} hint={hint} error={error} required={required} htmlFor={textareaId}>
      <textarea
        ref={ref}
        id={textareaId}
        className={cn(
          baseField,
          'min-h-24 resize-y py-2',
          error ? 'border-danger-400' : 'border-ink-200 focus:border-primary-400',
          className,
        )}
        required={required}
        {...rest}
      />
    </FieldShell>
  );
});
