import { BottomSheet } from './BottomSheet'
import { DateRangeBlock } from '@/components/ui/date-range-block'
import { useT } from '@/i18n'

interface Props {
  isOpen: boolean
  onClose: () => void
  from: string
  to: string
  onChange: (from: string, to: string) => void
  /** Optional: shows "Show N results" on the apply button when provided */
  resultsCount?: number
}

/**
 * Standalone full-width Date BottomSheet — "Today / Yesterday / This week /
 * This month / From-To" with a primary "Show N results" CTA. Use anywhere a
 * page needs ONLY a date range (Stock, Dashboard chart, Statistics, Transfers).
 *
 * For pages where date is one filter among several (Revenue, Purchases) embed
 * <DateRangeBlock> inside the page's own larger Filters sheet instead.
 */
export function DateRangeSheet({
  isOpen,
  onClose,
  from,
  to,
  onChange,
  resultsCount,
}: Props) {
  const t = useT()
  return (
    <BottomSheet isOpen={isOpen} onClose={onClose}>
      <div className="space-y-5">
        <DateRangeBlock from={from} to={to} onChange={onChange} />
        <button
          onClick={onClose}
          className="w-full bg-primary text-dark font-bold py-3 rounded-full"
        >
          {resultsCount === undefined
            ? t('common.apply')
            : `${t('common.show')} ${resultsCount} ${t('common.results')}`}
        </button>
        <button
          onClick={onClose}
          className="w-full text-center text-primary font-semibold"
        >
          {t('common.back')}
        </button>
      </div>
    </BottomSheet>
  )
}
