import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { TodoList, Todo } from '../../lib/types'
import { useTasksStore } from '../../stores/tasksStore'
import ConfirmButton from '../shared/ConfirmButton'

function useTodoLists() {
  return useQuery({
    queryKey: ['todo-lists'],
    queryFn: () => apiFetch<TodoList[]>('/api/todo-lists'),
  })
}

function useTodos() {
  return useQuery({
    queryKey: ['todos'],
    queryFn: () => apiFetch<Todo[]>('/api/todos'),
  })
}

function useCreateTodoList() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, color }: { name: string; color: string }) =>
      apiFetch<TodoList>('/api/todo-lists', {
        method: 'POST',
        body: JSON.stringify({ name, color }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todo-lists'] }),
  })
}

function useDeleteTodoList() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/todo-lists/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['todo-lists'] }),
  })
}

export default function TaskListsWidget() {
  const { data: lists, isLoading, error } = useTodoLists()
  const { data: todos } = useTodos()
  const { selectedListId, setList } = useTasksStore()
  const create = useCreateTodoList()
  const del = useDeleteTodoList()
  const [nameDraft, setNameDraft] = useState('')
  const [colorDraft, setColorDraft] = useState('#666666')

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} className="skeleton" style={{ height: '32px' }} />
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

  const allLists = lists ?? []
  const allTodos = todos ?? []

  function countForList(listId: number) {
    return allTodos.filter(t => t.list_id === listId && t.status !== 'done').length
  }

  function handleAdd() {
    const name = nameDraft.trim()
    if (!name) return
    create.mutate({ name, color: colorDraft }, { onSuccess: () => setNameDraft('') })
  }

  return (
    <div className="flex flex-col">
      {/* All tasks option */}
      <div
        onClick={() => setList(null)}
        style={{
          padding: '8px 12px',
          borderBottom: '1px solid var(--color-border)',
          cursor: 'pointer',
          background: selectedListId === null ? 'var(--color-surface-raised)' : 'transparent',
          fontSize: 'var(--text-sm)',
          color: selectedListId === null ? 'var(--color-text-primary)' : 'var(--color-text-label)',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}
      >
        <span style={{ width: '8px', height: '8px', background: 'var(--color-text-label)', display: 'inline-block', flexShrink: 0 }} />
        All Tasks
        <span style={{ marginLeft: 'auto', fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)' }}>
          {allTodos.filter(t => t.status !== 'done').length}
        </span>
      </div>

      {allLists.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '60px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO LISTS</span>
        </div>
      )}

      {allLists.map(list => {
        const active = list.id === selectedListId
        return (
          <div
            key={list.id}
            onClick={() => setList(active ? null : list.id)}
            style={{
              padding: '8px 12px',
              borderBottom: '1px solid var(--color-border)',
              cursor: 'pointer',
              background: active ? 'var(--color-surface-raised)' : 'transparent',
              fontSize: 'var(--text-sm)',
              color: active ? 'var(--color-text-primary)' : 'var(--color-text-label)',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <span
              style={{ width: '8px', height: '8px', background: list.color ?? 'var(--color-text-label)', display: 'inline-block', flexShrink: 0 }}
            />
            <span style={{ flex: 1 }}>{list.name}</span>
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)' }}>
              {countForList(list.id)}
            </span>
            <ConfirmButton onConfirm={() => del.mutate(list.id)} disabled={del.isPending} style={{ padding: '0 2px' }} />
          </div>
        )
      })}

      {/* Create list form */}
      <div className="flex gap-2" style={{ padding: '8px 12px', borderTop: '1px solid var(--color-border)', background: 'var(--color-surface)' }}>
        <input
          type="color"
          value={colorDraft}
          onChange={e => setColorDraft(e.target.value)}
          style={{ width: '28px', height: '28px', padding: '2px', border: '1px solid var(--color-border)', background: 'var(--color-surface)', cursor: 'pointer', flexShrink: 0 }}
        />
        <input
          type="text"
          value={nameDraft}
          onChange={e => setNameDraft(e.target.value)}
          placeholder="new list..."
          style={{ flex: 1, fontSize: 'var(--text-sm)' }}
          onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
        />
        <button
          className="btn-ghost"
          onClick={handleAdd}
          disabled={!nameDraft.trim() || create.isPending}
          style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
        >
          {create.isPending ? '…' : '+'}
        </button>
      </div>
    </div>
  )
}
