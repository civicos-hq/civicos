import { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { TopNav, Footer } from './HomePage';
import { useSeo } from '../hooks/useSeo';

const EFFECTIVE_DATE = '2026-07-03';

export function PrivacyPage() {
  const { t, i18n } = useTranslation();
  useSeo({
    title: 'Privacy Policy — CivicOS',
    description:
      'How CivicOS collects, uses, and protects your data. Bcrypt-hashed passwords, short-lived JWTs, audit-logged admin actions, no third-party trackers.',
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
    'why',
    'donot',
    'sharing',
    'security',
    'retention',
    'rights',
    'cookies',
    'children',
    'changes',
    'contact',
  ] as const;

  return (
    <div className="home-shell">
      <TopNav />

      <article className="privacy-page">
        <header className="privacy-head">
          <p className="privacy-eyebrow">{t('privacy.eyebrow')}</p>
          <h1 className="privacy-title">{t('privacy.title')}</h1>
          <p className="privacy-effective">{t('privacy.effective', { date: effective })}</p>
          <p className="privacy-intro">{t('privacy.intro')}</p>
        </header>

        <ol className="privacy-list">
          {sections.map((key, i) => (
            <li key={key} id={key} className="privacy-clause">
              <p className="privacy-clause-num">§ {String(i + 1).padStart(2, '0')}</p>
              <h2 className="privacy-clause-title">{t(`privacy.sections.${key}.title`)}</h2>
              <div className="privacy-clause-body">
                <Trans
                  i18nKey={`privacy.sections.${key}.body`}
                  components={{
                    p: <p />,
                    strong: <strong />,
                    em: <em />,
                    mail: <a href="mailto:privacy@civicos.ng" className="privacy-link" />,
                    security: <a href="mailto:security@civicos.ng" className="privacy-link" />,
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
