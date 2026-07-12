import { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';

const EFFECTIVE_DATE = '2026-07-04';

export function TermsPage() {
  const { t, i18n } = useTranslation();
  useSeo({
    title: 'Terms of Service — CivicOS',
    description:
      'The terms governing use of CivicOS — public register conduct, moderation, verified accounts, petitions, and account termination.',
  });

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }, []);

  const effective = new Date(EFFECTIVE_DATE).toLocaleDateString(i18n.language || 'en-GB', {
    day: '2-digit',
    month: 'long',
    year: 'numeric',
  });

  const sections = [
    'who',
    'what',
    'eligibility',
    'account',
    'content',
    'conduct',
    'moderation',
    'reps',
    'petitions',
    'availability',
    'termination',
    'law',
    'contact',
  ] as const;

  return (
    <div className="home-shell">
      <TopNav />

      <article className="privacy-page">
        <header className="privacy-head">
          <p className="privacy-eyebrow">{t('terms.eyebrow')}</p>
          <h1 className="privacy-title">{t('terms.title')}</h1>
          <p className="privacy-effective">{t('terms.effective', { date: effective })}</p>
          <p className="privacy-intro">{t('terms.intro')}</p>
        </header>

        <ol className="privacy-list">
          {sections.map((key, i) => (
            <li key={key} id={key} className="privacy-clause">
              <p className="privacy-clause-num">§ {String(i + 1).padStart(2, '0')}</p>
              <h2 className="privacy-clause-title">{t(`terms.sections.${key}.title`)}</h2>
              <div className="privacy-clause-body">
                <Trans
                  i18nKey={`terms.sections.${key}.body`}
                  components={{
                    p: <p />,
                    strong: <strong />,
                    em: <em />,
                    mail: <a href="mailto:hello@civicos.ng" className="privacy-link" />,
                    privacy: <a href="/privacy" className="privacy-link" />,
                  }}
                />
              </div>
            </li>
          ))}
        </ol>
      </article>

      <Footer />
    </div>
  );
}
