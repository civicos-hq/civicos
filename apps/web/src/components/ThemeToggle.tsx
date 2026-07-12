import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, ChevronDown, Monitor, Moon, Sun } from 'lucide-react';

type Variant = 'light' | 'dark';
type ThemeChoice = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'civicos-theme';

/**
 * Reads the saved theme choice. Anything invalid, missing, or explicitly
 * `system` resolves to `system` so the OS preference wins. Returns
 * `system` on SSR / non-browser environments (defensive — this app is
 * SPA-only but the guard is cheap).
 */
function readSaved(): ThemeChoice {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === 'light' || v === 'dark') return v;
  } catch {
    // ignore
  }
  return 'system';
}

/**
 * Resolves a choice to the actual visual mode (`light` | `dark`) by
 * consulting the OS preference when the choice is `system`.
 */
function resolveMode(choice: ThemeChoice): 'light' | 'dark' {
  if (choice === 'light' || choice === 'dark') return choice;
  if (typeof window === 'undefined' || !window.matchMedia) return 'dark';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * Applies the resolved mode to <html data-theme="…">. The pre-paint
 * script in index.html handles the initial write before React mounts;
 * this function keeps the attribute in sync when the user changes
 * their choice at runtime.
 */
function applyMode(mode: 'light' | 'dark') {
  document.documentElement.setAttribute('data-theme', mode);
}

/**
 * ThemeToggle — three-state theme picker (System / Light / Dark).
 * Structured to mirror LanguageSwitcher so both controls read as one
 * family in the top nav. Choices persist to localStorage under
 * `civicos-theme`. When "system" is active, this component re-applies
 * the OS preference on the fly if the OS setting flips.
 */
export function ThemeToggle({ variant = 'light' }: { variant?: Variant }) {
  const { t } = useTranslation();
  const [choice, setChoice] = useState<ThemeChoice>(() => readSaved());
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement | null>(null);
  const resolved = resolveMode(choice);

  // Apply on every choice change. The pre-paint script already handled
  // the initial mount so there's no first-frame flicker.
  useEffect(() => {
    applyMode(resolved);
  }, [resolved]);

  // When the user picked "system", listen for OS-level changes and
  // re-apply. Skipped for explicit light/dark choices — those are
  // sticky until the user changes them.
  useEffect(() => {
    if (choice !== 'system' || typeof window === 'undefined' || !window.matchMedia) return;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => applyMode(media.matches ? 'dark' : 'light');
    media.addEventListener('change', handler);
    return () => media.removeEventListener('change', handler);
  }, [choice]);

  // Close on outside click + Escape.
  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDocClick);
      document.removeEventListener('keydown', onKey);
    };
  }, []);

  function pick(next: ThemeChoice) {
    setChoice(next);
    try {
      if (next === 'system') localStorage.removeItem(STORAGE_KEY);
      else localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // ignore write failures — non-fatal, the in-memory state still updates
    }
    setOpen(false);
  }

  const OPTIONS: Array<{ value: ThemeChoice; label: string; Icon: typeof Sun }> = [
    { value: 'system', label: t('theme.system'), Icon: Monitor },
    { value: 'light', label: t('theme.light'), Icon: Sun },
    { value: 'dark', label: t('theme.dark'), Icon: Moon },
  ];

  const TriggerIcon = choice === 'light' ? Sun : choice === 'dark' ? Moon : Monitor;

  return (
    <div className={`lang ${variant === 'dark' ? 'lang--dark' : 'lang--light'}`} ref={wrapRef}>
      <button
        type="button"
        className="lang-btn"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={t('theme.label', { current: t(`theme.${choice}`) })}
        onClick={() => setOpen((v) => !v)}
      >
        <TriggerIcon className="h-4 w-4" aria-hidden="true" />
        <span className="lang-btn-code">{resolved === 'dark' ? 'DK' : 'LT'}</span>
        <ChevronDown className="h-3.5 w-3.5 lang-btn-chev" aria-hidden="true" />
      </button>

      {open && (
        <ul className="lang-menu" role="listbox" aria-label={t('theme.label', { current: '' })}>
          {OPTIONS.map(({ value, label, Icon }) => {
            const active = value === choice;
            return (
              <li key={value} role="option" aria-selected={active}>
                <button type="button" className="lang-item" onClick={() => pick(value)}>
                  <Icon className="h-4 w-4 lang-item-code" aria-hidden="true" />
                  <span className="lang-item-label">{label}</span>
                  {active && <Check className="h-3.5 w-3.5 lang-item-check" aria-hidden="true" />}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
