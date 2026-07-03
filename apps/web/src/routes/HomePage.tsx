import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Trans, useTranslation } from 'react-i18next';
import {
  ArrowRight,
  Bell,
  CheckCircle2,
  ChevronDown,
  Eye,
  FileText,
  Mail,
  Megaphone,
  MapPin,
  ShieldCheck,
  Users,
} from 'lucide-react';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

export function HomePage() {
  useScrollReveal();
  useScrollToHash();
  return (
    <div className="home-shell">
      <TopNav />
      <Hero />
      <Parties />
      <Articles />
      <Principles />
      <HowItWorks />
      <FAQ />
      <Newsletter />
      <CTA />
      <Footer />
    </div>
  );
}

// Section markers ("SECTION — PARTIES TO THE RECORD") type in glyph by
// glyph as their section enters the viewport. Wraps each character in a
// span with a per-index animation-delay; screen readers get the whole
// string via aria-label and skip the char spans.
function TypedMarker({
  text,
  className = 'home-section-marker',
}: {
  text: string;
  className?: string;
}) {
  return (
    <p className={className} aria-label={text}>
      {[...text].map((ch, i) => (
        <span
          key={`${i}-${ch}`}
          className="marker-char"
          aria-hidden="true"
          style={{ animationDelay: `${i * 22}ms` }}
        >
          {ch === ' ' ? ' ' : ch}
        </span>
      ))}
    </p>
  );
}

// One-stroke cursive flourish under the hero title. Draws left→right after
// the docket cascade lands, as if the public record has just been signed.
// Purely decorative — hidden from screen readers.
function HeroSignature() {
  const { t } = useTranslation();
  return (
    <>
      <svg className="hero-signature" viewBox="0 0 400 44" role="presentation" aria-hidden="true">
        <path
          d="M 4 30
             C 22 8, 40 40, 60 22
             S 96 6, 122 28
             Q 140 42, 162 20
             T 214 26
             C 236 34, 258 12, 288 30
             S 332 8, 356 24
             L 386 20"
        />
      </svg>
      <span className="hero-signature-caption" aria-hidden="true">
        {t('hero.signatureCaption')}
      </span>
    </>
  );
}

// Scrolls to the location.hash target on mount and whenever the hash
// changes. Lets the TopNav's `/#docket`-style links work whether the user
// clicks them while on the homepage or arrives from another route.
function useScrollToHash() {
  const { hash } = useLocation();
  useEffect(() => {
    if (!hash) return;
    const id = hash.slice(1);
    const el = document.getElementById(id);
    if (!el) return;
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    el.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' });
  }, [hash]);
}

function useScrollReveal() {
  useEffect(() => {
    const targets = document.querySelectorAll<HTMLElement>('.reveal');
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduce) {
      targets.forEach((el) => el.classList.add('in-view'));
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.classList.add('in-view');
            io.unobserve(entry.target);
          }
        }
      },
      { threshold: 0.14, rootMargin: '0px 0px -60px 0px' },
    );
    targets.forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, []);
}

export function TopNav() {
  const { t } = useTranslation();
  return (
    <header className="home-nav">
      <Link to="/" className="home-brand" aria-label="CivicOS home">
        <span className="home-brand-mark" aria-hidden="true">
          <img src="/civicos-mark.png" alt="" />
        </span>
        <div className="leading-tight">
          <p className="home-brand-title">CivicOS</p>
          <p className="home-brand-subtitle">{t('nav.brandSubtitle')}</p>
        </div>
      </Link>

      <nav className="home-nav-links" aria-label="Primary">
        <Link to="/#docket">{t('nav.links.docket')}</Link>
        <Link to="/#articles">{t('nav.links.whatItDoes')}</Link>
        <Link to="/#how">{t('nav.links.howItWorks')}</Link>
        <Link to="/#faq">{t('nav.links.faq')}</Link>
      </nav>

      <div className="home-nav-cta">
        <LanguageSwitcher />
        <Link to="/login" className="home-link">
          {t('nav.signIn')}
        </Link>
        <Link to="/register" className="home-btn home-btn-primary">
          {t('nav.register')}
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    </header>
  );
}

function Hero() {
  const { t, i18n } = useTranslation();
  const today = new Date()
    .toLocaleDateString(i18n.language || 'en-GB', {
      weekday: 'short',
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    })
    .toUpperCase();

  return (
    <section className="home-hero">
      <div className="home-hero-orb home-hero-orb--1" aria-hidden="true" />
      <div className="home-hero-orb home-hero-orb--2" aria-hidden="true" />

      <div className="home-hero-body">
        <div className="home-hero-copy">
          <div className="home-hero-masthead">
            <span>
              <span className="pr-cyan-dot" aria-hidden="true" /> {t('hero.masthead')}
            </span>
            <span>{today}</span>
          </div>

          <h1 className="home-hero-title">
            <Trans i18nKey="hero.headline" components={{ em: <em /> }} />
          </h1>

          <HeroSignature />

          <p className="home-hero-sub">{t('hero.sub')}</p>

          <div className="home-hero-cta">
            <Link to="/register" className="home-btn home-btn-primary home-btn-lg">
              {t('hero.ctaPrimary')}
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link to="/login" className="home-btn home-btn-ghost home-btn-lg">
              {t('hero.ctaSecondary')}
            </Link>
          </div>

          <p className="home-trust-strip">
            <ShieldCheck className="h-4 w-4" />
            <span>{t('hero.trust')}</span>
          </p>
        </div>

        <HeroArt />
      </div>

      <Docket />
    </section>
  );
}

// A stylized "Citizens' Register" card that sits beside the hero copy at
// wide widths. Communicates community without a stock photo: 80 citizen
// dots (~20% highlighted by engagement tier), a signature stroke, and a
// stamped seal. Purely decorative — hidden from screen readers. Swap the
// contents for an <img> if you have a real community photograph.
function HeroArt() {
  const { t } = useTranslation();
  const TOTAL = 80;
  const COLS = 8;
  const CELL_W = 40;
  const CELL_H = 34;
  const GRID_ORIGIN_X = 30;
  const GRID_ORIGIN_Y = 80;

  // Deterministic highlights so the composition renders identically each
  // load; index → tier tag. Rough 20% highlight density, spatially spread.
  const highlights: Record<number, 'cyan' | 'mint' | 'lavender'> = {
    4: 'cyan',
    11: 'cyan',
    22: 'cyan',
    37: 'cyan',
    53: 'cyan',
    67: 'cyan',
    8: 'mint',
    17: 'mint',
    31: 'mint',
    45: 'mint',
    58: 'mint',
    72: 'mint',
    14: 'lavender',
    28: 'lavender',
    43: 'lavender',
    61: 'lavender',
  };
  const FILL = {
    base: 'rgba(196, 199, 252, 0.22)',
    cyan: '#61E5FC',
    mint: '#7CF6B5',
    lavender: '#C4C7FC',
  } as const;

  return (
    <div className="hero-art" aria-hidden="true">
      <svg viewBox="0 0 380 500" xmlns="http://www.w3.org/2000/svg">
        <rect
          x="0.5"
          y="0.5"
          width="379"
          height="499"
          rx="4"
          fill="rgba(13, 22, 53, 0.55)"
          stroke="rgba(97, 229, 252, 0.22)"
        />

        <text x="20" y="30" className="hero-art-header">
          {t('hero.art.header')}
        </text>
        <g className="hero-art-live" transform="translate(360, 30)">
          <circle cx="-4" cy="-4" r="3.2" fill="#61E5FC" />
          <text x="-14" y="0" textAnchor="end" className="hero-art-live-text">
            {t('hero.art.live')}
          </text>
        </g>
        <line x1="20" y1="42" x2="360" y2="42" stroke="rgba(97, 229, 252, 0.3)" />

        <g>
          {Array.from({ length: TOTAL }).map((_, i) => {
            const col = i % COLS;
            const row = Math.floor(i / COLS);
            const cx = GRID_ORIGIN_X + col * CELL_W + CELL_W / 2;
            const cy = GRID_ORIGIN_Y + row * CELL_H + CELL_H / 2;
            const h = highlights[i];
            return (
              <g key={i}>
                {h && <circle cx={cx} cy={cy} r={10} fill={FILL[h]} opacity={0.14} />}
                <circle cx={cx} cy={cy} r={h ? 5 : 3.6} fill={h ? FILL[h] : FILL.base} />
              </g>
            );
          })}
        </g>

        <path
          className="hero-art-signature"
          d="M 30 428 C 55 410, 78 448, 108 424 S 158 402, 192 424 Q 214 442, 240 422 T 300 418 C 322 420, 336 430, 350 424"
        />

        <g className="hero-art-stamp" transform="translate(288, 372) rotate(-7)">
          <circle
            cx="0"
            cy="0"
            r="50"
            fill="none"
            stroke="rgba(124, 246, 181, 0.68)"
            strokeWidth="1.8"
          />
          <circle
            cx="0"
            cy="0"
            r="42"
            fill="none"
            stroke="rgba(124, 246, 181, 0.34)"
            strokeWidth="0.75"
          />
          <text x="0" y="-4" textAnchor="middle" className="hero-art-stamp-primary">
            {t('hero.art.stamp')}
          </text>
          <text x="0" y="10" textAnchor="middle" className="hero-art-stamp-secondary">
            {t('hero.art.stampSub')}
          </text>
        </g>

        <text x="20" y="482" className="hero-art-footer">
          {t('hero.art.footer')}
        </text>
      </svg>
    </div>
  );
}

type DocketType = 'issue' | 'petition' | 'response' | 'resolved';
type DocketEntry = {
  key: number;
  time: string;
  code: string;
  type: DocketType;
  title: string;
};

const DOCKET_SEED: Omit<DocketEntry, 'key' | 'time'>[] = [
  { code: 'IKY-W4', type: 'issue', title: 'Broken streetlight reported on Avenue Road' },
  { code: 'OTK-LGA', type: 'petition', title: 'Repair Otukpo Primary — 1,284 of 2,000 signatures' },
  {
    code: 'IKY-W2',
    type: 'response',
    title: 'Hon. Amina Yusuf: “Funding allocated through Q3 review.”',
  },
  {
    code: 'ABJ-CTR',
    type: 'resolved',
    title: 'Stormdrain cleared on 12th Avenue, 6 days after filing',
  },
];

const DOCKET_POOL: Omit<DocketEntry, 'key' | 'time'>[] = [
  { code: 'LAG-IKE', type: 'issue', title: 'Drainage blockage flooding Marina–Idumota junction' },
  {
    code: 'KAD-W3',
    type: 'petition',
    title: 'Free bus passes for seniors — 2,104 of 3,000 signatures',
  },
  {
    code: 'PHC-CTR',
    type: 'response',
    title: 'Mr. Adamu: “Repair tender awarded; works begin Monday.”',
  },
  {
    code: 'KAN-LGA',
    type: 'resolved',
    title: 'Power restored to Sabon Gari market after 11-day outage',
  },
  { code: 'ENU-W5', type: 'issue', title: 'Bus shelter collapse on Old Park Road' },
  {
    code: 'IBA-W2',
    type: 'petition',
    title: 'Reopen public library on Adeoyo Street — 412 of 1,000',
  },
  {
    code: 'CAL-W1',
    type: 'response',
    title: 'Hon. Ekanem: “Walkway repairs scheduled for July 8.”',
  },
  {
    code: 'JOS-W6',
    type: 'resolved',
    title: 'Pothole on Tafawa Balewa Road filled after 4-day filing',
  },
  { code: 'BEN-W2', type: 'issue', title: 'Water main leak outside Ekiosa market' },
  {
    code: 'ABJ-W7',
    type: 'response',
    title: 'Min. of Works: “Bridge inspection report posted publicly.”',
  },
];

function nowWAT(): string {
  const d = new Date();
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function Docket() {
  const { t } = useTranslation();
  const initial = useRef<DocketEntry[]>(
    DOCKET_SEED.map((e, i) => ({
      ...e,
      key: -i - 1,
      time: ['10:42', '09:18', '08:55', '07:30'][i] ?? '07:00',
    })),
  );
  const [entries, setEntries] = useState<DocketEntry[]>(initial.current);
  const [updatedAt, setUpdatedAt] = useState('10:42');
  const [paused, setPaused] = useState(false);
  const [newKey, setNewKey] = useState<number | null>(null);
  const pausedRef = useRef(paused);
  pausedRef.current = paused;

  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    let poolIdx = 0;
    const id = window.setInterval(() => {
      if (pausedRef.current) return;
      const next = DOCKET_POOL[poolIdx % DOCKET_POOL.length];
      poolIdx += 1;
      const key = Date.now();
      const time = nowWAT();
      setEntries((prev) => [{ ...next, key, time }, ...prev.slice(0, 3)]);
      setUpdatedAt(time);
      setNewKey(key);
    }, 8000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div
      id="docket"
      className={`docket${paused ? ' is-paused' : ''}`}
      aria-label={t('docket.title')}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="docket-head">
        <span className="docket-title">{t('docket.title')}</span>
        <span className="docket-live">
          {paused ? t('docket.paused') : t('docket.live', { time: updatedAt })}
        </span>
      </div>
      <div className="docket-list">
        {entries.map((e) => (
          <div key={e.key} className={`docket-row${e.key === newKey ? ' is-new' : ''}`}>
            <span className="docket-time">{e.time}</span>
            <span className="docket-code">{e.code}</span>
            <span className={`docket-type docket-type--${e.type}`}>
              {t(`docket.types.${e.type}`)}
            </span>
            <span className="docket-title-cell">{e.title}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function Parties() {
  const { t } = useTranslation();
  const keys = ['citizens', 'reps', 'government', 'ngos'] as const;
  return (
    <section className="home-section reveal">
      <TypedMarker text={t('parties.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('parties.title')}</h2>
      </div>
      <div className="home-parties">
        {keys.map((k) => (
          <article key={k} className="home-party">
            <p className="home-party-role">{t(`parties.${k}.role`)}</p>
            <h3 className="home-party-title">{t(`parties.${k}.title`)}</h3>
            <p className="home-party-body">{t(`parties.${k}.body`)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

function Articles() {
  const { t } = useTranslation();
  const articles = [
    { key: 'issues', icon: Megaphone },
    { key: 'petitions', icon: FileText },
    { key: 'reps', icon: Users },
    { key: 'discover', icon: MapPin },
    { key: 'notifications', icon: Bell },
    { key: 'transparency', icon: Eye },
  ] as const;
  return (
    <section id="articles" className="home-section reveal">
      <TypedMarker text={t('articles.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('articles.title')}</h2>
      </div>
      <div className="home-articles">
        {articles.map((a, i) => (
          <article key={a.key} className="home-article">
            <div className="home-article-meta">
              <span className="home-article-num">Art. {String(i + 1).padStart(2, '0')}</span>
              <a.icon className="h-3.5 w-3.5 home-article-icon" aria-hidden="true" />
            </div>
            <h3 className="home-article-title">{t(`articles.${a.key}.title`)}</h3>
            <p className="home-article-body">{t(`articles.${a.key}.body`)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

function Principles() {
  const { t } = useTranslation();
  const keys = [
    'transparency',
    'participation',
    'accountability',
    'trust',
    'accessibility',
  ] as const;
  return (
    <section className="home-section home-section-soft reveal">
      <div>
        <TypedMarker text={t('principles.marker')} />
        <div className="home-section-head">
          <h2 className="home-section-title">{t('principles.title')}</h2>
        </div>
        <div className="home-principles">
          {keys.map((k, i) => (
            <div key={k} className="home-principle">
              <span className="home-principle-num">{String(i + 1).padStart(2, '0')}</span>
              <span className="home-principle-label">{t(`principles.list.${k}`)}</span>
            </div>
          ))}
        </div>
        <p className="home-principles-note">
          <Trans i18nKey="principles.note" components={{ em: <em /> }} />
        </p>
      </div>
    </section>
  );
}

function HowItWorks() {
  const { t } = useTranslation();
  const keys = ['join', 'pick', 'act', 'follow'] as const;
  return (
    <section id="how" className="home-section reveal">
      <TypedMarker text={t('steps.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('steps.title')}</h2>
      </div>
      <ol className="home-steps">
        {keys.map((k, i) => (
          <li key={k} className="home-step">
            <span className="home-step-num">
              {t('steps.stepLabel', { n: String(i + 1).padStart(2, '0') })}
            </span>
            <span className="home-step-meta">{t(`steps.${k}.meta`)}</span>
            <h3 className="home-step-title">{t(`steps.${k}.title`)}</h3>
            <p className="home-step-body">{t(`steps.${k}.body`)}</p>
          </li>
        ))}
      </ol>
    </section>
  );
}

function FAQ() {
  const { t } = useTranslation();
  const keys = ['cost', 'privacy', 'reps', 'coverage', 'ownership', 'abuse'] as const;
  return (
    <section id="faq" className="home-section reveal">
      <TypedMarker text={t('faq.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('faq.title')}</h2>
      </div>
      <div className="home-faq">
        {keys.map((k, i) => (
          <details key={k} className="home-faq-item">
            <summary className="home-faq-q">
              <span className="home-faq-num">
                {t('faq.qLabel', { n: String(i + 1).padStart(2, '0') })}
              </span>
              <span>{t(`faq.${k}.q`)}</span>
              <ChevronDown className="home-faq-chevron h-4 w-4" aria-hidden="true" />
            </summary>
            <p className="home-faq-a">{t(`faq.${k}.a`)}</p>
          </details>
        ))}
      </div>
    </section>
  );
}

function Newsletter() {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'submitting' | 'done'>('idle');

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!email.trim()) return;
    setStatus('submitting');
    window.setTimeout(() => setStatus('done'), 500);
  }

  return (
    <section id="updates" className="home-section home-section-soft reveal">
      <div>
        <TypedMarker text={t('newsletter.marker')} />
        <div className="home-newsletter">
          <div className="home-newsletter-copy">
            <h2 className="home-section-title">{t('newsletter.title')}</h2>
            <p className="home-newsletter-sub">{t('newsletter.sub')}</p>
          </div>

          {status === 'done' ? (
            <div className="home-newsletter-done" role="status">
              <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
              <div>
                <p>{t('newsletter.doneTitle')}</p>
                <p className="home-newsletter-done-sub">{t('newsletter.doneSub')}</p>
              </div>
            </div>
          ) : (
            <form className="home-newsletter-form" onSubmit={onSubmit} noValidate>
              <label htmlFor="newsletter-email" className="sr-only">
                {t('newsletter.placeholder')}
              </label>
              <input
                id="newsletter-email"
                type="email"
                required
                autoComplete="email"
                placeholder={t('newsletter.placeholder')}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="home-newsletter-input"
                disabled={status === 'submitting'}
              />
              <button
                type="submit"
                className="home-btn home-btn-primary"
                disabled={status === 'submitting'}
              >
                {status === 'submitting' ? t('newsletter.submitting') : t('newsletter.submit')}
                <ArrowRight className="h-4 w-4" />
              </button>
            </form>
          )}
        </div>
      </div>
    </section>
  );
}

function CTA() {
  const { t } = useTranslation();
  return (
    <section className="home-cta-strip reveal">
      <div className="home-cta-inner">
        <div>
          <span className="home-cta-eyebrow">
            <Mail className="inline h-3 w-3" aria-hidden="true" /> {t('cta.eyebrow')}
          </span>
          <h2 className="home-cta-title">
            <Trans i18nKey="cta.title" components={{ em: <em /> }} />
          </h2>
          <p className="home-cta-sub">{t('cta.sub')}</p>
        </div>
        <div className="home-cta-buttons">
          <Link to="/register" className="home-btn home-btn-primary home-btn-lg">
            {t('cta.ctaPrimary')}
            <ArrowRight className="h-4 w-4" />
          </Link>
          <Link to="/login" className="home-btn home-btn-on-dark home-btn-lg">
            {t('cta.ctaSecondary')}
          </Link>
        </div>
      </div>
    </section>
  );
}

export function Footer() {
  const { t } = useTranslation();
  return (
    <footer className="home-footer">
      <div className="home-footer-row">
        <div className="home-brand">
          <span className="home-brand-mark" aria-hidden="true">
            <img src="/civicos-mark.png" alt="" />
          </span>
          <div className="leading-tight">
            <p className="home-brand-title">CivicOS</p>
            <p className="home-brand-subtitle">{t('nav.brandSubtitle')}</p>
          </div>
        </div>
        <p className="home-footer-mission">{t('footer.mission')}</p>
      </div>
      <div className="home-footer-row home-footer-meta">
        <p>{t('footer.meta', { year: new Date().getFullYear() })}</p>
        <nav className="home-footer-links" aria-label={t('footer.legal.label')}>
          <Link to="/privacy">{t('footer.legal.privacy')}</Link>
        </nav>
        <p className="home-footer-checks">
          <span>
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" /> {t('footer.checks.privacy')}
          </span>
          <span>
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" /> {t('footer.checks.audit')}
          </span>
          <span>
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" /> {t('footer.checks.open')}
          </span>
        </p>
      </div>
    </footer>
  );
}
