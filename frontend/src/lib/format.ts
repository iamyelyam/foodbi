import { useAppStore } from '@/stores/app'

export function formatMoney(value: number, decimals = 0): string {
  const symbol = useAppStore.getState().companySettings.currency_symbol
  return `${value.toLocaleString('ru-KZ', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}${symbol}`
}

export function formatMoneyWithSymbol(value: number, symbol: string, decimals = 0): string {
  return `${value.toLocaleString('ru-KZ', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}${symbol}`
}

/** Capitalize first letter, lowercase rest. Used for product names, categories, any iiko-sourced names. */
export function formatProductName(name: string | null | undefined): string {
  if (!name) return ''
  const lower = name.toLocaleLowerCase('ru-RU')
  return lower.charAt(0).toLocaleUpperCase('ru-RU') + lower.slice(1)
}

/** Alias for clarity when formatting category labels. */
export const formatCategory = formatProductName

/** Returns true if string looks like a UUID (8-4-4-4-12 hex pattern). */
export function isUuid(s: string | null | undefined): boolean {
  if (!s) return false
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s)
}

/** Display a friendly supplier label — falls back when iiko gave us only a GUID. */
export function formatSupplierName(name: string | null | undefined): string {
  if (!name) return 'Unknown supplier'
  if (isUuid(name)) return 'Unknown supplier'
  return formatProductName(name)
}

/** Title Case each word (for people names like waiters/employees). */
export function formatPersonName(name: string | null | undefined): string {
  if (!name) return ''
  return name
    .split(/\s+/)
    .filter(Boolean)
    .map((word) => {
      const lower = word.toLocaleLowerCase('ru-RU')
      return lower.charAt(0).toLocaleUpperCase('ru-RU') + lower.slice(1)
    })
    .join(' ')
}
