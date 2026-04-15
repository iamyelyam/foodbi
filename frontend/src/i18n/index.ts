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
 * t('dashboard.totalRevenue')                          // "Общая выручка"
 * t('ai.s.topSeller.title', { product: 'Плов' })       // "Promote top seller: Плов"
 * ```
 *
 * Supports `{name}`-style placeholders interpolated from the optional `params`
 * object. Used by backend i18n flow where backend returns keys + params and
 * frontend renders the localized string.
 */
export function useT() {
  const locale = useI18nStore((s) => s.locale)
  const dict = translations[locale] ?? translations.en

  return function t(
    key: string,
    params?: Record<string, string | number>
  ): string {
    const lookup = (root: any) => {
      let val: any = root
      for (const part of key.split('.')) {
        val = val?.[part]
        if (val === undefined) return undefined
      }
      return val
    }
    let resolved = lookup(dict)
    if (typeof resolved !== 'string') {
      const fallback = lookup(translations.en)
      resolved = typeof fallback === 'string' ? fallback : key
    }
    if (params) {
      resolved = resolved.replace(/\{(\w+)\}/g, (_m: string, k: string) =>
        params[k] !== undefined ? String(params[k]) : `{${k}}`
      )
    }
    return resolved
  }
}
