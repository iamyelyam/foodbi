import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Snackbar } from '@/components/ui/snackbar'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Sparkles, TrendingUp, DollarSign, ShoppingCart, Plus, User } from 'lucide-react'
import { cn } from '@/lib/utils'
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

export function AISuggestionDetailPage() {
  const { id } = useParams()
  const queryClient = useQueryClient()
  const [showCreateTask, setShowCreateTask] = useState(false)
  const [taskTitle, setTaskTitle] = useState('')
  const [selectedEmployee, setSelectedEmployee] = useState<string>('')
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; type: 'success' | 'error' }>({ open: false, message: '', type: 'success' })

  const { data: suggestion, isLoading } = useQuery({
    queryKey: ['ai-suggestion', id],
    queryFn: () => api.get(`/ai/suggestions/${id}`).then((r) => r.data),
    enabled: !!id,
  })

  const { data: employees = [] } = useQuery({
    queryKey: ['employees'],
    queryFn: () => api.get('/employees').then((r) => r.data),
    enabled: showCreateTask,
  })

  const createTask = useMutation({
    mutationFn: (data: { title: string; description: string; assignee_id: string }) =>
      api.post('/ai/tasks', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ai-tasks'] })
      setShowCreateTask(false)
      setTaskTitle('')
      setSelectedEmployee('')
      setSnackbar({ open: true, message: 'Task created successfully', type: 'success' })
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to create task', type: 'error' })
    },
  })

  const handleCreateTask = () => {
    if (!taskTitle || !suggestion) return
    createTask.mutate({
      title: taskTitle,
      description: suggestion.description,
      assignee_id: selectedEmployee,
    })
  }

  const config = suggestion ? (typeConfig[suggestion.type] || { icon: Sparkles, color: 'text-primary bg-primary-lighter' }) : null
  const Icon = config?.icon || Sparkles

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="Suggestion" showBack />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-3">
        {isLoading ? (
          <><ListItemSkeleton /><ListItemSkeleton /></>
        ) : suggestion ? (
          <>
            {/* Header card */}
            <div className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center">
              <div className={cn('w-16 h-16 rounded-full flex items-center justify-center', config?.color)}>
                <Icon className="h-8 w-8" />
              </div>
              <p className="mt-3 text-lg font-bold text-dark text-center">{suggestion.title}</p>
              <div className="flex items-center gap-2 mt-2">
                <span className={cn('text-xs px-3 py-1 rounded-full font-medium capitalize', impactBadge[suggestion.impact] || impactBadge.low)}>
                  {suggestion.impact} impact
                </span>
                <span className="text-xs px-3 py-1 rounded-full bg-bg-alt text-gray font-medium capitalize">
                  {suggestion.type?.replace(/_/g, ' ')}
                </span>
              </div>
            </div>

            {/* Description */}
            <div className="bg-white rounded-[16px] p-4 shadow-sm">
              <p className="text-sm font-semibold text-dark mb-2">Description</p>
              <p className="text-sm text-gray leading-relaxed">{suggestion.description}</p>
            </div>

            {/* Create Task button */}
            <Button fullWidth onClick={() => { setShowCreateTask(true); setTaskTitle(suggestion.title) }}>
              <Plus className="h-4 w-4 mr-2" /> Create Task
            </Button>
          </>
        ) : (
          <p className="text-center text-sm text-gray py-12">Suggestion not found</p>
        )}
      </main>

      <Tabbar />

      {/* Create task BottomSheet */}
      <BottomSheet isOpen={showCreateTask} onClose={() => setShowCreateTask(false)} title="Create Task">
        <div className="space-y-4">
          <Input label="Task title" value={taskTitle} onChange={(e) => setTaskTitle(e.target.value)} />

          <div>
            <p className="text-xs font-medium text-gray mb-2">Assign to</p>
            <div className="max-h-48 overflow-y-auto space-y-1 rounded-[12px] border border-bg-alt p-2">
              <button
                onClick={() => setSelectedEmployee('')}
                className={cn(
                  'w-full flex items-center gap-3 px-3 py-2 rounded-[8px] text-sm transition-colors',
                  !selectedEmployee ? 'bg-primary/10 text-primary font-medium' : 'text-dark'
                )}
              >
                <User className="h-4 w-4" /> Unassigned
              </button>
              {employees.map((emp: any) => (
                <button
                  key={emp.id}
                  onClick={() => setSelectedEmployee(emp.id)}
                  className={cn(
                    'w-full flex items-center gap-3 px-3 py-2 rounded-[8px] text-sm transition-colors',
                    selectedEmployee === emp.id ? 'bg-primary/10 text-primary font-medium' : 'text-dark'
                  )}
                >
                  <User className="h-4 w-4" /> {emp.first_name} {emp.last_name}
                </button>
              ))}
            </div>
          </div>

          <Button fullWidth onClick={handleCreateTask} disabled={createTask.isPending || !taskTitle}>
            {createTask.isPending ? 'Creating...' : 'Create Task'}
          </Button>
        </div>
      </BottomSheet>

      <Snackbar
        message={snackbar.message}
        type={snackbar.type}
        isOpen={snackbar.open}
        onClose={() => setSnackbar((s) => ({ ...s, open: false }))}
      />
    </div>
  )
}
