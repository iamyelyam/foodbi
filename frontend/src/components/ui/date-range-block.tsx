import { useState } from 'react'
import { Calendar } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useT } from '@/i18n'
import {
  PRESETS,
  presetRange,
  isPresetActive,
  formatInputDate,
} from '@/lib/dateRange'
import { DatePicker } from './date-picker'

interface Props {
  from: string // YYYY-MM-DD
  to: string
  onChange: (from: string, to: string) => void
  /** When true, hides the inner section title (caller already shows one) */
  hideTitle?: boolean
}

/**
 * Embeddable date-range section: 4 preset chips (Today / Yesterday / This week /
 * This month) and a From/To row with a calendar overlay. Used by DateRangeSheet
 * and any page that needs date selection nested inside a larger filters BottomSheet.
 *
 * Custom date selection: clicking From or To opens an inline DatePicker; the picked
 * date replaces only that side, leaving the other intact (so users can build a
 * custom range incrementally).
 */
export function DateRangeBlock({ from, to, onChange, hideTitle }: Props) {
  const t = useT()
  const [openSide, setOpenSide] = useState<'from' | 'to' | null>(null)

  const handleDayPick = (date: string) => {
    if (openSide === 'from') {
      // If the new from is after current to, snap to to to date.
      const newTo = date > to ? date : to
      onChange(date, newTo)
    } else if (openSide === 'to') {
      const newFrom = date < from ? date : from
      onChange(newFrom, date)
    }
    setOpenSide(null)
  }

  return (
    <div>
      {!hideTitle && (
        <p className="text-base font-bold text-dark mb-3">{t('date.title')}</p>
      )}

      {/* Preset chips */}
      <div className="space-y-2">
        {PRESETS.map(({ key, labelKey }) => {
          const active = isPresetActive(key, from, to)
          return (
            <button
              key={key}
              onClick={() => {
                const [f, ttt] = presetRange(key)
                onChange(f, ttt)
              }}
              className={cn(
                'w-full py-3 rounded-[12px] text-sm font-medium transition-colors',
                active
                  ? 'bg-primary-lighter text-dark border-2 border-primary'
                  : 'bg-bg text-dark'
              )}
            >
              {t(labelKey)}
            </button>
          )
        })}
      </div>

      {/* From / To inputs */}
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 mt-3">
        <div>
          <p className="text-xs text-gray mb-1">{t('date.from')}</p>
          <button
            onClick={() => setOpenSide('from')}
            className={cn(
              'w-full bg-bg rounded-[10px] px-3 py-2 flex items-center gap-1.5 text-xs text-dark',
              openSide === 'from' && 'ring-2 ring-primary'
            )}
          >
            <Calendar className="h-3.5 w-3.5 text-gray" />
            <span>{formatInputDate(from)}</span>
          </button>
        </div>
        <span className="text-gray mt-4">—</span>
        <div>
          <p className="text-xs text-gray mb-1">{t('date.to')}</p>
          <button
            onClick={() => setOpenSide('to')}
            className={cn(
              'w-full bg-bg rounded-[10px] px-3 py-2 flex items-center gap-1.5 text-xs text-dark',
              openSide === 'to' && 'ring-2 ring-primary'
            )}
          >
            <Calendar className="h-3.5 w-3.5 text-gray" />
            <span>{formatInputDate(to)}</span>
          </button>
        </div>
      </div>

      {/* Inline calendar — appears below the inputs once From or To is clicked */}
      {openSide && (
        <div className="mt-3">
          <DatePicker
            value={openSide === 'from' ? from : to}
            onChange={handleDayPick}
          />
          <button
            onClick={() => setOpenSide(null)}
            className="mt-2 w-full text-center text-primary text-sm font-semibold py-1"
          >
            {t('common.cancel')}
          </button>
        </div>
      )}
    </div>
  )
}
