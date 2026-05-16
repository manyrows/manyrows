import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import en from './locales/en.json';

const supportedLanguages = ['en'] as const;
type SupportedLanguage = typeof supportedLanguages[number];

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
    },
    fallbackLng: 'en',
    supportedLngs: [...supportedLanguages],
    interpolation: {
      escapeValue: false, // React already escapes by default
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'i18nextLng',
      caches: ['localStorage'],
    },
  });

/**
 * Sync language from user account preference.
 * Call this after fetching user data from the backend.
 */
export function setLanguageFromUser(lang?: string): void {
  if (lang && supportedLanguages.includes(lang as SupportedLanguage)) {
    i18n.changeLanguage(lang);
  }
}
