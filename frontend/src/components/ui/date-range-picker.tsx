import { useState, useMemo } from 'react'
import { cn } from '@/lib/utils'
import { Button } from './button'
import { useT } from '@/i18n'
import { useAppStore } from '@/stores/app'

interface DateRangePickerProps {
  startDate?: string // YYYY-MM-DD
  endDate?: string
  onConfirm: (start: string, end: string) => void
  onBack: () => void
}

function getMonthDays(year: number, month: number) {
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  let startOffset = firstDay.getDay() - 1
  if (startOffset < 0) startOffset = 6
  const cells: (number | null)[] = []
  for (let i = 0; i < startOffset; i++) cells.push(null)
  for (let d = 1; d <= lastDay.getDate(); d++) cells.push(d)
  return cells
}

function formatShort(dateStr: string) {
  const d = new Date(dateStr + 'T00:00:00')
  const dd = String(d.getDate()).padStart(2, '0')
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const yy = String(d.getFullYear()).slice(2)
  return `${dd}.${mm}.${yy}`
}

function toDateStr(year: number, month: number, day: number) {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

export function DateRangePicker({ startDate, endDate, onConfirm, onBack }: DateRangePickerProps) {
  const t = useT()
  const locale = useAppStore((s) => s.companySettings.locale)
  const today = new Date()

  const [start, setStart] = useState(startDate || '')
  const [end, setEnd] = useState(endDate || '')

  // Show 2 months: current and previous
  const month1Year = today.getMonth() === 0 ? today.getFullYear() - 1 : today.getFullYear()
  const month1 = today.getMonth() === 0 ? 11 : today.getMonth() - 1
  const month2Year = today.getFullYear()
  const month2 = today.getMonth()

  const days1 = useMemo(() => getMonthDays(month1Year, month1), [month1Year, month1])
  const days2 = useMemo(() => getMonthDays(month2Year, month2), [month2Year, month2])

  const dayNames = useMemo(() => {
    const base = new Date(2024, 0, 1) // Monday
    return Array.from({ length: 7 }, (_, i) => {
      const d = new Date(base)
      d.setDate(d.getDate() + i)
      return d.toLocaleDateString(locale, { weekday: 'short' }).slice(0, 2)
    })
  }, [locale])

  const monthName = (year: number, month: number) =>
    new Date(year, month).toLocaleDateString(locale, { month: 'long', year: 'numeric' })

  const handleDayClick = (year: number, month: number, day: number) => {
    const dateStr = toDateStr(year, month, day)
    if (!start || (start && end)) {
      // Start new selection
      setStart(dateStr)
      setEnd('')
    } else {
      // Set end date
      if (dateStr < start) {
        setEnd(start)
        setStart(dateStr)
      } else {
        setEnd(dateStr)
      }
    }
  }

  const isInRange = (year: number, month: number, day: number) => {
    if (!start || !end) return false
    const dateStr = toDateStr(year, month, day)
    return dateStr >= start && dateStr <= end
  }

  const isStart = (year: number, month: number, day: number) =>
    start === toDateStr(year, month, day)

  const isEnd = (year: number, month: number, day: number) =>
    end === toDateStr(year, month, day)

  const renderMonth = (year: number, month: number, days: (number | null)[]) => (
    <div className="mb-6">
      <p className="text-sm font-semibold text-dark mb-3 capitalize">{monthName(year, month)}</p>
      <div className="grid grid-cols-7 gap-y-1">
        {days.map((day, i) => (
          <div key={i} className="flex items-center justify-center">
            {day ? (
              <button
                onClick={() => handleDayClick(year, month, day)}
                className={cn(
                  'w-9 h-9 rounded-full text-sm font-medium transition-colors',
                  isStart(year, month, day) || isEnd(year, month, day)
                    ? 'bg-primary text-white'
                    : isInRange(year, month, day)
                      ? 'bg-primary-lighter text-primary'
                      : 'text-dark'
                )}
              >
                {day}
              </button>
            ) : (
              <div className="w-9 h-9" />
            )}
          </div>
        ))}
      </div>
    </div>
  )

  const rangeLabel = start && end
    ? `${formatShort(start)} – ${formatShort(end)}`
    : start
      ? formatShort(start)
      : ''

  return (
    <div>
      {/* Selected range display */}
      {rangeLabel && (
        <p className="text-base font-bold text-dark text-center mb-4">{rangeLabel}</p>
      )}

      {/* Day headers */}
      <div className="grid grid-cols-7 gap-1 mb-3">
        {dayNames.map((d, i) => (
          <div key={i} className="text-center text-xs font-semibold text-gray">{d}</div>
        ))}
      </div>

      {/* Two months */}
      <div className="max-h-[400px] overflow-y-auto">
        {renderMonth(month1Year, month1, days1)}
        {renderMonth(month2Year, month2, days2)}
      </div>

      {/* Buttons */}
      <div className="mt-4 space-y-2">
        <Button
          fullWidth
          onClick={() => start && end && onConfirm(start, end)}
          disabled={!start || !end}
        >
          {t('common.confirm')}
        </Button>
        <button
          onClick={onBack}
          className="w-full text-center text-sm font-medium text-primary py-2"
        >
          {t('common.back')}
        </button>
      </div>
    </div>
  )
}
