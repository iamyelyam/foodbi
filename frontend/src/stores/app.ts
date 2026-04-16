import { create } from 'zustand'

interface CompanySettings {
  country: string
  currency: string
  currency_symbol: string
  locale: string
}

// User-level UI preferences persisted to localStorage. Survives reloads & re-logins
// on the same device. Not synced to backend (per-device by design — different users
// on the same browser may want different experiences).
interface UiPrefs {
  showUploadInvoicesBanner: boolean
}
const ACTIVE_LOC_KEY = 'foodbi-active-location'
const UI_PREFS_KEY = 'foodbi-ui-prefs-v1'
const DEFAULT_UI_PREFS: UiPrefs = { showUploadInvoicesBanner: true }
function readUiPrefs(): UiPrefs {
  try {
    const s = typeof localStorage !== 'undefined' ? localStorage.getItem(UI_PREFS_KEY) : null
    if (!s) return DEFAULT_UI_PREFS
    return { ...DEFAULT_UI_PREFS, ...JSON.parse(s) }
  } catch {
    return DEFAULT_UI_PREFS
  }
}
function writeUiPrefs(p: UiPrefs) {
  try {
    localStorage.setItem(UI_PREFS_KEY, JSON.stringify(p))
  } catch {
    /* localStorage unavailable (private mode, quota) — preference is session-only */
  }
}

// Global date range — shared across Revenue, Purchases, Dashboard, Statistics.
// Changing on one page reflects everywhere. Default: today.
const todayStr = () => new Date().toISOString().split('T')[0]
const thirtyDaysAgo = () => new Date(Date.now() - 30 * 86400000).toISOString().split('T')[0]

interface AppState {
  // Global date filter
  dateFrom: string
  dateTo: string
  setDateRange: (from: string, to: string) => void
  // Multi-select location filter. Empty array == "all locations".
  selectedLocationIds: string[]
  // Derived: when exactly 1 location selected → its id. Else null.
  // Backend currently accepts a single ?location_id=, so this drives API filtering.
  activeLocationId: string | null
  setSelectedLocations: (ids: string[]) => void
  /** @deprecated use setSelectedLocations */
  setActiveLocation: (id: string | null) => void
  companySettings: CompanySettings
  setCompanySettings: (settings: CompanySettings) => void
  uiPrefs: UiPrefs
  setUiPref: <K extends keyof UiPrefs>(key: K, value: UiPrefs[K]) => void
}

export const useAppStore = create<AppState>((set) => ({
  dateFrom: thirtyDaysAgo(),
  dateTo: todayStr(),
  setDateRange: (from, to) => set({ dateFrom: from, dateTo: to }),
  selectedLocationIds: (() => {
    try {
      const v = localStorage.getItem(ACTIVE_LOC_KEY)
      if (!v) return []
      // Support both legacy single-id and new JSON array format
      if (v.startsWith('[')) return JSON.parse(v) as string[]
      return [v]
    } catch { return [] }
  })(),
  activeLocationId: (() => {
    try {
      const v = localStorage.getItem(ACTIVE_LOC_KEY)
      if (!v) return null
      if (v.startsWith('[')) { const arr = JSON.parse(v) as string[]; return arr.length === 1 ? arr[0] : null }
      return v
    } catch { return null }
  })(),
  setSelectedLocations: (ids) => {
    const active = ids.length === 1 ? ids[0] : null
    try { localStorage.setItem(ACTIVE_LOC_KEY, JSON.stringify(ids)) } catch {}
    set({ selectedLocationIds: ids, activeLocationId: active })
  },
  setActiveLocation: (id) => {
    const ids = id ? [id] : []
    try { localStorage.setItem(ACTIVE_LOC_KEY, JSON.stringify(ids)) } catch {}
    set({ activeLocationId: id, selectedLocationIds: ids })
  },
  companySettings: {
    country: 'KZ',
    currency: 'KZT',
    currency_symbol: '₸',
    locale: 'ru-KZ',
  },
  setCompanySettings: (settings) => set({ companySettings: settings }),
  uiPrefs: readUiPrefs(),
  setUiPref: (key, value) =>
    set((s) => {
      const next = { ...s.uiPrefs, [key]: value }
      writeUiPrefs(next)
      return { uiPrefs: next }
    }),
}))

/** Helper hook — returns just the currency symbol for formatting */
export function useCurrency() {
  return useAppStore((s) => s.companySettings.currency_symbol)
}
