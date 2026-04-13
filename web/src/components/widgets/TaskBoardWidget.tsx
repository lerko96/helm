import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Todo } from '../../lib/types'
import { useTasksStore } from '../../stores/tasksStore'

const STATUS_ORDER: Todo['status'][] = ['not_started', 'in_progress', 'done']
const STATUS_LABELS: Record<Todo['status'], string> = {
  not_started: 'Not Started',
  in_progress: 'In Progress',
  done: 'Done',
}

const PRIORITY_COLOR: Record<Todo['priority'], string> = {
  low: 'var(--color-text-dim)',
  medium: 'var(--color-text-label)',
  high: 'var(--color-accent-red)',
}

function useTodos(listId: number | null) {
  return useQuery({
    queryKey: ['todos', listId],
    queryFn: () => {
      const qs = listId != null ? `?list_id=${listId}` : ''
      return apiFetch<Todo[]>(`/api/todos${qs}`)
    },
  })
}

function useToggleTodo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (todo: Todo) =>
      apiFetch<Todo>(`/api/todos/${todo.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          list_id: todo.list_id,
          title: todo.title,
          description: todo.description,
          status: todo.status === 'done' ? 'not_started' : 'done',
          priority: todo.priority,
          due_date: todo.due_date,
          is_pinned: todo.is_pinned,
        }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['todos'] })
    },
  })
}

function useDeleteTodo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<void>(`/api/todos/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todos'] }),
  })
}

function useCreateTodo(listId: number | null) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ title, due_date }: { title: string; due_date?: string }) =>
      apiFetch<Todo>('/api/todos', {
        method: 'POST',
        body: JSON.stringify({ title, status: 'not_started', priority: 'medium', list_id: listId, due_date: due_date || undefined }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todos'] }),
  })
}

function formatDue(due: string) {
  return new Date(due).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

export default function TaskBoardWidget() {
  const { selectedListId } = useTasksStore()
  const { data, isLoading, error } = useTodos(selectedListId)
  const toggle = useToggleTodo()
  const del = useDeleteTodo()
  const create = useCreateTodo(selectedListId)
  const [draft, setDraft] = useState('')
  const [dueDraft, setDueDraft] = useState('')
  const [showDate, setShowDate] = useState(false)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} style={{ height: '32px', background: 'var(--color-surface-raised)' }} />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ padding: '12px' }}>
        <span className="status status-alert">{(error as Error).message}</span>
      </div>
    )
  }

  const todos = data ?? []

  const grouped = STATUS_ORDER.reduce<Record<Todo['status'], Todo[]>>(
    (acc, s) => ({ ...acc, [s]: todos.filter(t => t.status === s) }),
    { not_started: [], in_progress: [], done: [] }
  )

  function handleAdd() {
    const title = draft.trim()
    if (!title) return
    create.mutate(
      { title, due_date: dueDraft || undefined },
      { onSuccess: () => { setDraft(''); setDueDraft(''); setShowDate(false) } }
    )
  }

  return (
    <div className="flex flex-col">
      {STATUS_ORDER.map(status => {
        const group = grouped[status]
        return (
          <div key={status}>
            <div
              style={{
                padding: '6px 12px',
                background: 'var(--color-surface)',
                borderBottom: '1px solid var(--color-border)',
                fontSize: 'var(--text-xs)',
                letterSpacing: 'var(--letter-spacing-label)',
                textTransform: 'uppercase',
                color: 'var(--color-text-label)',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              {STATUS_LABELS[status]}
              <span style={{ color: 'var(--color-text-dim)' }}>{group.length}</span>
            </div>
            {group.map(todo => {
              const done = todo.status === 'done'
              return (
                <div
                  key={todo.id}
                  className="flex items-center gap-2"
                  style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)' }}
                >
                  <span
                    style={{ width: '6px', height: '6px', background: PRIORITY_COLOR[todo.priority], display: 'inline-block', flexShrink: 0 }}
                  />
                  <button
                    onClick={() => toggle.mutate(todo)}
                    style={{
                      width: '16px',
                      height: '16px',
                      border: '1px solid var(--color-border-bright)',
                      background: done ? 'var(--color-text-primary)' : 'transparent',
                      color: done ? 'var(--color-bg)' : 'transparent',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                      fontSize: '10px',
                      padding: 0,
                    }}
                  >
                    {done ? '✓' : ''}
                  </button>
                  <span
                    style={{
                      fontSize: 'var(--text-sm)',
                      flex: 1,
                      textDecoration: done ? 'line-through' : 'none',
                      color: done ? 'var(--color-text-dim)' : 'var(--color-text-primary)',
                    }}
                  >
                    {todo.title}
                  </span>
                  {todo.due_date && (
                    <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)', flexShrink: 0 }}>
                      {formatDue(todo.due_date)}
                    </span>
                  )}
                  <button
                    onClick={() => del.mutate(todo.id)}
                    style={{
                      background: 'transparent',
                      border: 'none',
                      color: 'var(--color-text-dim)',
                      fontSize: '12px',
                      padding: '0 4px',
                      cursor: 'pointer',
                    }}
                    onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
                    onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
                  >
                    ×
                  </button>
                </div>
              )
            })}
          </div>
        )
      })}

      <div className="flex flex-col gap-2" style={{ padding: '8px 12px', background: 'var(--color-surface)' }}>
        <div className="flex gap-2">
          <input
            type="text"
            value={draft}
            onChange={e => setDraft(e.target.value)}
            placeholder="add task..."
            style={{ flex: 1, fontSize: 'var(--text-sm)' }}
            onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
          />
          <button
            className="btn-ghost"
            onClick={() => setShowDate(v => !v)}
            style={{ fontSize: 'var(--text-xs)', padding: '6px 8px', color: showDate ? 'var(--color-text-primary)' : 'var(--color-text-dim)' }}
          >
            ◷
          </button>
          <button
            className="btn-ghost"
            onClick={handleAdd}
            disabled={!draft.trim() || create.isPending}
            style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
          >
            +
          </button>
        </div>
        {showDate && (
          <input
            type="date"
            value={dueDraft}
            onChange={e => setDueDraft(e.target.value)}
            style={{ fontSize: 'var(--text-sm)', width: '100%' }}
          />
        )}
      </div>
    </div>
  )
}
