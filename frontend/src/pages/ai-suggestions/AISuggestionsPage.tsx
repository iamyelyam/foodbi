import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CardSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Sparkles, TrendingUp, DollarSign, ShoppingCart, Plus, CheckSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import api from '@/lib/api'

const typeConfig: Record<string, { icon: typeof Sparkles; color: string }> = {
  menu_optimization: { icon: TrendingUp, color: 'text-primary bg-primary-lighter' },
  price_adjustment: { icon: DollarSign, color: 'text-warning bg-warning/10' },
  purchase_recommendation: { icon: ShoppingCart, color: 'text-info bg-info-light' },
}

const impactBadge: Record<string, string> = {
  high: 'bg-success/10 text-success',
  medium: 'bg-warning/10 text-warning',
  low: 'bg-bg-alt text-gray',
}

export function AISuggestionsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [selectedSuggestion, setSelectedSuggestion] = useState<any>(null)
  const [taskTitle, setTaskTitle] = useState('')
  const [showTasks, setShowTasks] = useState(false)

  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const { data: suggestions = [], isLoading } = useQuery({
    queryKey: ['ai-suggestions'],
    queryFn: () => api.get('/ai/suggestions').then((r) => r.data),
  })

  const { data: tasks = [] } = useQuery({
    queryKey: ['ai-tasks'],
    queryFn: () => api.get('/ai/tasks').then((r) => r.data),
  })

  const createTask = useMutation({
    mutationFn: (data: { title: string; description: string }) => api.post('/ai/tasks', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ai-tasks'] })
      setSelectedSuggestion(null)
      setTaskTitle('')
    },
  })

  const handleCreateTask = () => {
    if (!taskTitle || !selectedSuggestion) return
    createTask.mutate({ title: taskTitle, description: selectedSuggestion.description })
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="AI Suggestions" showBack showNotification badgeCount={unreadCount} />

      <div className="px-4 pt-3 pb-3 flex items-center justify-between">
        <span className="text-xs text-gray">{suggestions.length} suggestions</span>
        <button onClick={() => setShowTasks(true)} className="flex items-center gap-1 text-xs font-medium text-primary">
          <CheckSquare className="h-3.5 w-3.5" /> Tasks ({tasks.length})
        </button>
      </div>

      <main className="flex-1 px-4 pb-20 space-y-3">
        {isLoading ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
        <>
        {suggestions.map((s: any) => {
          const config = typeConfig[s.type] || { icon: Sparkles, color: 'text-primary bg-primary-lighter' }
          const Icon = config.icon
          return (
            <button
              key={s.id}
              onClick={() => navigate(`/ai-suggestions/${s.id}`)}
              className="w-full bg-white rounded-[16px] p-4 shadow-sm text-left"
            >
              <div className="flex items-start gap-3">
                <div className={cn('w-10 h-10 rounded-full flex items-center justify-center shrink-0', config.color)}>
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-semibold text-dark">{s.title}</p>
                    <span className={cn('text-[10px] px-2 py-0.5 rounded-full font-medium capitalize', impactBadge[s.impact] || impactBadge.low)}>
                      {s.impact}
                    </span>
                  </div>
                  <p className="text-xs text-gray mt-1 line-clamp-2">{s.description}</p>
                </div>
              </div>
            </button>
          )
        })}

        {suggestions.length === 0 && (
          <div className="text-center py-12">
            <Sparkles className="h-12 w-12 text-gray-light mx-auto mb-3" />
            <p className="text-sm text-gray">No suggestions yet. Sync more data with iiko.</p>
          </div>
        )}
        </>
        )}
      </main>

      <Tabbar />

      {/* Create task from suggestion */}
      <BottomSheet isOpen={!!selectedSuggestion} onClose={() => setSelectedSuggestion(null)} title="Create Task">
        {selectedSuggestion && (
          <div className="space-y-4">
            <p className="text-sm text-gray">{selectedSuggestion.description}</p>
            <Input label="Task title" value={taskTitle} onChange={(e) => setTaskTitle(e.target.value)} />
            <Button fullWidth onClick={handleCreateTask} disabled={createTask.isPending || !taskTitle}>
              <Plus className="h-4 w-4 mr-2" />
              {createTask.isPending ? 'Creating...' : 'Create Task'}
            </Button>
          </div>
        )}
      </BottomSheet>

      {/* Tasks list */}
      <BottomSheet isOpen={showTasks} onClose={() => setShowTasks(false)} title="Tasks">
        <div className="space-y-2">
          {tasks.map((t: any) => (
            <div key={t.id} className="bg-bg rounded-[12px] p-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-dark">{t.title}</p>
                <span className={cn('text-xs px-2 py-0.5 rounded-full',
                  t.status === 'done' ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'
                )}>{t.status}</span>
              </div>
              <p className="text-xs text-gray mt-1">{new Date(t.created_at).toLocaleDateString()}</p>
            </div>
          ))}
          {tasks.length === 0 && <p className="text-sm text-gray text-center py-4">No tasks yet</p>}
        </div>
      </BottomSheet>
    </div>
  )
}
