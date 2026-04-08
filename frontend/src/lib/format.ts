import { useAppStore } from '@/stores/app'

export function formatMoney(value: number, decimals = 0): string {
  const symbol = useAppStore.getState().companySettings.currency_symbol
  return `${value.toLocaleString('en', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}${symbol}`
}

export function formatMoneyWithSymbol(value: number, symbol: string, decimals = 0): string {
  return `${value.toLocaleString('en', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}${symbol}`
}
