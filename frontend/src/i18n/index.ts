import { create } from 'zustand'
import en from './en.json'
import ru from './ru.json'
import kk from './kk.json'
import es from './es.json'

export type Locale = 'en' | 'ru' | 'kk' | 'es'

export const LOCALE_NAMES: Record<Locale, string> = {
  en: 'English',
  ru: 'Русский',
  kk: 'Қазақша',
  es: 'Español',
}

type Translations = typeof en

const translations: Record<Locale, Translations> = { en, ru, kk, es }

interface I18nState {
  locale: Locale
  setLocale: (locale: Locale) => void
}

export const useI18nStore = create<I18nState>((set) => ({
  locale: (localStorage.getItem('foodbi_locale') as Locale) || 'ru',
  setLocale: (locale) => {
    localStorage.setItem('foodbi_locale', locale)
    set({ locale })
  },
}))

/**
 * Translation hook. Usage:
 * ```
 * const t = useT()
 * t('dashboard.totalRevenue') // "Общая выручка"
 * ```
 */
export function useT() {
  const locale = useI18nStore((s) => s.locale)
  const dict = translations[locale] ?? translations.en

  return function t(key: string): string {
    const parts = key.split('.')
    let val: any = dict
    for (const part of parts) {
      val = val?.[part]
      if (val === undefined) break
    }
    if (typeof val === 'string') return val
    // Fallback to English
    let fallback: any = translations.en
    for (const part of parts) {
      fallback = fallback?.[part]
      if (fallback === undefined) break
    }
    return typeof fallback === 'string' ? fallback : key
  }
}
