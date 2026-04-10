import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Memo } from '../../lib/types'

function useMemos() {
  return useQuery({
    queryKey: ['memos'],
    queryFn: () => apiFetch<Memo[]>('/api/memos'),
  })
}

function useCreateMemo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (content: string) =>
      apiFetch<Memo>('/api/memos', {
        method: 'POST',
        body: JSON.stringify({ content, visibility: 'private' }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['memos'] }),
  })
}

function formatTs(ts: string) {
  const d = new Date(ts)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) + ' ' +
    d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
}

export default function MemosWidget() {
  const { data, isLoading, error } = useMemos()
  const create = useCreateMemo()
  const [draft, setDraft] = useState('')

  function handleSubmit() {
    const content = draft.trim()
    if (!content) return
    create.mutate(content, { onSuccess: () => setDraft('') })
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} style={{ height: '48px', background: 'var(--color-surface-raised)' }} />
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

  const memos = data ?? []

  return (
    <div className="flex flex-col" style={{ height: '100%' }}>
      <div className="flex flex-col" style={{ overflowY: 'auto', flex: 1 }}>
        {memos.length === 0 ? (
          <div className="flex items-center justify-center" style={{ height: '80px' }}>
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
          </div>
        ) : (
          memos.map(memo => (
            <div
              key={memo.id}
              style={{
                padding: '10px 12px',
                borderBottom: '1px solid var(--color-border)',
              }}
            >
              <div
                style={{
                  fontSize: 'var(--text-sm)',
                  color: 'var(--color-text-primary)',
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical',
                  overflow: 'hidden',
                  lineHeight: '1.5',
                  marginBottom: '4px',
                }}
              >
                {memo.content}
              </div>
              <div style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)' }}>
                {formatTs(memo.created_at)}
              </div>
            </div>
          ))
        )}
      </div>

      <div style={{ borderTop: '1px solid var(--color-border)', padding: '8px 12px', display: 'flex', flexDirection: 'column', gap: '6px', background: 'var(--color-surface)' }}>
        <textarea
          value={draft}
          onChange={e => setDraft(e.target.value)}
          placeholder="new memo..."
          rows={2}
          style={{ width: '100%', resize: 'none', fontSize: 'var(--text-sm)' }}
          onKeyDown={e => {
            if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSubmit()
          }}
        />
        <button
          className="btn-ghost"
          onClick={handleSubmit}
          disabled={!draft.trim() || create.isPending}
          style={{ alignSelf: 'flex-end', fontSize: 'var(--text-xs)' }}
        >
          {create.isPending ? 'saving...' : 'add'}
        </button>
      </div>
    </div>
  )
}
