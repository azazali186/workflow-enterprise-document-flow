import { useEffect, useId, useRef, useState } from 'react';
import { Check, ChevronsUpDown, Loader2, Search, X } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useOptions } from '@/hooks/useOptions';
import type { OptionItem, OptionKind } from '@/services/options.service';

interface SearchableSelectProps {
  label?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  placeholder?: string;
  disabled?: boolean;
  autoFocus?: boolean;
  /** Accessible name when the control has no visible label. */
  ariaLabel?: string;
  /** Entity the dropdown lists (users, roles, categories, templates, documents). */
  kind: OptionKind;
  /** Selected option id ('' = none). Controlled from the parent form. */
  value: string;
  onChange: (id: string) => void;
  /** Display name when `value` is set externally (e.g. edit mode). */
  valueName?: string;
  /** Show a clear (×) button once a value is selected. */
  allowClear?: boolean;
}

/**
 * A searchable dropdown backed by the shared /options/list endpoint. Type to
 * filter server-side (debounced), arrow keys to navigate, Enter to select.
 * Every form that references another entity by id should use this instead of
 * a raw id input or an unsorted native select.
 */
export function SearchableSelect({
  label,
  hint,
  error,
  required,
  placeholder = 'Search…',
  disabled,
  autoFocus,
  ariaLabel,
  kind,
  value,
  onChange,
  valueName,
  allowClear = true,
}: SearchableSelectProps) {
  const fallbackId = useId();
  const inputId = `searchable-${fallbackId}`;
  const listboxId = `${inputId}-listbox`;

  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const [selectedName, setSelectedName] = useState('');
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Fetch while open (typing/searching) or while a value needs its name
  // resolved (e.g. a pre-set filter id after a page reload).
  const needsName = Boolean(value && !valueName);
  const { data: options = [], isFetching, isError } = useOptions(kind, query, open || needsName);

  // If the parent replaces the selection externally, mirror its label.
  useEffect(() => {
    if (value === '') {
      setSelectedName('');
      return;
    }
    if (valueName) setSelectedName(valueName);
    else {
      const known = options.find((o) => o.id === value);
      if (known) setSelectedName(known.name);
    }
  }, [value, valueName, options]);

  // Close when clicking outside — and drop any in-progress query so the
  // input reverts to the selected option's name instead of stale typed text.
  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQuery('');
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [open]);

  const displayedValue = query !== '' ? query : value ? selectedName || valueName || '' : '';

  const select = (opt: OptionItem) => {
    onChange(opt.id);
    setSelectedName(opt.name);
    setQuery('');
    setOpen(false);
    setHighlighted(0);
  };

  const clear = () => {
    onChange('');
    setSelectedName('');
    setQuery('');
    setOpen(false);
    setHighlighted(0);
    inputRef.current?.focus();
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        if (!open) { setOpen(true); return; }
        setHighlighted((h) => Math.min(h + 1, Math.max(options.length - 1, 0)));
        break;
      case 'ArrowUp':
        e.preventDefault();
        if (!open) { setOpen(true); return; }
        setHighlighted((h) => Math.max(h - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (open && options[highlighted]) select(options[highlighted]);
        break;
      case 'Escape':
        e.preventDefault();
        setQuery('');
        setOpen(false);
        break;
      case 'Tab':
        setQuery('');
        setOpen(false);
        break;
    }
  };

  return (
    <div ref={rootRef} className="flex flex-col gap-1.5">
      {label && (
        <label htmlFor={inputId} className="text-[13px] font-medium text-ink-700">
          {label}
          {required && <span className="ml-0.5 text-danger-500">*</span>}
        </label>
      )}
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-300" aria-hidden />
        <input
          ref={inputRef}
          id={inputId}
          role="combobox"
          aria-label={ariaLabel}
          aria-expanded={open}
          aria-controls={listboxId}
          aria-activedescendant={open && options[highlighted] ? `${listboxId}-${options[highlighted].id}` : undefined}
          aria-autocomplete="list"
          aria-invalid={Boolean(error)}
          autoComplete="off"
          autoFocus={autoFocus}
          disabled={disabled}
          value={displayedValue}
          placeholder={placeholder}
          onChange={(e) => { setQuery(e.target.value); setOpen(true); setHighlighted(0); }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKeyDown}
          className={cn(
            'h-9.5 w-full rounded-lg border bg-white pl-9 pr-16 text-sm text-ink-900 placeholder:text-ink-300',
            'transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-primary-500/30',
            'disabled:cursor-not-allowed disabled:bg-paper-100 disabled:text-ink-400',
            error ? 'border-danger-400 focus:border-danger-400 focus:ring-danger-500/30'
              : 'border-ink-200 focus:border-primary-400',
          )}
        />
        <div className="absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-1">
          {value && allowClear && !disabled && (
            <button
              type="button"
              onClick={clear}
              aria-label="Clear selection"
              className="rounded p-0.5 text-ink-400 transition-colors hover:text-ink-700"
            >
              <X className="size-4" />
            </button>
          )}
          <ChevronsUpDown className={cn('size-4 text-ink-300 transition-transform duration-150', open && 'rotate-180')} aria-hidden />
        </div>

        {open && (
          <div
            id={listboxId}
            role="listbox"
            className="absolute z-30 mt-1.5 max-h-64 w-full overflow-y-auto rounded-xl border border-ink-200 bg-white p-1.5 shadow-pop"
          >
            {isFetching && options.length === 0 ? (
              <div className="flex items-center gap-2 px-3 py-2.5 text-[13px] text-ink-400">
                <Loader2 className="size-3.5 animate-spin" aria-hidden /> Searching…
              </div>
            ) : isError ? (
              <p className="px-3 py-2.5 text-[13px] text-danger-600">Could not load options.</p>
            ) : options.length === 0 ? (
              <p className="px-3 py-2.5 text-[13px] text-ink-400">
                {query ? `No matches for “${query}”.` : 'No options available.'}
              </p>
            ) : (
              <ul className="space-y-0.5">
                {options.map((opt, i) => {
                  const active = i === highlighted;
                  const selected = opt.id === value;
                  return (
                    <li
                      key={opt.id}
                      id={`${listboxId}-${opt.id}`}
                      role="option"
                      aria-selected={selected}
                      onMouseEnter={() => setHighlighted(i)}
                      onMouseDown={(e) => { e.preventDefault(); select(opt); }}
                      className={cn(
                        'flex cursor-pointer items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm transition-colors duration-100',
                        active ? 'bg-primary-50 text-primary-800' : 'text-ink-800',
                        selected && 'font-medium',
                      )}
                    >
                      <span className="truncate">{opt.name}</span>
                      {selected && <Check className="size-4 shrink-0 text-primary-600" aria-hidden />}
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        )}
      </div>
      {error ? (
        <p className="text-xs text-danger-600">{error}</p>
      ) : hint ? (
        <p className="text-xs text-ink-400">{hint}</p>
      ) : null}
    </div>
  );
}
