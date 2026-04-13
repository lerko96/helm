import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Tag } from '../../lib/types'

function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: () => apiFetch<Tag[]>('/api/tags'),
  })
}

function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, color }: { name: string; color: string }) =>
      apiFetch<{ id: number }>('/api/tags', {
        method: 'POST',
        body: JSON.stringify({ name, color }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}

function useDeleteTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/tags/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tags'] }),
  })
}

export default function TagsWidget() {
  const { data, isLoading, error } = useTags()
  const create = useCreateTag()
  const del = useDeleteTag()
  const [nameDraft, setNameDraft] = useState('')
  const [colorDraft, setColorDraft] = useState('#6b7280')

  function handleAdd() {
    const name = nameDraft.trim()
    if (!name) return
    create.mutate({ name, color: colorDraft }, {
      onSuccess: () => { setNameDraft('') },
    })
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} style={{ height: '28px', background: 'var(--color-surface-raised)' }} />
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

  const tags = data ?? []

  return (
    <div className="flex flex-col">
      {tags.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '60px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO TAGS</span>
        </div>
      )}
      {tags.map(tag => (
        <div
          key={tag.id}
          className="flex items-center gap-2"
          style={{ padding: '6px 12px', borderBottom: '1px solid var(--color-border)' }}
        >
          <span style={{ width: '10px', height: '10px', background: tag.color, flexShrink: 0, display: 'inline-block' }} />
          <span style={{ flex: 1, fontSize: 'var(--text-sm)', color: 'var(--color-text-primary)' }}>{tag.name}</span>
          <button
            onClick={() => del.mutate(tag.id)}
            disabled={del.isPending}
            style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '12px', padding: '0 4px', cursor: 'pointer' }}
            onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
            onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
          >
            ×
          </button>
        </div>
      ))}

      <div
        className="flex gap-2 items-center"
        style={{ padding: '8px 12px', borderTop: tags.length > 0 ? '1px solid var(--color-border)' : 'none', background: 'var(--color-surface)' }}
      >
        <input
          type="color"
          value={colorDraft}
          onChange={e => setColorDraft(e.target.value)}
          style={{ width: '28px', height: '28px', padding: '2px', border: '1px solid var(--color-border)', background: 'transparent', cursor: 'pointer', flexShrink: 0 }}
        />
        <input
          type="text"
          value={nameDraft}
          onChange={e => setNameDraft(e.target.value)}
          placeholder="tag name..."
          style={{ flex: 1, fontSize: 'var(--text-sm)' }}
          onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
        />
        <button
          className="btn-ghost"
          onClick={handleAdd}
          disabled={!nameDraft.trim() || create.isPending}
          style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
        >
          +
        </button>
      </div>
    </div>
  )
}
