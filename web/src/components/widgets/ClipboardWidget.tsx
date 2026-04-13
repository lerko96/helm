import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { ClipboardItem } from '../../lib/types'

function useClipboard() {
  return useQuery({
    queryKey: ['clipboard'],
    queryFn: () => apiFetch<ClipboardItem[]>('/api/clipboard'),
  })
}

function useCreateClipboardItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (content: string) =>
      apiFetch<ClipboardItem>('/api/clipboard', {
        method: 'POST',
        body: JSON.stringify({ content }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clipboard'] }),
  })
}

export default function ClipboardWidget() {
  const { data, isLoading, error } = useClipboard()
  const create = useCreateClipboardItem()
  const [draft, setDraft] = useState('')
  const [copiedId, setCopiedId] = useState<number | null>(null)

  function handleCopy(item: ClipboardItem) {
    navigator.clipboard.writeText(item.content).then(() => {
      setCopiedId(item.id)
      setTimeout(() => setCopiedId(null), 1500)
    })
  }

  function handleAdd() {
    const content = draft.trim()
    if (!content) return
    create.mutate(content, { onSuccess: () => setDraft('') })
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} style={{ height: '36px', background: 'var(--color-surface-raised)' }} />
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

  const items = data ?? []

  return (
    <div className="flex flex-col">
      {items.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '80px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
        </div>
      )}
      {items.map(item => (
        <div
          key={item.id}
          className="flex items-center gap-2"
          style={{
            padding: '8px 12px',
            borderBottom: '1px solid var(--color-border)',
            cursor: 'pointer',
          }}
          onClick={() => handleCopy(item)}
        >
          <div
            style={{
              flex: 1,
              fontSize: 'var(--text-sm)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              color: 'var(--color-text-primary)',
            }}
          >
            {item.title ?? item.content}
          </div>
          {copiedId === item.id && (
            <span className="status status-active" style={{ flexShrink: 0, fontSize: '10px' }}>COPIED</span>
          )}
        </div>
      ))}

      <div className="flex gap-2" style={{ padding: '8px 12px', borderTop: items.length > 0 ? '1px solid var(--color-border)' : 'none', background: 'var(--color-surface)' }}>
        <input
          type="text"
          value={draft}
          onChange={e => setDraft(e.target.value)}
          placeholder="clip content..."
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
