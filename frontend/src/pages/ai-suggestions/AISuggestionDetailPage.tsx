import { useState } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Calendar, ChevronRight, Info, User } from 'lucide-react'
import { useCurrency } from '@/stores/app'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import { cn } from '@/lib/utils'
import api from '@/lib/api'
import { findRoleLabel } from '@/lib/employeeRoles'
import { useT } from '@/i18n'

interface Suggestion {
  id: string
  type: string
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

const formatMoney = (v: number) => Math.round(v).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })

function formatHeaderDate(iso?: string): string {
  const d = iso ? new Date(iso + 'T00:00:00') : new Date()
  return `Today, ${d.toLocaleDateString('en-US', { month: 'long', day: 'numeric' })}`
}

// Build the WhatsApp deep link. Without a phone number, wa.me opens the native
// contact picker so the user chooses who to send to. With a phone, it pre-fills
// that contact. Text is URL-encoded.
function whatsappShareUrl(text: string, phone?: string): string {
  const encoded = encodeURIComponent(text)
  if (phone && phone.replace(/\D/g, '').length >= 8) {
    return `https://wa.me/${phone.replace(/\D/g, '')}?text=${encoded}`
  }
  return `https://api.whatsapp.com/send?text=${encoded}`
}

function buildTaskMessage(
  s: Suggestion,
  t: (k: string, p?: Record<string, string | number>) => string
): string {
  const title = t(s.title_key, s.title_params)
  const body = s.solution_key
    ? t(s.solution_key, s.solution_params)
    : t(s.description_key, s.description_params)
  const lines = [`📋 ${title}`, '', body]
  if (s.loss_amount && s.loss_amount > 0) {
    lines.push('', `💸 ${t('ai.currentLoss')}: ${formatMoney(s.loss_amount)} KZT`)
  } else if (s.gain_amount && s.gain_amount > 0) {
    lines.push('', `💰 ${t('ai.potentialGain')}: ${formatMoney(s.gain_amount)} KZT`)
  }
  lines.push('', t('ai.signature'))
  return lines.join('\n')
}

export function AISuggestionDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const t = useT()
  const cs = useCurrency()
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const [showInfo, setShowInfo] = useState(false)
  const [showAssign, setShowAssign] = useState(false)

  // Read suggestion + summary from navigation state (set by the list page).
  // Fallback: re-fetch the list and find by id (handles refresh / direct nav).
  const stateData = location.state as { suggestion?: Suggestion; summary?: Summary } | undefined
  const fetchedFallback = useQuery<{ suggestion: Suggestion; summary: Summary } | null>({
    queryKey: ['ai-suggestions-fallback', id],
    enabled: !stateData?.suggestion && !!id,
    queryFn: async () => {
      const data = await api.get('/ai/suggestions').then((r) => r.data)
      const found = (data?.suggestions ?? []).find((s: Suggestion) => s.id === id)
      return found ? { suggestion: found, summary: data.summary } : null
    },
  })

  const suggestion = stateData?.suggestion ?? fetchedFallback.data?.suggestion ?? null
  const summary = stateData?.summary ?? fetchedFallback.data?.summary

  const { data: employees = [] } = useQuery({
    queryKey: ['employees'],
    queryFn: () => api.get('/employees').then((r) => r.data),
    enabled: showAssign,
  })

  const sendToWhatsApp = (phone?: string) => {
    if (!suggestion) return
    const url = whatsappShareUrl(buildTaskMessage(suggestion, t), phone)
    window.open(url, '_blank', 'noopener,noreferrer')
    setShowAssign(false)
  }

  if (!suggestion && fetchedFallback.isLoading) {
    return (
      <div className="flex flex-col min-h-dvh bg-bg">
        <Header title="AI Suggestion" showBack />
        <main className="flex-1 px-4 pt-4 space-y-3">
          <ListItemSkeleton />
          <ListItemSkeleton />
        </main>
        <Tabbar />
      </div>
    )
  }

  if (!suggestion) {
    return (
      <div className="flex flex-col min-h-dvh bg-bg">
        <Header title="AI Suggestion" showBack />
        <main className="flex-1 px-4 pt-4">
          <p className="text-center text-sm text-gray py-12">Suggestion not found.</p>
          <button
            onClick={() => navigate('/ai-suggestions')}
            className="mt-2 w-full text-center text-primary font-semibold py-2"
          >
            Back to suggestions
          </button>
        </main>
        <Tabbar />
      </div>
    )
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="AI Suggestion" showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-3 pb-24 space-y-4">
        {/* Summary card — same shape as list page header */}
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

        {/* Suggestion title — moved here so it has room to breathe */}
        <h2 className="text-xl font-bold text-dark leading-tight pt-1">
          {t(suggestion.title_key, suggestion.title_params)}
        </h2>

        {/* Description */}
        <section>
          <h3 className="text-base font-bold text-dark mb-2">Description</h3>
          <p className="text-sm text-dark leading-relaxed">
            {t(suggestion.description_key, suggestion.description_params)}
          </p>
        </section>

        <hr className="border-bg-alt" />

        {/* Solution */}
        <section>
          <h3 className="text-base font-bold text-dark mb-2">Solution</h3>
          <p className="text-sm text-dark leading-relaxed">
            {suggestion.solution_key
              ? t(suggestion.solution_key, suggestion.solution_params)
              : t('ai.fallbackSolution')}
          </p>
        </section>
      </main>

      {/* Sticky bottom CTA */}
      <div className="fixed bottom-16 inset-x-0 px-4 pb-4 pt-2 bg-bg/95 backdrop-blur">
        <button
          onClick={() => setShowAssign(true)}
          className="w-full bg-primary text-dark font-semibold py-3 rounded-full active:opacity-80"
        >
          Create task
        </button>
      </div>

      <Tabbar />

      {/* Info BottomSheet */}
      <BottomSheet isOpen={showInfo} onClose={() => setShowInfo(false)} title="How is this calculated?">
        <div className="space-y-3 text-sm text-dark">
          <p><b>Total Loss</b> — sum of estimated money slipping away today: data errors on stock, low margins, items at risk of write-off.</p>
          <p><b>Total gain with AI</b> — sum of estimated upside if you act on every suggestion (price negotiations, promoting top sellers, etc.).</p>
          <p className="text-xs text-gray">Numbers are rough estimates — they tell you where to look first, not exact P&amp;L.</p>
        </div>
      </BottomSheet>

      {/* Assign-to-employee BottomSheet — opens WhatsApp with the chosen phone */}
      <BottomSheet isOpen={showAssign} onClose={() => setShowAssign(false)} title="Send task to">
        <div className="space-y-2">
          <button
            onClick={() => sendToWhatsApp()}
            className="w-full flex items-center gap-3 px-3 py-3 rounded-[12px] bg-bg text-dark active:opacity-70"
          >
            <User className="h-5 w-5 text-gray" />
            <div className="flex-1 text-left">
              <p className="text-sm font-semibold">Pick contact in WhatsApp</p>
              <p className="text-xs text-gray">Opens the WhatsApp contact picker</p>
            </div>
          </button>

          {employees.length > 0 && (
            <p className="text-xs text-gray uppercase tracking-wide pt-3 px-1">From your team</p>
          )}
          {employees.map((emp: any) => {
            const phone: string | undefined = emp.phone
            const disabled = !phone
            return (
              <button
                key={emp.id}
                disabled={disabled}
                onClick={() => sendToWhatsApp(phone)}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-3 rounded-[12px] text-left',
                  disabled ? 'bg-bg/60 cursor-not-allowed' : 'bg-bg active:opacity-70'
                )}
              >
                <div className="w-10 h-10 rounded-full bg-primary-lighter flex items-center justify-center shrink-0">
                  <User className="h-5 w-5 text-primary" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className={cn('text-sm font-semibold truncate', disabled ? 'text-gray' : 'text-dark')}>
                    {emp.first_name} {emp.last_name}
                  </p>
                  <p className="text-xs text-gray truncate">
                    {findRoleLabel(emp.role)}{phone ? ` · ${phone}` : ` · ${t('ai.noPhone')}`}
                  </p>
                </div>
              </button>
            )
          })}
        </div>
      </BottomSheet>
    </div>
  )
}
