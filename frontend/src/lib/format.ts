const CURRENCY = '€'

export function formatMoney(value: number, decimals = 2): string {
  return `${CURRENCY}${value.toLocaleString('en', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}`
}
