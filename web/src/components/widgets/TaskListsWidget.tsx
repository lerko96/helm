import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { TodoList, Todo } from '../../lib/types'
import { useTasksStore } from '../../stores/tasksStore'

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

export default function TaskListsWidget() {
  const { data: lists, isLoading, error } = useTodoLists()
  const { data: todos } = useTodos()
  const { selectedListId, setList } = useTasksStore()

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

  const allLists = lists ?? []
  const allTodos = todos ?? []

  function countForList(listId: number) {
    return allTodos.filter(t => t.list_id === listId && t.status !== 'done').length
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
              style={{
                width: '8px',
                height: '8px',
                background: list.color ?? 'var(--color-text-label)',
                display: 'inline-block',
                flexShrink: 0,
              }}
            />
            {list.name}
            <span style={{ marginLeft: 'auto', fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)' }}>
              {countForList(list.id)}
            </span>
          </div>
        )
      })}
    </div>
  )
}
