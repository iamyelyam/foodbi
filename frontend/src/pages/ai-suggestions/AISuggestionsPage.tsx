import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { LocationSwitcher } from '@/components/layout/LocationSwitcher'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { Info, Sparkles, Loader2, MapPin, ChevronDown } from 'lucide-react'
import { useCurrency } from '@/stores/app'
import api from '@/lib/api'
import { useT } from '@/i18n'
import { useAppStore } from '@/stores/app'

interface Suggestion {
  id: string
  type: string
  // i18n keys + params (canonical shape; backend renders nothing). See useT().
  title_key: string
  title_params?: Record<string, string | number>
  description_key: string
  description_params?: Record<string, string | number>
  solution_key?: string
  solution_params?: Record<string, string | number>
  impact: string
  loss_amount?: number
  gain_amount?: number
}

interface Summary {
  total_loss: number
  total_gain_with_ai: number
  date: string
}

interface SuggestionsResponse {
  summary: Summary
  suggestions: Suggestion[]
}

const formatMoney = (v: number) => Math.round(v).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

export function AISuggestionsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const t = useT()
  const cs = useCurrency()
  const [showInfo, setShowInfo] = useState(false)
  const [showLocations, setShowLocations] = useState(false)
  const { data: unreadCount = 0 } = useUnreadNotificationCount()

  const selectedLocationIds = useAppStore((s) => s.selectedLocationIds)
  const locationId = useAppStore((s) => s.activeLocationId)

  const { data: locations = [] } = useQuery<any[]>({
    queryKey: ['locations'],
    queryFn: () => api.get('/locations').then((r) => r.data),
  })

  const locationName = (() => {
    if (locations.length === 0) return t('common.loading')
    if (selectedLocationIds.length === 0 || selectedLocationIds.length === locations.length)
      return t('location.allLocations')
    if (selectedLocationIds.length === 1) {
      const sel = locations.find((l: any) => l.id === selectedLocationIds[0])
      return sel?.name || t('common.loading')
    }
    return t('location.multipleLocations', { count: selectedLocationIds.length })
  })()

  const { data, isLoading } = useQuery<SuggestionsResponse>({
    queryKey: ['ai-suggestions', selectedLocationIds],
    queryFn: () => api.get('/ai/suggestions', {
      params: selectedLocationIds.length > 0 ? { location_ids: selectedLocationIds.join(',') } : {},
    }).then((r) => r.data),
  })

  const generateMutation = useMutation({
    mutationFn: () =>
      api.post('/ai/suggestions/generate', null, { params: { location_id: selectedLocationIds.length === 1 ? selectedLocationIds[0] : locationId } }).then((r) => r.data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['ai-suggestions', selectedLocationIds] }),
  })

  const summary = data?.summary
  const suggestions = data?.suggestions ?? []

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="AI Suggestions" showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-3 pb-24 space-y-3">
        {/* Location picker */}
        {locations.length > 1 && (
          <button
            onClick={() => setShowLocations(true)}
            className="flex items-center gap-1.5 px-1"
          >
            <MapPin className="h-4 w-4 text-primary" />
            <span className="text-sm font-medium text-dark">{locationName}</span>
            <ChevronDown className="h-3.5 w-3.5 text-gray" />
          </button>
        )}

        {/* Date + summary card */}
        <div className="bg-white rounded-[16px] p-4 space-y-3 shadow-sm">
          <div className="flex justify-end -mb-1">
            <button
              onClick={() => setShowInfo(true)}
              aria-label="What is this?"
              className="w-7 h-7 rounded-full bg-bg flex items-center justify-center"
            >
              <Info className="h-4 w-4 text-gray" />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <p className="text-xs text-gray mb-1">Total Loss</p>
              <p className="text-xl font-bold text-danger">
                {summary && summary.total_loss > 0 ? `-${formatMoney(summary.total_loss)}${cs}` : `0${cs}`}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray mb-1">Total gain with AI</p>
              <p className="text-xl font-bold text-dark">
                {summary ? `${formatMoney(summary.total_gain_with_ai)}${cs}` : `0${cs}`}
              </p>
            </div>
          </div>
        </div>

        {/* Suggestions */}
        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : suggestions.length === 0 ? (
          <div className="text-center py-12">
            <Sparkles className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">{t('ai.noSuggestions')}</p>
            {locationId && (
              <button
                onClick={() => generateMutation.mutate()}
                disabled={generateMutation.isPending}
                className="mt-4 bg-primary text-dark font-semibold py-3 px-6 rounded-full active:opacity-80 disabled:opacity-60 inline-flex items-center gap-2"
              >
                {generateMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t('ai.generating')}
                  </>
                ) : (
                  <>
                    <Sparkles className="h-4 w-4" />
                    {t('ai.generateButton')}
                  </>
                )}
              </button>
            )}
          </div>
        ) : (
          suggestions.map((s) => (
            <SuggestionCard
              key={s.id}
              s={s}
              cs={cs}
              t={t}
              onFix={() =>
                navigate(`/ai-suggestions/${s.id}`, { state: { suggestion: s, summary } })
              }
            />
          ))
        )}
      </main>

      <Tabbar />

      {/* Info BottomSheet — explains how amounts are computed */}
      <BottomSheet isOpen={showInfo} onClose={() => setShowInfo(false)} title="How is this calculated?">
        <div className="space-y-3 text-sm text-dark">
          <p><b>Total Loss</b> — sum of estimated money slipping away today: data errors on stock, low margins, items at risk of write-off.</p>
          <p><b>Total gain with AI</b> — sum of estimated upside if you act on every suggestion (price negotiations, promoting top sellers, etc.).</p>
          <p className="text-xs text-gray">Numbers are rough estimates based on rules of thumb (e.g. promoting top seller bumps volume ~10%). They tell you where to look first, not exact P&amp;L.</p>
        </div>
      </BottomSheet>

      <LocationSwitcher isOpen={showLocations} onClose={() => setShowLocations(false)} />
    </div>
  )
}

function SuggestionCard({
  s,
  cs,
  onFix,
  t,
}: {
  s: Suggestion
  cs: string
  onFix: () => void
  t: (k: string, p?: Record<string, string | number>) => string
}) {
  const hasLoss = (s.loss_amount ?? 0) > 0
  const hasGain = (s.gain_amount ?? 0) > 0
  return (
    <div className="bg-white rounded-[16px] p-4 shadow-sm space-y-3">
      <div>
        <p className="text-base font-bold text-dark">{t(s.title_key, s.title_params)}</p>
        {hasLoss && (
          <div className="flex items-center justify-between mt-1">
            <span className="text-xs text-gray">{t('ai.currentLoss')}</span>
            <span className="text-sm font-bold text-danger">
              -{formatMoney(s.loss_amount!)}{cs}
            </span>
          </div>
        )}
        {hasGain && !hasLoss && (
          <div className="flex items-center justify-between mt-1">
            <span className="text-xs text-gray">{t('ai.potentialGain')}</span>
            <span className="text-sm font-bold text-success">
              +{formatMoney(s.gain_amount!)}{cs}
            </span>
          </div>
        )}
      </div>

      <p className="text-sm text-dark leading-relaxed">
        {t(s.description_key, s.description_params)}
      </p>

      <button
        onClick={onFix}
        className="w-full bg-primary text-dark font-semibold py-3 rounded-full active:opacity-80"
      >
        {t('ai.howToFix')}
      </button>
    </div>
  )
}
