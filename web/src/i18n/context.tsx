import { createContext, useContext, useState, useCallback } from 'react';
import type { Locale, TranslationStrings } from './translations';
import { getTranslation, SUPPORTED_LOCALES } from './translations';

interface I18nContextValue {
  locale: Locale;
  t: TranslationStrings;
  setLocale: (locale: Locale) => void;
  supportedLocales: typeof SUPPORTED_LOCALES;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error('useI18n must be used within I18nProvider');
  return ctx;
}

export function I18nProvider({ children, defaultLocale = 'en' }: { children: React.ReactNode; defaultLocale?: Locale }) {
  const [locale, setLocaleState] = useState<Locale>(defaultLocale);
  const t = getTranslation(locale);
  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    document.documentElement.lang = l;
  }, []);
  return (
    <I18nContext.Provider value={{ locale, t, setLocale, supportedLocales: SUPPORTED_LOCALES }}>
      {children}
    </I18nContext.Provider>
  );
}

export { I18nContext };
