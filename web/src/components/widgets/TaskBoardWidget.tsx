import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Todo } from '../../lib/types'
import { useTasksStore } from '../../stores/tasksStore'
import TagPicker from '../shared/TagPicker'

const STATUS_ORDER: Todo['status'][] = ['not_started', 'in_progress', 'done']
const STATUS_LABELS: Record<Todo['status'], string> = {
  not_started: 'NOT STARTED',
  in_progress: 'IN PROGRESS',
  done: 'DONE',
}
const STATUS_NEXT: Record<Todo['status'], Todo['status']> = {
  not_started: 'in_progress',
  in_progress: 'done',
  done: 'not_started',
}

const PRIORITY_ORDER: Todo['priority'][] = ['low', 'medium', 'high']
const PRIORITY_NEXT: Record<Todo['priority'], Todo['priority']> = {
  low: 'medium',
  medium: 'high',
  high: 'low',
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

function useTodoDetail(id: number | null) {
  return useQuery({
    queryKey: ['todos', id],
    queryFn: () => apiFetch<Todo>(`/api/todos/${id}`),
    enabled: id != null,
  })
}

type TodoUpdate = Partial<Pick<Todo, 'status' | 'priority' | 'description' | 'due_date' | 'title' | 'is_pinned'>>

function useUpdateTodo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ todo, patch }: { todo: Todo; patch: TodoUpdate }) =>
      apiFetch<Todo>(`/api/todos/${todo.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          list_id: todo.list_id,
          title: todo.title,
          description: todo.description,
          status: todo.status,
          priority: todo.priority,
          due_date: todo.due_date,
          is_pinned: todo.is_pinned,
          ...patch,
        }),
      }),
    onSuccess: (_data, { todo }) => {
      qc.invalidateQueries({ queryKey: ['todos'] })
      qc.invalidateQueries({ queryKey: ['todos', todo.id] })
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
    mutationFn: (fields: { title: string; due_date?: string; parent_id?: number }) =>
      apiFetch<Todo>('/api/todos', {
        method: 'POST',
        body: JSON.stringify({
          title: fields.title,
          status: 'not_started',
          priority: 'medium',
          list_id: listId,
          due_date: fields.due_date || undefined,
          parent_id: fields.parent_id || undefined,
        }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todos'] }),
  })
}

function formatDue(due: string) {
  return new Date(due).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

// ── Detail panel shown when a todo row is expanded ─────────────────────────

interface DetailPanelProps {
  todo: Todo
  listId: number | null
}

function DetailPanel({ todo, listId }: DetailPanelProps) {
  const { data: detail } = useTodoDetail(todo.id)
  const update = useUpdateTodo()
  const del = useDeleteTodo()
  const create = useCreateTodo(listId)
  const [descDraft, setDescDraft] = useState<string | null>(null)
  const [subtaskDraft, setSubtaskDraft] = useState('')

  const merged = detail ?? todo
  const subtasks = merged.subtasks ?? []

  function saveDesc() {
    if (descDraft === null) return
    update.mutate({ todo: merged, patch: { description: descDraft || null } })
    setDescDraft(null)
  }

  function setPriority(p: Todo['priority']) {
    update.mutate({ todo: merged, patch: { priority: p } })
  }

  function setDue(date: string) {
    update.mutate({ todo: merged, patch: { due_date: date || null } })
  }

  function addSubtask() {
    const title = subtaskDraft.trim()
    if (!title) return
    create.mutate(
      { title, parent_id: todo.id },
      { onSuccess: () => setSubtaskDraft('') }
    )
  }

  const currentDesc = descDraft !== null ? descDraft : (merged.description ?? '')
  const currentDue = merged.due_date ? merged.due_date.slice(0, 10) : ''

  return (
    <div
      style={{
        borderBottom: '1px solid var(--color-border)',
        background: 'var(--color-surface)',
        padding: '10px 12px 12px 36px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
      }}
    >
      {/* Priority selector */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: 'var(--letter-spacing-label)', width: '60px' }}>
          PRIORITY
        </span>
        {PRIORITY_ORDER.map(p => (
          <button
            key={p}
            onClick={() => setPriority(p)}
            style={{
              fontSize: '9px',
              letterSpacing: 'var(--letter-spacing-label)',
              padding: '2px 6px',
              border: '1px solid',
              borderColor: merged.priority === p ? PRIORITY_COLOR[p] : 'var(--color-border)',
              background: 'transparent',
              color: merged.priority === p ? PRIORITY_COLOR[p] : 'var(--color-text-dim)',
              cursor: 'pointer',
            }}
          >
            {p.toUpperCase()}
          </button>
        ))}
      </div>

      {/* Due date */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: 'var(--letter-spacing-label)', width: '60px' }}>
          DUE
        </span>
        <input
          type="date"
          value={currentDue}
          onChange={e => setDue(e.target.value)}
          style={{ fontSize: 'var(--text-xs)', padding: '2px 4px' }}
        />
      </div>

      {/* Description */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: 'var(--letter-spacing-label)' }}>
          DESCRIPTION
        </span>
        <textarea
          value={currentDesc}
          onChange={e => setDescDraft(e.target.value)}
          onBlur={saveDesc}
          placeholder="add description..."
          rows={2}
          style={{ fontSize: 'var(--text-sm)', resize: 'vertical', width: '100%', padding: '4px 6px' }}
        />
      </div>

      {/* Tags */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: 'var(--letter-spacing-label)', width: '60px' }}>
          TAGS
        </span>
        <TagPicker entityType="todo" entityId={todo.id} />
      </div>

      {/* Subtasks */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: 'var(--letter-spacing-label)' }}>
          SUBTASKS {subtasks.length > 0 && <span style={{ color: 'var(--color-text-dim)' }}>({subtasks.length})</span>}
        </span>
        {subtasks.map(sub => (
          <div key={sub.id} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <button
              onClick={() => update.mutate({ todo: sub, patch: { status: sub.status === 'done' ? 'not_started' : 'done' } })}
              style={{
                width: '14px',
                height: '14px',
                border: '1px solid var(--color-border-bright)',
                background: sub.status === 'done' ? 'var(--color-text-primary)' : 'transparent',
                color: sub.status === 'done' ? 'var(--color-bg)' : 'transparent',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
                fontSize: '9px',
                padding: 0,
                cursor: 'pointer',
              }}
            >
              {sub.status === 'done' ? '✓' : ''}
            </button>
            <span
              style={{
                fontSize: 'var(--text-sm)',
                flex: 1,
                textDecoration: sub.status === 'done' ? 'line-through' : 'none',
                color: sub.status === 'done' ? 'var(--color-text-dim)' : 'var(--color-text-primary)',
              }}
            >
              {sub.title}
            </span>
            <button
              onClick={() => del.mutate(sub.id)}
              style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '12px', padding: '0 4px', cursor: 'pointer' }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
            >
              ×
            </button>
          </div>
        ))}

        {/* Add subtask */}
        <div style={{ display: 'flex', gap: '4px', marginTop: '2px' }}>
          <input
            type="text"
            value={subtaskDraft}
            onChange={e => setSubtaskDraft(e.target.value)}
            placeholder="add subtask..."
            style={{ flex: 1, fontSize: 'var(--text-sm)' }}
            onKeyDown={e => { if (e.key === 'Enter') addSubtask() }}
          />
          <button
            className="btn-ghost"
            onClick={addSubtask}
            disabled={!subtaskDraft.trim() || create.isPending}
            style={{ fontSize: 'var(--text-xs)', padding: '4px 8px' }}
          >
            +
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Main widget ─────────────────────────────────────────────────────────────

export default function TaskBoardWidget() {
  const { selectedListId } = useTasksStore()
  const { data, isLoading, error } = useTodos(selectedListId)
  const update = useUpdateTodo()
  const del = useDeleteTodo()
  const create = useCreateTodo(selectedListId)
  const [draft, setDraft] = useState('')
  const [dueDraft, setDueDraft] = useState('')
  const [showDate, setShowDate] = useState(false)
  const [expandedId, setExpandedId] = useState<number | null>(null)

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

  // Only show top-level todos (no parent) in the board
  const todos = (data ?? []).filter(t => t.parent_id == null)

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

  function cyclePriority(e: React.MouseEvent, todo: Todo) {
    e.stopPropagation()
    update.mutate({ todo, patch: { priority: PRIORITY_NEXT[todo.priority] } })
  }

  function toggleDone(e: React.MouseEvent, todo: Todo) {
    e.stopPropagation()
    update.mutate({ todo, patch: { status: todo.status === 'done' ? 'not_started' : 'done' } })
  }

  function cycleStatus(e: React.MouseEvent, todo: Todo) {
    e.preventDefault()
    e.stopPropagation()
    update.mutate({ todo, patch: { status: STATUS_NEXT[todo.status] } })
  }

  function toggleExpand(id: number) {
    setExpandedId(prev => (prev === id ? null : id))
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
              const expanded = expandedId === todo.id
              return (
                <div key={todo.id}>
                  {/* Collapsed row */}
                  <div
                    className="flex items-center gap-2"
                    style={{
                      padding: '8px 12px',
                      borderBottom: expanded ? 'none' : '1px solid var(--color-border)',
                      cursor: 'default',
                      background: expanded ? 'var(--color-surface-raised)' : 'transparent',
                    }}
                  >
                    {/* Priority square — click to cycle */}
                    <span
                      title={`Priority: ${todo.priority} — click to cycle`}
                      onClick={e => cyclePriority(e, todo)}
                      style={{
                        width: '6px',
                        height: '6px',
                        background: PRIORITY_COLOR[todo.priority],
                        display: 'inline-block',
                        flexShrink: 0,
                        cursor: 'pointer',
                      }}
                    />

                    {/* Checkbox — left-click toggles done, right-click cycles status */}
                    <button
                      onClick={e => toggleDone(e, todo)}
                      onContextMenu={e => cycleStatus(e, todo)}
                      title="Left-click: done/not started — Right-click: cycle status"
                      style={{
                        width: '16px',
                        height: '16px',
                        border: '1px solid',
                        borderColor: todo.status === 'in_progress'
                          ? 'var(--color-text-label)'
                          : 'var(--color-border-bright)',
                        background: done
                          ? 'var(--color-text-primary)'
                          : todo.status === 'in_progress'
                            ? 'var(--color-surface-raised)'
                            : 'transparent',
                        color: done ? 'var(--color-bg)' : 'transparent',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        flexShrink: 0,
                        fontSize: '10px',
                        padding: 0,
                        cursor: 'pointer',
                      }}
                    >
                      {done ? '✓' : todo.status === 'in_progress' ? '·' : ''}
                    </button>

                    {/* Title — click to expand */}
                    <span
                      onClick={() => toggleExpand(todo.id)}
                      style={{
                        fontSize: 'var(--text-sm)',
                        flex: 1,
                        textDecoration: done ? 'line-through' : 'none',
                        color: done ? 'var(--color-text-dim)' : 'var(--color-text-primary)',
                        cursor: 'pointer',
                        userSelect: 'none',
                      }}
                    >
                      {todo.title}
                    </span>

                    {todo.due_date && (
                      <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)', flexShrink: 0 }}>
                        {formatDue(todo.due_date)}
                      </span>
                    )}

                    {/* Expand toggle */}
                    <button
                      onClick={() => toggleExpand(todo.id)}
                      style={{
                        background: 'transparent',
                        border: 'none',
                        color: expanded ? 'var(--color-text-label)' : 'var(--color-text-dim)',
                        fontSize: '10px',
                        padding: '0 2px',
                        cursor: 'pointer',
                        flexShrink: 0,
                        lineHeight: 1,
                      }}
                    >
                      {expanded ? '▲' : '▼'}
                    </button>

                    <button
                      onClick={e => { e.stopPropagation(); del.mutate(todo.id) }}
                      style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '12px', padding: '0 4px', cursor: 'pointer' }}
                      onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
                      onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
                    >
                      ×
                    </button>
                  </div>

                  {/* Detail panel */}
                  {expanded && (
                    <DetailPanel
                      todo={todo}
                      listId={selectedListId}
                    />
                  )}
                </div>
              )
            })}
          </div>
        )
      })}

      {/* Add task form */}
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
