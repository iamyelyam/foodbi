// Shared date-range helpers used by the unified DateRangeSheet/DateRangeBlock UI.
// Centralized here so RevenuePage / PurchasesPage / StockPage / Dashboard etc all
// agree on what "Today" / "This week" mean and how dates are formatted in inputs.

export type PresetKey = 'today' | 'yesterday' | 'this_week' | 'this_month'

export const PRESETS: { key: PresetKey; labelKey: string }[] = [
  { key: 'today', labelKey: 'date.today' },
  { key: 'yesterday', labelKey: 'date.yesterday' },
  { key: 'this_week', labelKey: 'date.thisWeek' },
  { key: 'this_month', labelKey: 'date.thisMonth' },
]

const iso = (d: Date) => d.toISOString().split('T')[0]

export function presetRange(key: PresetKey): [string, string] {
  const now = new Date()
  if (key === 'today') return [iso(now), iso(now)]
  if (key === 'yesterday') {
    const y = new Date(now)
    y.setDate(y.getDate() - 1)
    return [iso(y), iso(y)]
  }
  if (key === 'this_week') {
    const start = new Date(now)
    const dow = start.getDay() === 0 ? 6 : start.getDay() - 1
    start.setDate(start.getDate() - dow)
    return [iso(start), iso(now)]
  }
  // this_month
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  return [iso(start), iso(now)]
}

export function isPresetActive(key: PresetKey, from: string, to: string): boolean {
  const [f, t] = presetRange(key)
  return f === from && t === to
}

// "DD.MM.YYYY" — what we render inside the From/To input buttons.
export function formatInputDate(input: string): string {
  if (!input) return ''
  const d = new Date(input + 'T00:00:00')
  if (isNaN(d.getTime())) return input
  return `${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

export function todayIso(): string {
  return iso(new Date())
}
