import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Memo } from '../../lib/types'
import { useSearchStore } from '../../stores/searchStore'
import TagPicker from '../shared/TagPicker'
import MarkdownRenderer from '../shared/MarkdownRenderer'
import ConfirmButton from '../shared/ConfirmButton'

function useMemos(query: string) {
  return useQuery({
    queryKey: ['memos', query],
    queryFn: () => {
      const qs = query ? `?q=${encodeURIComponent(query)}` : ''
      return apiFetch<Memo[]>(`/api/memos${qs}`)
    },
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

function useUpdateMemo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, visibility, is_pinned }: { id: number; visibility: Memo['visibility']; is_pinned: boolean }) =>
      apiFetch<Memo>(`/api/memos/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ visibility, is_pinned }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['memos'] }),
  })
}

function useDeleteMemo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/memos/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['memos'] }),
  })
}

function formatTs(ts: string) {
  const d = new Date(ts)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) + ' ' +
    d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
}

export default function MemosWidget() {
  const { query } = useSearchStore()
  const { data, isLoading, error } = useMemos(query)
  const create = useCreateMemo()
  const update = useUpdateMemo()
  const del = useDeleteMemo()
  const [draft, setDraft] = useState('')
  const [copiedTokenId, setCopiedTokenId] = useState<number | null>(null)
  const [expandedMemoId, setExpandedMemoId] = useState<number | null>(null)

  function handleSubmit() {
    const content = draft.trim()
    if (!content) return
    create.mutate(content, { onSuccess: () => setDraft('') })
  }

  function toggleVisibility(memo: Memo) {
    update.mutate({ id: memo.id, visibility: memo.visibility === 'private' ? 'public' : 'private', is_pinned: memo.is_pinned })
  }

  function togglePin(memo: Memo) {
    update.mutate({ id: memo.id, visibility: memo.visibility, is_pinned: !memo.is_pinned })
  }

  function copyToken(token: string, id: number) {
    navigator.clipboard.writeText(token).then(() => {
      setCopiedTokenId(id)
      setTimeout(() => setCopiedTokenId(null), 1500)
    })
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} className="skeleton" style={{ height: '48px' }} />
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
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>
              {query ? `NO RESULTS FOR "${query}"` : 'NO DATA'}
            </span>
          </div>
        ) : (
          memos.map(memo => (
            <div
              key={memo.id}
              style={{ padding: '10px 12px', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: '8px', alignItems: 'flex-start' }}
            >
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{ display: 'flex', alignItems: 'flex-start', gap: '6px', marginBottom: '4px', cursor: 'pointer' }}
                  onClick={() => setExpandedMemoId(expandedMemoId === memo.id ? null : memo.id)}
                >
                  {memo.is_pinned && (
                    <span style={{ fontSize: '10px', color: 'var(--color-text-primary)', flexShrink: 0, marginTop: '3px' }}>◆</span>
                  )}
                  {expandedMemoId === memo.id ? (
                    <MarkdownRenderer content={memo.content} />
                  ) : (
                    <div
                      style={{
                        fontSize: 'var(--text-sm)',
                        color: 'var(--color-text-primary)',
                        display: '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                        lineHeight: '1.5',
                      }}
                    >
                      {memo.content}
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                  <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)' }}>
                    {formatTs(memo.created_at)}
                  </span>
                  {memo.visibility === 'public' && (
                    <span className="status status-active" style={{ fontSize: '9px' }}>PUBLIC</span>
                  )}
                  {memo.visibility === 'public' && memo.share_token && (
                    <span
                      onClick={() => copyToken(memo.share_token!, memo.id)}
                      title="click to copy share token"
                      style={{
                        fontSize: '10px',
                        fontFamily: 'monospace',
                        color: copiedTokenId === memo.id ? 'var(--color-accent-green)' : 'var(--color-text-dim)',
                        cursor: 'pointer',
                        letterSpacing: '0',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        maxWidth: '120px',
                        whiteSpace: 'nowrap',
                        display: 'inline-block',
                      }}
                    >
                      {copiedTokenId === memo.id ? 'copied!' : memo.share_token}
                    </span>
                  )}
                </div>
                <div style={{ marginTop: '4px' }}>
                  <TagPicker entityType="memo" entityId={memo.id} />
                </div>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', alignItems: 'center', flexShrink: 0 }}>
                <button
                  onClick={() => togglePin(memo)}
                  title="pin"
                  style={{ background: 'transparent', border: 'none', fontSize: '10px', cursor: 'pointer', padding: '2px 4px', color: memo.is_pinned ? 'var(--color-text-primary)' : 'var(--color-text-dim)' }}
                >
                  ◆
                </button>
                <button
                  onClick={() => toggleVisibility(memo)}
                  title="toggle visibility"
                  style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '10px', padding: '2px 4px', cursor: 'pointer' }}
                  onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-text-primary)')}
                  onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
                >
                  {memo.visibility === 'public' ? '◉' : '○'}
                </button>
                <ConfirmButton onConfirm={() => del.mutate(memo.id)} disabled={del.isPending} style={{ padding: '2px 4px', fontSize: '10px' }} />
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
