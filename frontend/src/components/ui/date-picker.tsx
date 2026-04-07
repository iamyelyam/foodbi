import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from './button'

interface DatePickerProps {
  value?: string // YYYY-MM-DD
  onChange: (date: string) => void
  onClose?: () => void
}

const DAYS = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December']

export function DatePicker({ value, onChange, onClose }: DatePickerProps) {
  const today = new Date()
  const selected = value ? new Date(value + 'T00:00:00') : null
  const [viewYear, setViewYear] = useState(selected?.getFullYear() ?? today.getFullYear())
  const [viewMonth, setViewMonth] = useState(selected?.getMonth() ?? today.getMonth())

  const days = useMemo(() => {
    const firstDay = new Date(viewYear, viewMonth, 1)
    const lastDay = new Date(viewYear, viewMonth + 1, 0)
    let startOffset = firstDay.getDay() - 1
    if (startOffset < 0) startOffset = 6

    const cells: (number | null)[] = []
    for (let i = 0; i < startOffset; i++) cells.push(null)
    for (let d = 1; d <= lastDay.getDate(); d++) cells.push(d)
    return cells
  }, [viewYear, viewMonth])

  const prevMonth = () => {
    if (viewMonth === 0) { setViewMonth(11); setViewYear(viewYear - 1) }
    else setViewMonth(viewMonth - 1)
  }

  const nextMonth = () => {
    if (viewMonth === 11) { setViewMonth(0); setViewYear(viewYear + 1) }
    else setViewMonth(viewMonth + 1)
  }

  const selectDay = (day: number) => {
    const m = String(viewMonth + 1).padStart(2, '0')
    const d = String(day).padStart(2, '0')
    onChange(`${viewYear}-${m}-${d}`)
  }

  const isSelected = (day: number) =>
    selected?.getFullYear() === viewYear &&
    selected?.getMonth() === viewMonth &&
    selected?.getDate() === day

  const isToday = (day: number) =>
    today.getFullYear() === viewYear &&
    today.getMonth() === viewMonth &&
    today.getDate() === day

  return (
    <div className="bg-white rounded-[16px] p-4">
      {/* Month navigation */}
      <div className="flex items-center justify-between mb-4">
        <button onClick={prevMonth} className="p-1"><ChevronLeft className="h-5 w-5 text-dark" /></button>
        <span className="text-sm font-semibold text-dark">{MONTHS[viewMonth]} {viewYear}</span>
        <button onClick={nextMonth} className="p-1"><ChevronRight className="h-5 w-5 text-dark" /></button>
      </div>

      {/* Day headers */}
      <div className="grid grid-cols-7 gap-1 mb-2">
        {DAYS.map((d) => (
          <div key={d} className="text-center text-[10px] font-medium text-gray">{d}</div>
        ))}
      </div>

      {/* Day grid */}
      <div className="grid grid-cols-7 gap-1">
        {days.map((day, i) => (
          <div key={i} className="flex items-center justify-center">
            {day ? (
              <button
                onClick={() => selectDay(day)}
                className={cn(
                  'w-9 h-9 rounded-full text-sm font-medium transition-colors',
                  isSelected(day) ? 'bg-primary text-white' :
                  isToday(day) ? 'bg-primary-lighter text-primary' :
                  'text-dark hover:bg-bg-alt'
                )}
              >
                {day}
              </button>
            ) : <div className="w-9 h-9" />}
          </div>
        ))}
      </div>

      {onClose && (
        <div className="mt-4 flex gap-3">
          <Button variant="secondary" fullWidth size="sm" onClick={onClose}>Cancel</Button>
          <Button fullWidth size="sm" onClick={onClose}>Done</Button>
        </div>
      )}
    </div>
  )
}
