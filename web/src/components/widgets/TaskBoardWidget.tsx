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
          ...todo,
          status: todo.status === 'done' ? 'not_started' : 'done',
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
    mutationFn: (title: string) =>
      apiFetch<Todo>('/api/todos', {
        method: 'POST',
        body: JSON.stringify({ title, status: 'not_started', priority: 'medium', list_id: listId }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todos'] }),
  })
}

export default function TaskBoardWidget() {
  const { selectedListId } = useTasksStore()
  const { data, isLoading, error } = useTodos(selectedListId)
  const toggle = useToggleTodo()
  const del = useDeleteTodo()
  const create = useCreateTodo(selectedListId)
  const [draft, setDraft] = useState('')

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
    create.mutate(title, { onSuccess: () => setDraft('') })
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

      <div className="flex gap-2" style={{ padding: '8px 12px', background: 'var(--color-surface)' }}>
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
          onClick={handleAdd}
          disabled={!draft.trim() || create.isPending}
          style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
        >
          +
        </button>
      </div>
    </div>
  )
}
