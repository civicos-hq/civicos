import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Globe, Check, ChevronDown } from 'lucide-react';
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from '../i18n';

type Variant = 'light' | 'dark';

const LABELS: Record<SupportedLanguage, string> = {
  en: 'English',
  ig: 'Igbo',
  yo: 'Yorùbá',
  ha: 'Hausa',
  pcm: 'Pidgin',
};

const SHORT: Record<SupportedLanguage, string> = {
  en: 'EN',
  ig: 'IG',
  yo: 'YO',
  ha: 'HA',
  pcm: 'PCM',
};

function normalize(lng: string): SupportedLanguage {
  const base = lng.split('-')[0].toLowerCase();
  return (SUPPORTED_LANGUAGES as readonly string[]).includes(base)
    ? (base as SupportedLanguage)
    : 'en';
}

export function LanguageSwitcher({ variant = 'light' }: { variant?: Variant }) {
  const { i18n, t } = useTranslation();
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement | null>(null);
  const current = normalize(i18n.language || 'en');

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

  function choose(lng: SupportedLanguage) {
    i18n.changeLanguage(lng);
    setOpen(false);
  }

  return (
    <div className={`lang ${variant === 'dark' ? 'lang--dark' : 'lang--light'}`} ref={wrapRef}>
      <button
        type="button"
        className="lang-btn"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={t('language.label')}
        onClick={() => setOpen((v) => !v)}
      >
        <Globe className="h-4 w-4" aria-hidden="true" />
        <span className="lang-btn-code">{SHORT[current]}</span>
        <ChevronDown className="h-3.5 w-3.5 lang-btn-chev" aria-hidden="true" />
      </button>

      {open && (
        <ul className="lang-menu" role="listbox" aria-label={t('language.label')}>
          {SUPPORTED_LANGUAGES.map((code) => {
            const active = code === current;
            return (
              <li key={code} role="option" aria-selected={active}>
                <button type="button" className="lang-item" onClick={() => choose(code)}>
                  <span className="lang-item-code">{SHORT[code]}</span>
                  <span className="lang-item-label">{LABELS[code]}</span>
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
