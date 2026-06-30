import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

import en from './locales/en.json';
import ig from './locales/ig.json';
import yo from './locales/yo.json';
import ha from './locales/ha.json';
import pcm from './locales/pcm.json';

export const SUPPORTED_LANGUAGES = ['en', 'ig', 'yo', 'ha', 'pcm'] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      ig: { translation: ig },
      yo: { translation: yo },
      ha: { translation: ha },
      pcm: { translation: pcm },
    },
    fallbackLng: 'en',
    supportedLngs: SUPPORTED_LANGUAGES as unknown as string[],
    nonExplicitSupportedLngs: true,
    interpolation: { escapeValue: false },
    detection: {
      order: ['querystring', 'localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupQuerystring: 'lng',
      lookupLocalStorage: 'civicos.lang',
    },
    returnEmptyString: false,
  });

if (typeof document !== 'undefined') {
  i18n.on('languageChanged', (lng) => {
    document.documentElement.lang = lng;
  });
  document.documentElement.lang = i18n.language || 'en';
}

export default i18n;
