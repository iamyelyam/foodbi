import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Tabbar } from '@/components/layout/Tabbar'
import { LocationSwitcher } from '@/components/layout/LocationSwitcher'
import { SegmentedControl } from '@/components/ui/segmented-control'
import { DateRangeSheet } from '@/components/layout/DateRangeSheet'
import { todayIso } from '@/lib/dateRange'
import { CardSkeleton } from '@/components/ui/skeleton'
import { useDashboard, useUnreadNotificationCount } from '@/hooks/useApi'
import { useAppStore, useCurrency } from '@/stores/app'
import { MapPin, ChevronDown, Calendar, ChevronRight, Info, AlertCircle } from 'lucide-react'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

type View = 'revenue' | 'purchase' | 'stocks'

export function DashboardPage() {
  const navigate = useNavigate()
  const t = useT()
  const [view, setView] = useState<View>('revenue')
  const [showLocations, setShowLocations] = useState(false)
  const [showDatePicker, setShowDatePicker] = useState(false)
  const [dateFrom, setDateFrom] = useState<string | undefined>()
  const [dateTo, setDateTo] = useState<string | undefined>()
  const selectedLocationIds = useAppStore((s) => s.selectedLocationIds)
  const showUploadInvoicesBanner = useAppStore((s) => s.uiPrefs.showUploadInvoicesBanner)
  const cs = useCurrency()

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  // Stock value — fetched only when Stocks view is selected
  const { data: stockItems = [] } = useQuery<any[]>({
    queryKey: ['stock-summary'],
    queryFn: () => api.get('/stock').then((r) => r.data),
    enabled: true, // always loaded so switching is instant
  })
  const stockValue = (stockItems as any[])
    .filter((i: any) => (i.amount || 0) > 0)
    .reduce((s: number, i: any) => s + (i.cost_sum || 0), 0)

  // Smart location label based on multi-select state:
  //  - 1 location in company → its name (no selector value matters)
  //  - 0 selected (or all selected) → "All locations"
  //  - 1 selected → its name
  //  - >1 (subset) → "N locations"
  let locationName = t('common.loading')
  if (locations.length === 0) {
    locationName = t('location.noLocation')
  } else if (locations.length === 1) {
    locationName = locations[0].name
  } else if (selectedLocationIds.length === 0 || selectedLocationIds.length === locations.length) {
    locationName = t('location.allLocations')
  } else if (selectedLocationIds.length === 1) {
    const sel = locations.find((l: any) => l.id === selectedLocationIds[0])
    locationName = sel?.name || t('common.loading')
  } else {
    locationName = t('location.multipleLocations', { count: selectedLocationIds.length })
  }

  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: summary, isLoading } = useDashboard(dateFrom, dateTo)

  const today = new Date()
  // UI date formatting follows the user's chosen UI language, not the company's
  // currency locale (which is fixed to ru-KZ for this restaurant).
  const locale = useI18nStore((s) => s.locale)
  const dateStr = today.toLocaleDateString(locale, { month: 'long', day: 'numeric' })

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      {/* Header */}
      <header className="flex items-center justify-between px-4 h-14">
        <button onClick={() => setShowLocations(true)} className="flex items-center gap-1.5">
          <MapPin className="h-6 w-6 text-dark" strokeWidth={1.5} />
          <span className="text-base text-dark">{locationName}</span>
          <ChevronDown className="h-4 w-4 text-gray" />
        </button>
        {/* Notifications hidden globally — see NOTIFICATIONS_ENABLED in components/layout/Header.tsx.
            Re-enable by uncommenting and importing { Bell } again. */}
        <span aria-hidden />
        {false && unreadCount > 0 && (
          <button onClick={() => navigate('/notifications')} className="relative p-1" />
        )}
      </header>

      <main className="flex-1 px-4 pb-[100px] space-y-4">
        {/* Segmented Control — 3 tabs */}
        <SegmentedControl
          value={view}
          onChange={setView}
          options={[
            { value: 'revenue', label: t('dashboard.revenue') },
            { value: 'purchase', label: t('dashboard.purchases') },
            { value: 'stocks', label: t('dashboard.stockManagement') },
          ]}
        />

        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
          <>
            {/* Metrics Card */}
            <div className="bg-bg-alt rounded-[16px] p-4">
              {/* Date row */}
              <div className="flex items-center justify-between mb-4">
                <button onClick={() => setShowDatePicker(true)} className="flex items-center gap-1.5">
                  <Calendar className="h-4 w-4 text-gray" strokeWidth={1.5} />
                  <span className="text-xs font-medium text-gray">
                    {dateFrom && dateTo
                      ? `${new Date(dateFrom + 'T00:00:00').toLocaleDateString(locale, { day: '2-digit', month: '2-digit' })} – ${new Date(dateTo + 'T00:00:00').toLocaleDateString(locale, { day: '2-digit', month: '2-digit' })}`
                      : `${t('common.today')}, ${dateStr}`}
                  </span>
                  <ChevronRight className="h-4 w-4 text-gray" />
                </button>
                <Info className="h-6 w-6 text-bg-alt stroke-[#dddee1]" strokeWidth={1.5} />
              </div>

              {/* Big metric — depends on selected view (revenue / purchases / stocks) */}
              {(() => {
                const fmt = (v: number) => v.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })
                const revenue = summary?.today_revenue ?? 0
                const purchases = summary?.today_purchases ?? 0
                const profit = summary?.today_profit ?? 0
                let title = t('dashboard.totalRevenue')
                let bigValue = revenue
                if (view === 'purchase') {
                  title = t('dashboard.purchases')
                  bigValue = purchases
                } else if (view === 'stocks') {
                  title = t('dashboard.stockManagement')
                  bigValue = stockValue
                }
                return (
                  <>
                    <div className="mb-3">
                      <p className="text-xs font-semibold text-dark-alt">{title}</p>
                      <p className="text-4xl font-extrabold text-black mt-0.5">
                        {fmt(bigValue)}{cs}
                      </p>
                    </div>

                    <div className="space-y-1.5">
                      {/* Sub-row 1: opposing metric */}
                      {view === 'revenue' && (
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-[#606060]">{t('dashboard.currentDayLoss')}</span>
                          <span className="text-xs text-danger">
                            {purchases > 0 ? '-' : ''}{fmt(purchases)}{cs}
                          </span>
                        </div>
                      )}
                      {view === 'purchase' && (
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-[#606060]">{t('dashboard.totalRevenue')}</span>
                          <span className="text-xs text-success-alt">
                            +{fmt(revenue)}{cs}
                          </span>
                        </div>
                      )}

                      {/* Sub-row 2: profit (always shown for revenue/purchase) */}
                      {view !== 'stocks' && (
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-[#606060]">{t('dashboard.allTimeGain')}</span>
                          <span className={`text-xs ${profit >= 0 ? 'text-success-alt' : 'text-danger'}`}>
                            {profit >= 0 ? '+' : ''}{fmt(profit)}{cs}
                          </span>
                        </div>
                      )}
                    </div>
                  </>
                )
              })()}
            </div>

            {/* Upload invoices banner — toggleable from Profile → Settings */}
            {showUploadInvoicesBanner && (
              <button
                onClick={() => navigate('/file-upload')}
                className="w-full flex items-center gap-3 bg-white border border-bg-alt rounded-[16px] px-4 py-3"
              >
                <div className="w-10 h-10 rounded-full bg-warning/10 flex items-center justify-center shrink-0">
                  <AlertCircle className="h-5 w-5 text-warning" />
                </div>
                <div className="flex-1 text-left">
                  <p className="text-[15px] font-medium text-dark">{t('dashboard.uploadInvoices')}</p>
                  <p className="text-[15px] text-[#797979]">{t('dashboard.uploadInvoicesDesc')}</p>
                </div>
                <div className="bg-primary rounded-[10px] px-4 py-1.5">
                  <span className="text-base font-medium text-black">{t('common.upload')}</span>
                </div>
              </button>
            )}

            {/* Activities */}
            <div>
              <h2 className="text-xl font-medium text-black mb-3">{t('dashboard.activities')}</h2>

              <div className="flex gap-3">
                {/* AI Suggestions — tall card */}
                <button
                  onClick={() => navigate('/ai-suggestions')}
                  className="w-[156px] shrink-0 rounded-[20px] bg-primary p-4 flex flex-col relative overflow-hidden text-left"
                  style={{ height: 216 }}
                >
                  <span className="absolute top-4 right-4 bg-[#FFEA13] rounded-[10px] px-3 py-1 text-xs font-semibold text-black z-10">
                    12
                  </span>
                  <p className="text-[20px] font-bold text-black leading-[1.15]">{t('dashboard.aiSuggestions').split(' ').map((word, i) => <span key={i}>{i > 0 && <br />}{word}</span>)}</p>
                  <p className="text-xs text-black mt-2 leading-snug">{t('dashboard.aiSuggestionsDesc')}</p>
                  <img
                    src="/illustrations/lightbulb-ai.png"
                    alt=""
                    className="absolute -bottom-1 -right-1 w-[100px] h-[100px] object-contain"
                  />
                </button>

                {/* Right column — Revenue + Purchases */}
                <div className="flex-1 flex flex-col gap-3">
                  {/* Revenue card */}
                  <button
                    onClick={() => navigate('/revenue')}
                    className="flex-1 rounded-[20px] bg-bg-alt p-4 overflow-hidden relative text-left flex flex-col justify-start items-start"
                    style={{ height: 100 }}
                  >
                    <p className="text-[20px] font-bold text-black relative z-10">{t('dashboard.revenue')}</p>
                    <p className="text-xs text-black mt-1 relative z-10">{t('common.moreDetails')}</p>
                    <img
                      src="/illustrations/money-revenue.png"
                      alt=""
                      className="absolute right-2 bottom-1 w-[70px] h-[70px] object-contain"
                    />
                  </button>

                  {/* Purchases card */}
                  <button
                    onClick={() => navigate('/purchases')}
                    className="flex-1 rounded-[20px] bg-bg-alt p-4 overflow-hidden relative text-left flex flex-col justify-start items-start"
                    style={{ height: 100 }}
                  >
                    <p className="text-[20px] font-bold text-black relative z-10">{t('dashboard.purchases')}</p>
                    <p className="text-xs text-black mt-1 relative z-10">{t('common.moreDetails')}</p>
                    <img
                      src="/illustrations/purchases-grocery.png"
                      alt=""
                      className="absolute right-1 -bottom-1 w-[75px] h-[75px] object-contain"
                    />
                  </button>
                </div>
              </div>

              {/* Stock management — wide card */}
              <button
                onClick={() => navigate('/stock')}
                className="w-full mt-3 rounded-[20px] bg-bg-alt p-4 overflow-hidden relative text-left flex flex-col justify-start items-start"
                style={{ height: 100 }}
              >
                <p className="text-[20px] font-bold text-black relative z-10">{t('dashboard.stockManagement')}</p>
                <p className="text-xs text-black mt-1 relative z-10">{t('common.moreDetails')}</p>
                <img
                  src="/illustrations/stock-management.png"
                  alt=""
                  className="absolute right-4 bottom-0 w-[100px] h-[80px] object-contain"
                />
              </button>
            </div>
          </>
        )}
      </main>

      <Tabbar />
      <LocationSwitcher isOpen={showLocations} onClose={() => setShowLocations(false)} />

      <DateRangeSheet
        isOpen={showDatePicker}
        onClose={() => setShowDatePicker(false)}
        from={dateFrom ?? todayIso()}
        to={dateTo ?? todayIso()}
        onChange={(f, t) => { setDateFrom(f); setDateTo(t) }}
      />
    </div>
  )
}
