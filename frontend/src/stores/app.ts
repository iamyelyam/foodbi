import { create } from 'zustand'

interface CompanySettings {
  country: string
  currency: string
  currency_symbol: string
  locale: string
}

interface AppState {
  activeLocationId: string | null
  setActiveLocation: (id: string | null) => void
  companySettings: CompanySettings
  setCompanySettings: (settings: CompanySettings) => void
}

export const useAppStore = create<AppState>((set) => ({
  activeLocationId: null,
  setActiveLocation: (id) => set({ activeLocationId: id }),
  companySettings: {
    country: 'KZ',
    currency: 'KZT',
    currency_symbol: '₸',
    locale: 'ru-KZ',
  },
  setCompanySettings: (settings) => set({ companySettings: settings }),
}))

/** Helper hook — returns just the currency symbol for formatting */
export function useCurrency() {
  return useAppStore((s) => s.companySettings.currency_symbol)
}
