import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Trans, useTranslation } from 'react-i18next';
import {
  ArrowRight,
  Bell,
  CheckCircle2,
  ChevronDown,
  Eye,
  FileText,
  Github,
  Mail,
  Megaphone,
  MapPin,
  ShieldCheck,
  Users,
} from 'lucide-react';
import { LanguageSwitcher } from '../components/LanguageSwitcher';
import { ThemeToggle } from '../components/ThemeToggle';
import { useSeo } from '../hooks/useSeo';
import { hasAccessToken } from '../App';

export function HomePage() {
  useScrollReveal();
  useScrollToHash();
  useSeo({
    title: 'CivicOS — Where citizen reports reach local government',
    description:
      'Report local issues, sign verified community petitions, and track official responses across your ward, LGA, and state on one public record.',
  });
  return (
    <div className="home-shell">
      <TopNav />
      <Hero />
      <Manifesto />
      <Parties />
      <Articles />
      <Stories />
      <Principles />
      <HowItWorks />
      <Stewardship />
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

// The emphasized phrase in the hero headline ("local government" in
// English) cycles through real Nigerian LGA names before landing back on
// the resting phrase. The point is specificity: the headline claims
// reports reach local government, and the cycle names the actual councils
// a visitor might live under instead of gesturing at "democracy" in the
// abstract. LGAs are sampled from the same NIGERIAN_STATES table the
// onboarding wizard uses, so nothing here is invented. Rendered as an
// inline-grid so the container is sized to the widest entry; individual
// phrases crossfade with a small upward slide. Screen readers only hear
// the resting phrase via aria-label — the visible cycle is aria-hidden.
const CYCLED_LGAS = [
  'Ikeja LGA',
  'Otukpo LGA',
  'Kaduna North',
  'Alimosho LGA',
  'Sabon Gari',
  'Umuahia North',
  'Gwagwalada',
] as const;

function CyclingHeroEm({ children }: { children?: React.ReactNode }) {
  // <Trans> hands the <em> body over as an ARRAY of text nodes, not a bare
  // string, so a `typeof children === 'string'` check silently fell through
  // to the English fallback in every locale — Hausa read "…ke isa ga local
  // government". Flatten instead, and keep the literal only as a real
  // last resort.
  const restingWord = useMemo(() => {
    const flatten = (node: React.ReactNode): string => {
      if (typeof node === 'string') return node;
      if (typeof node === 'number') return String(node);
      if (Array.isArray(node)) return node.map(flatten).join('');
      return '';
    };
    return flatten(children).trim() || 'local government';
  }, [children]);

  const words = useMemo(() => [restingWord, ...CYCLED_LGAS], [restingWord]);
  const [idx, setIdx] = useState(0);
  const [width, setWidth] = useState<number | null>(null);
  const containerRef = useRef<HTMLElement>(null);

  // Cycle the phrase.
  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    // The resting phrase lingers 2.4s (the claim itself). Each LGA gets 1.4s.
    const delay = idx === 0 ? 2400 : 1400;
    const id = window.setTimeout(() => {
      setIdx((prev) => (prev + 1) % words.length);
    }, delay);
    return () => window.clearTimeout(id);
  }, [idx, words.length]);

  // Lock the container to the WIDEST phrase, measured once per layout —
  // not to the active one.
  //
  // The emphasized phrase sits at the end of the headline, so any change to
  // its width reflows the wrap of the text before it. Sizing the container
  // per-phrase therefore made the h1 flip between 4 and 5 lines and shoved
  // the paragraph + CTAs down by up to 90px mid-animation (measured at
  // 1024/1440/1920px). A single locked width means the wrap can never
  // change; the crossfade + upward slide still carry the effect.
  //
  // Re-measured on resize and after fonts settle, since the h1 font-size is
  // a vw-based clamp and Fraunces metrics differ from the fallback serif.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const measure = () => {
      const spans = el.querySelectorAll<HTMLSpanElement>('.hero-daily-swap-word');
      const widest = [...spans].reduce((max, s) => Math.max(max, s.scrollWidth), 0);
      if (widest > 0) setWidth(widest);
    };

    measure();
    document.fonts?.ready.then(measure).catch(() => {});
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, [words]);

  return (
    <em
      ref={containerRef}
      className="hero-daily-swap"
      style={width !== null ? { width: `${width}px` } : undefined}
      aria-label={restingWord}
    >
      {/* Height driver — invisible, in flow. Keeps the container's line
          height correct without an anchor width. Its content mirrors the
          active word so height stays accurate across font-size clamps. */}
      <span className="hero-daily-swap-void" aria-hidden="true">
        {words[idx]}
      </span>
      {words.map((w, i) => (
        <span
          key={w}
          className={`hero-daily-swap-word${i === idx ? ' is-active' : ''}`}
          aria-hidden="true"
        >
          {w}
        </span>
      ))}
    </em>
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

// Mouse-tracked radial glow that follows the cursor across the hero. The
// CSS reads --mx / --my as pixel offsets and paints a soft blue spotlight
// through them. Fine-pointer + reduced-motion aware — no work on touch
// screens or when motion is opted out.
function useHeroSpotlight() {
  const ref = useRef<HTMLElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (!window.matchMedia('(pointer: fine)').matches) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const handler = (e: PointerEvent) => {
      const rect = el.getBoundingClientRect();
      el.style.setProperty('--mx', `${e.clientX - rect.left}px`);
      el.style.setProperty('--my', `${e.clientY - rect.top}px`);
    };
    el.addEventListener('pointermove', handler);
    return () => el.removeEventListener('pointermove', handler);
  }, []);
  return ref;
}

// True once the user has scrolled past the given threshold. Used to give
// the nav bar a stronger backdrop-blur + shadow once you leave the hero,
// so it separates from the content beneath.
function useScrolledPast(threshold: number) {
  const [past, setPast] = useState(false);
  useEffect(() => {
    const handler = () => setPast(window.scrollY > threshold);
    handler();
    window.addEventListener('scroll', handler, { passive: true });
    return () => window.removeEventListener('scroll', handler);
  }, [threshold]);
  return past;
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
  const scrolled = useScrolledPast(80);
  // Auth-aware CTAs — signed-in visitors see a "Dashboard" shortcut
  // instead of Sign in / Get started so they can jump back into the
  // app after browsing the marketing homepage.
  const signedIn = hasAccessToken();
  return (
    <header className={`home-nav${scrolled ? ' is-scrolled' : ''}`}>
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
        {/* External resources — new tab so citizens don't lose their
            place on the landing page. rel="noopener" is standard for
            target="_blank" links; noreferrer keeps the destination
            from seeing where the click came from. */}
        <a href="https://docs.civicos.ng" target="_blank" rel="noopener noreferrer">
          {t('nav.links.docs')}
        </a>
        <a href="https://github.com/civicos-hq/civicos" target="_blank" rel="noopener noreferrer">
          {t('nav.links.github')}
        </a>
      </nav>

      <div className="home-nav-cta">
        <ThemeToggle />
        <LanguageSwitcher />
        {signedIn ? (
          <Link to="/discover" className="home-btn home-btn-primary">
            {t('nav.dashboard', 'Dashboard')}
            <ArrowRight className="h-4 w-4" />
          </Link>
        ) : (
          <>
            <Link to="/login" className="home-link">
              {t('nav.signIn')}
            </Link>
            <Link to="/register" className="home-btn home-btn-primary">
              {t('nav.register')}
              <ArrowRight className="h-4 w-4" />
            </Link>
          </>
        )}
      </div>
    </header>
  );
}

/**
 * HeroSlideshow — cycles through every design in /public/designs/ as
 * the hero cover. Cross-fades every ~4.5s. All frames render into the
 * DOM stacked with opacity transitions so the browser can swap without
 * layout thrash. Respects prefers-reduced-motion (holds on the first
 * frame). Every image but the first is lazy-loaded to keep first-paint
 * cheap; the first frame gets `fetchPriority="high"` so LCP still
 * lands on the hero image.
 */
function HeroSlideshow() {
  const frames = [
    '/designs/01_hero_cover.png?v=7',
    '/designs/02_government_listening.png?v=7',
    '/designs/03_university_consultation.png?v=7',
    '/designs/04_ngo_engagement.png?v=7',
    '/designs/05_community_participation.png?v=7',
    '/designs/06_issue_reporting.png?v=7',
    '/designs/07_consultation_lifecycle.png?v=7',
    '/designs/08_representatives_engage.png?v=7',
    '/designs/09_trust_transparency.png?v=7',
    '/designs/10_data_impact.png?v=7',
    '/designs/14_future_together.png?v=7',
  ];
  const [idx, setIdx] = useState(0);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const id = window.setInterval(() => {
      setIdx((prev) => (prev + 1) % frames.length);
    }, 4500);
    return () => window.clearInterval(id);
  }, [frames.length]);

  return (
    <div className="home-hero-visual" aria-hidden="true">
      {frames.map((src, i) => (
        <img
          key={src}
          src={src}
          alt=""
          loading={i === 0 ? 'eager' : 'lazy'}
          fetchPriority={i === 0 ? 'high' : 'low'}
          width={1672}
          height={668}
          className={`home-hero-visual-frame${i === idx ? ' is-active' : ''}`}
        />
      ))}
    </div>
  );
}

function Hero() {
  const { t, i18n } = useTranslation();
  const spotlightRef = useHeroSpotlight();
  const today = new Date()
    .toLocaleDateString(i18n.language || 'en-GB', {
      weekday: 'short',
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    })
    .toUpperCase();

  return (
    <section className="home-hero" ref={spotlightRef}>
      <div className="home-hero-orb home-hero-orb--1" aria-hidden="true" />
      <div className="home-hero-orb home-hero-orb--2" aria-hidden="true" />

      {/* Masthead (tagline + today's date) sits above the slideshow so
          the very first thing a visitor sees is who this is for. */}
      <div className="home-hero-masthead">
        <span>
          <span className="pr-cyan-dot" aria-hidden="true" /> {t('hero.masthead')}
        </span>
        <span>{today}</span>
      </div>

      <HeroSlideshow />

      <div className="home-hero-body">
        <div className="home-hero-copy">
          <h1 className="home-hero-title">
            <Trans i18nKey="hero.headline" components={{ em: <CyclingHeroEm /> }} />
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

        <Docket />
      </div>
    </section>
  );
}

type DocketType = 'issue' | 'petition' | 'response' | 'resolved';
type DocketEntry = {
  key: number;
  time: string;
  /** Human-readable ward + LGA, e.g. "Sabon Gari, Zaria LGA". */
  location: string;
  type: DocketType;
  title: string;
};

// Sample records for the live-activity panel. Deliberately written as
// things a resident would actually recognise — a named street, a real
// ward, an LGA that exists in data/nigeria.ts — rather than reference
// codes. Opaque identifiers (IKY-W4, ABJ-CTR) read as placeholder
// fixture data and undercut the panel's whole job, which is to show a
// visitor what their own feed will look like.
const DOCKET_SEED: Omit<DocketEntry, 'key' | 'time'>[] = [
  {
    location: 'Sabon Gari, Zaria LGA',
    type: 'issue',
    title: 'Transformer repair requested on Commercial Avenue',
  },
  {
    location: 'Otukpo LGA, Benue',
    type: 'petition',
    title: 'Solar streetlights for Otukpo market road — 840 of 1,500 signatures',
  },
  {
    location: 'Ikeja Ward 2, Lagos',
    type: 'response',
    title: 'Hon. Amina Yusuf: “Contractor mobilised for the Q3 road grading.”',
  },
  {
    location: 'Municipal Area Council, Abuja',
    type: 'resolved',
    title: 'Drainage cleared along 12th Avenue after 4 days',
  },
];

const DOCKET_POOL: Omit<DocketEntry, 'key' | 'time'>[] = [
  {
    location: 'Lagos Island LGA, Lagos',
    type: 'issue',
    title: 'Blocked drainage flooding the Marina–Idumota junction',
  },
  {
    location: 'Kaduna North LGA, Kaduna',
    type: 'petition',
    title: 'Free bus passes for seniors — 2,104 of 3,000 signatures',
  },
  {
    location: 'Port Harcourt LGA, Rivers',
    type: 'response',
    title: 'Works department: “Repair tender awarded, crews start Monday.”',
  },
  {
    location: 'Fagge LGA, Kano',
    type: 'resolved',
    title: 'Power restored to Sabon Gari market after an 11-day outage',
  },
  {
    location: 'Enugu North LGA, Enugu',
    type: 'issue',
    title: 'Collapsed bus shelter on Old Park Road',
  },
  {
    location: 'Ibadan North LGA, Oyo',
    type: 'petition',
    title: 'Reopen the public library on Adeoyo Street — 412 of 1,000 signatures',
  },
  {
    location: 'Calabar Municipal, Cross River',
    type: 'response',
    title: 'Hon. Ekanem: “Walkway repairs scheduled to start on the 8th.”',
  },
  {
    location: 'Jos North LGA, Plateau',
    type: 'resolved',
    title: 'Pothole on Tafawa Balewa Road filled 4 days after filing',
  },
  {
    location: 'Oredo LGA, Edo',
    type: 'issue',
    title: 'Water main leaking outside Ekiosa market',
  },
  {
    location: 'Bwari LGA, Abuja',
    type: 'response',
    title: 'Ministry of Works: “Bridge inspection report published in full.”',
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
      // Keep 4 records — panel now spans the hero column and needs enough
      // to feel alive without scrolling.
      setEntries((prev) => [{ ...next, key, time }, ...prev.slice(0, 3)]);
      setNewKey(key);
    }, 8000);
    return () => window.clearInterval(id);
  }, []);

  const visible = entries.slice(0, 4);

  return (
    <aside
      id="docket"
      className={`docket${paused ? ' is-paused' : ''}`}
      aria-label={t('docket.title')}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <header className="docket-chrome" aria-hidden="true">
        <span className="docket-chrome-dot" />
        <span className="docket-chrome-dot" />
        <span className="docket-chrome-dot" />
        <span className="docket-chrome-url">{t('docket.chromeUrl')}</span>
      </header>

      <div className="docket-body">
        <div className="docket-meta">
          <span className="docket-eyebrow">{t('docket.eyebrow')}</span>
          <span className="docket-live">
            <span className="docket-live-dot" aria-hidden="true" />
            {t('docket.liveLabel')}
          </span>
        </div>

        <h2 className="docket-district">{t('docket.district')}</h2>

        <div className="docket-records">
          {visible.map((e) => (
            <article key={e.key} className={`docket-record${e.key === newKey ? ' is-new' : ''}`}>
              <div className="docket-record-head">
                <h3 className="docket-record-title">{e.title}</h3>
                <span className={`docket-pill docket-pill--${e.type}`}>
                  {t(`docket.pill.${e.type}`)}
                </span>
              </div>
              {(e.type === 'issue' || e.type === 'petition') && (
                <div className="docket-progress" aria-hidden="true">
                  <span className="docket-progress-seg is-done" />
                  <span className="docket-progress-seg is-done" />
                  <span className="docket-progress-seg" />
                  <span className="docket-progress-seg" />
                </div>
              )}
              <p className="docket-record-meta">
                <span>{t('docket.reportedAt', { time: e.time })}</span>
                <span className="docket-record-place">
                  <MapPin className="h-3 w-3" aria-hidden="true" />
                  {e.location}
                </span>
              </p>
            </article>
          ))}
        </div>
      </div>
    </aside>
  );
}

// § 00 preamble that positions CivicOS above the numbered sections. Big
// serif-sans pull quote, blue-shimmer emphasis on the thesis phrase, and
// a short body identifying the four constituencies (governments, unis,
// NGOs, communities). Uses the standard .reveal system so it fades in
// via the clerk's-scan pattern; actor tags stagger in after.
function Manifesto() {
  const { t } = useTranslation();
  const actorKeys = ['governments', 'universities', 'ngos', 'communities'] as const;
  const actorImages: Record<(typeof actorKeys)[number], string> = {
    governments: '/designs/02_government_listening.png?v=7',
    universities: '/designs/03_university_consultation.png?v=7',
    ngos: '/designs/04_ngo_engagement.png?v=7',
    communities: '/designs/05_community_participation.png?v=7',
  };
  const beliefKeys = ['one', 'two', 'three', 'four', 'five'] as const;
  return (
    <section className="home-section home-section-manifesto reveal">
      <TypedMarker text={t('manifesto.marker')} />
      <div className="home-manifesto">
        <h2 className="home-manifesto-title">
          <Trans i18nKey="manifesto.headline" components={{ em: <em /> }} />
        </h2>
        <p className="home-manifesto-body">{t('manifesto.body')}</p>
        <ul className="home-manifesto-actors" aria-label="Who CivicOS is for">
          {actorKeys.map((k) => (
            <li key={k} className="home-manifesto-actor">
              <img
                className="home-manifesto-actor-img"
                src={actorImages[k]}
                alt=""
                loading="lazy"
                width={1672}
                height={668}
              />
              <span>{t(`manifesto.actors.${k}`)}</span>
            </li>
          ))}
        </ul>

        {/* Brand-guide beliefs — poetic closer to the manifesto. Each
            "we believe" line stands alone; the closer lands on
            "together" to match the tagline in the hero + top nav. */}
        <div className="home-manifesto-beliefs" aria-label={t('manifesto.beliefs.heading')}>
          <p className="home-manifesto-beliefs-heading">{t('manifesto.beliefs.heading')}</p>
          <ul className="home-manifesto-beliefs-list">
            {beliefKeys.map((k) => (
              <li key={k}>{t(`manifesto.beliefs.${k}`)}</li>
            ))}
          </ul>
          <p className="home-manifesto-beliefs-closer">{t('manifesto.beliefs.closer')}</p>
        </div>
      </div>
    </section>
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
        {articles.map((a) => (
          <article key={a.key} className="home-article">
            <div className="home-article-meta">
              <a.icon className="h-4 w-4 home-article-icon" aria-hidden="true" />
            </div>
            <h3 className="home-article-title">{t(`articles.${a.key}.title`)}</h3>
            <p className="home-article-body">{t(`articles.${a.key}.body`)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

/**
 * "Stories" — three big illustrated cards showing CivicOS's flagship
 * flows (issue reporting, consultations, representative engagement),
 * each written as a concrete situation rather than a capability claim.
 */
function Stories() {
  const { t } = useTranslation();
  const stories = [
    { key: 'issues', img: '/designs/06_issue_reporting.png?v=7' },
    { key: 'consultations', img: '/designs/07_consultation_lifecycle.png?v=7' },
    { key: 'reps', img: '/designs/08_representatives_engage.png?v=7' },
  ] as const;
  return (
    <section className="home-section reveal">
      <TypedMarker text={t('stories.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('stories.title')}</h2>
      </div>
      <div className="home-stories-grid">
        {stories.map((s) => (
          <article key={s.key} className="home-story-card">
            <img
              className="home-story-img"
              src={s.img}
              alt=""
              loading="lazy"
              width={1672}
              height={668}
            />
            <h3 className="home-story-title">{t(`stories.${s.key}.title`)}</h3>
            <p className="home-story-body">{t(`stories.${s.key}.body`)}</p>
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
          {keys.map((k) => (
            <div key={k} className="home-principle">
              <span className="home-principle-label">{t(`principles.list.${k}.label`)}</span>
              <p className="home-principle-body">{t(`principles.list.${k}.body`)}</p>
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

const STEP_KEYS = ['join', 'pick', 'act', 'follow'] as const;

const STEP_IMAGES: Record<(typeof STEP_KEYS)[number], string> = {
  join: '/designs/05_community_participation.png?v=7',
  pick: '/designs/09_trust_transparency.png?v=7',
  act: '/designs/06_issue_reporting.png?v=7',
  follow: '/designs/07_consultation_lifecycle.png?v=7',
};

/**
 * HowItWorks — a step-through rather than four uniform cards. The rail on
 * the left lists the four stages; selecting one swaps the detail panel
 * beside it. Implemented as a real ARIA tablist with roving tabindex, so
 * ←/→ (and ↑/↓ on the vertical rail) move between steps and Home/End jump
 * to the ends. Advancing past the last step wraps to the first.
 *
 * The section is the one place on the page where the visitor is being told
 * a sequence, so it earns a different interaction model from the sibling
 * card grids above and below it.
 */
function HowItWorks() {
  const { t } = useTranslation();
  const [active, setActive] = useState(0);
  const tabsRef = useRef<(HTMLButtonElement | null)[]>([]);

  function focusStep(next: number) {
    const wrapped = (next + STEP_KEYS.length) % STEP_KEYS.length;
    setActive(wrapped);
    tabsRef.current[wrapped]?.focus();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
    switch (e.key) {
      case 'ArrowDown':
      case 'ArrowRight':
        e.preventDefault();
        focusStep(active + 1);
        break;
      case 'ArrowUp':
      case 'ArrowLeft':
        e.preventDefault();
        focusStep(active - 1);
        break;
      case 'Home':
        e.preventDefault();
        focusStep(0);
        break;
      case 'End':
        e.preventDefault();
        focusStep(STEP_KEYS.length - 1);
        break;
    }
  }

  const activeKey = STEP_KEYS[active];

  return (
    <section id="how" className="home-section reveal">
      <TypedMarker text={t('steps.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('steps.title')}</h2>
      </div>

      <div className="home-stepper">
        <div
          className="home-stepper-rail"
          role="tablist"
          aria-orientation="vertical"
          aria-label={t('steps.listLabel')}
          onKeyDown={onKeyDown}
        >
          {STEP_KEYS.map((k, i) => (
            <button
              key={k}
              type="button"
              role="tab"
              id={`step-tab-${k}`}
              aria-selected={i === active}
              aria-controls={`step-panel-${k}`}
              tabIndex={i === active ? 0 : -1}
              ref={(el) => {
                tabsRef.current[i] = el;
              }}
              className={`home-stepper-tab${i === active ? ' is-active' : ''}`}
              onClick={() => setActive(i)}
            >
              <span className="home-stepper-tab-num" aria-hidden="true">
                {String(i + 1).padStart(2, '0')}
              </span>
              <span className="home-stepper-tab-label">
                <span className="home-stepper-tab-title">{t(`steps.${k}.title`)}</span>
                <span className="home-stepper-tab-meta">{t(`steps.${k}.meta`)}</span>
              </span>
            </button>
          ))}
        </div>

        {/* Keyed on the active step so React remounts the panel and the
            fade-up animation replays on every selection. */}
        <div
          key={activeKey}
          className="home-stepper-panel"
          role="tabpanel"
          id={`step-panel-${activeKey}`}
          aria-labelledby={`step-tab-${activeKey}`}
          tabIndex={0}
        >
          <img
            className="home-stepper-panel-img"
            src={STEP_IMAGES[activeKey]}
            alt=""
            loading="lazy"
            width={1672}
            height={668}
          />
          <p className="home-stepper-panel-step">
            {t('steps.stepLabel', { n: String(active + 1).padStart(2, '0') })}
          </p>
          <h3 className="home-stepper-panel-title">{t(`steps.${activeKey}.title`)}</h3>
          <p className="home-stepper-panel-body">{t(`steps.${activeKey}.body`)}</p>
        </div>
      </div>
    </section>
  );
}

/**
 * Stewardship — the "who is actually behind this" block. A civic platform
 * asking residents to file their street address under their real name has
 * to answer that question somewhere on the landing page, and answering it
 * honestly means saying what we have not built yet: coverage is per-ward
 * and opt-in, not national. No pilot counts are quoted here until there
 * are real ones to quote.
 */
function Stewardship() {
  const { t } = useTranslation();
  const cards = ['openSource', 'invite', 'contribute'] as const;
  return (
    <section id="stewardship" className="home-section reveal">
      <TypedMarker text={t('stewardship.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('stewardship.title')}</h2>
        <p className="home-stewardship-lede">{t('stewardship.body')}</p>
      </div>

      <div className="home-stewardship">
        {cards.map((c) => (
          <article key={c} className="home-stewardship-card">
            <h3 className="home-stewardship-card-title">{t(`stewardship.${c}.title`)}</h3>
            <p className="home-stewardship-card-body">{t(`stewardship.${c}.body`)}</p>
          </article>
        ))}
      </div>

      <div className="home-stewardship-cta">
        <a
          className="home-btn home-btn-ghost"
          href="https://github.com/civicos-hq/civicos"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Github className="h-4 w-4" aria-hidden="true" />
          {t('stewardship.ctaRepo')}
        </a>
        <Link to="/register" className="home-btn home-btn-ghost">
          {t('stewardship.ctaPartner')}
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    </section>
  );
}

function FAQ() {
  const { t } = useTranslation();
  const keys = ['cost', 'privacy', 'coverage', 'reps', 'abuse', 'ownership'] as const;
  return (
    <section id="faq" className="home-section reveal">
      <TypedMarker text={t('faq.marker')} />
      <div className="home-section-head">
        <h2 className="home-section-title">{t('faq.title')}</h2>
      </div>
      <div className="home-faq">
        {keys.map((k) => (
          <details key={k} className="home-faq-item">
            <summary className="home-faq-q">
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
        <div className="home-cta-copy">
          <span className="home-cta-eyebrow">
            <Mail className="inline h-3 w-3" aria-hidden="true" /> {t('cta.eyebrow')}
          </span>
          <h2 className="home-cta-title">
            <Trans i18nKey="cta.title" components={{ em: <em /> }} />
          </h2>
          <p className="home-cta-sub">{t('cta.sub')}</p>
        </div>
        <img
          className="home-cta-visual"
          src="/designs/14_future_together.png?v=7"
          alt=""
          loading="lazy"
          width={1672}
          height={668}
        />
        <div className="home-cta-buttons">
          <Link to="/register" className="home-btn home-btn-primary home-btn-lg">
            {t('cta.ctaPrimary')}
            <ArrowRight className="h-4 w-4" />
          </Link>
          {/* Officials and LGA staff get sent to the same form with their
              account type pre-selected — see accountTypeFromParam in
              RegisterPage. */}
          <Link
            to="/register?type=REPRESENTATIVE"
            className="home-btn home-btn-on-dark home-btn-lg"
          >
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
      {/* Three-column footer row: brand (left), link columns
          (center), mission (right). The parent .home-footer-row uses
          justify-content: space-between so the middle .home-footer-columns
          block sits naturally between brand and mission at full width,
          and each piece wraps to its own row on narrow screens. */}
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

        <div className="home-footer-columns">
          <nav className="home-footer-col" aria-label={t('footer.developers.label')}>
            <p className="home-footer-col-title">{t('footer.developers.title')}</p>
            <a href="https://docs.civicos.ng" target="_blank" rel="noopener noreferrer">
              {t('footer.developers.docs')}
            </a>
            <a
              href="https://github.com/civicos-hq/civicos"
              target="_blank"
              rel="noopener noreferrer"
            >
              {t('footer.developers.github')}
            </a>
            <a
              href="https://civicos-gateway.onrender.com/docs"
              target="_blank"
              rel="noopener noreferrer"
            >
              {t('footer.developers.api')}
            </a>
          </nav>

          <nav className="home-footer-col" aria-label={t('footer.legal.label')}>
            <p className="home-footer-col-title">{t('footer.legal.label')}</p>
            <Link to="/privacy">{t('footer.legal.privacy')}</Link>
            <Link to="/terms">{t('footer.legal.terms')}</Link>
          </nav>
        </div>

        <p className="home-footer-mission">{t('footer.mission')}</p>
      </div>

      <div className="home-footer-row home-footer-meta">
        <p>{t('footer.meta', { year: new Date().getFullYear() })}</p>
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
