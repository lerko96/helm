import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { ClipboardItem } from '../../lib/types'
import TagPicker from '../shared/TagPicker'

function useClipboard() {
  return useQuery({
    queryKey: ['clipboard'],
    queryFn: () => apiFetch<ClipboardItem[]>('/api/clipboard'),
  })
}

function useCreateClipboardItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ content, title, language }: { content: string; title?: string; language?: string }) =>
      apiFetch<ClipboardItem>('/api/clipboard', {
        method: 'POST',
        body: JSON.stringify({ content, title: title || undefined, language: language || undefined }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clipboard'] }),
  })
}

function useUpdateClipboardItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, title }: { id: number; title: string | null }) =>
      apiFetch<ClipboardItem>(`/api/clipboard/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ title }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clipboard'] }),
  })
}

function useDeleteClipboardItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/clipboard/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clipboard'] }),
  })
}

export default function ClipboardWidget() {
  const { data, isLoading, error } = useClipboard()
  const create = useCreateClipboardItem()
  const update = useUpdateClipboardItem()
  const del = useDeleteClipboardItem()
  const [draft, setDraft] = useState('')
  const [titleDraft, setTitleDraft] = useState('')
  const [languageDraft, setLanguageDraft] = useState('')
  const [showExtra, setShowExtra] = useState(false)
  const [copiedId, setCopiedId] = useState<number | null>(null)
  const [editTitleId, setEditTitleId] = useState<number | null>(null)
  const [editTitleDraft, setEditTitleDraft] = useState('')

  function handleCopy(item: ClipboardItem) {
    navigator.clipboard.writeText(item.content).then(() => {
      setCopiedId(item.id)
      setTimeout(() => setCopiedId(null), 1500)
    })
  }

  function handleAdd() {
    const content = draft.trim()
    if (!content) return
    create.mutate(
      { content, title: titleDraft.trim() || undefined, language: languageDraft.trim() || undefined },
      { onSuccess: () => { setDraft(''); setTitleDraft(''); setLanguageDraft('') } }
    )
  }

  function startEditTitle(item: ClipboardItem) {
    setEditTitleId(item.id)
    setEditTitleDraft(item.title ?? '')
  }

  function commitEditTitle(id: number) {
    update.mutate(
      { id, title: editTitleDraft.trim() || null },
      { onSuccess: () => setEditTitleId(null) }
    )
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
          style={{ padding: '6px 12px', borderBottom: '1px solid var(--color-border)' }}
        >
          <div className="flex items-center gap-2">
            {editTitleId === item.id ? (
              <input
                type="text"
                value={editTitleDraft}
                onChange={e => setEditTitleDraft(e.target.value)}
                autoFocus
                style={{ flex: 1, fontSize: 'var(--text-sm)' }}
                onKeyDown={e => {
                  if (e.key === 'Enter') commitEditTitle(item.id)
                  if (e.key === 'Escape') setEditTitleId(null)
                }}
                onBlur={() => commitEditTitle(item.id)}
              />
            ) : (
              <div
                style={{ flex: 1, fontSize: 'var(--text-sm)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--color-text-primary)', cursor: 'pointer' }}
                onClick={() => handleCopy(item)}
              >
                {item.title ?? item.content}
              </div>
            )}
            {item.language && editTitleId !== item.id && (
              <span className="status status-neutral" style={{ fontSize: '9px', flexShrink: 0 }}>{item.language}</span>
            )}
            {copiedId === item.id && (
              <span className="status status-active" style={{ flexShrink: 0, fontSize: '9px' }}>COPIED</span>
            )}
            <button
              onClick={e => { e.stopPropagation(); startEditTitle(item) }}
              style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '11px', padding: '0 2px', cursor: 'pointer', flexShrink: 0 }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-text-primary)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
            >
              ✎
            </button>
            <button
              onClick={() => del.mutate(item.id)}
              disabled={del.isPending}
              style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '12px', padding: '0 4px', cursor: 'pointer', flexShrink: 0 }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
            >
              ×
            </button>
          </div>
          <div style={{ marginTop: '4px' }}>
            <TagPicker entityType="clipboard" entityId={item.id} />
          </div>
        </div>
      ))}

      <div
        className="flex flex-col gap-2"
        style={{ padding: '8px 12px', borderTop: items.length > 0 ? '1px solid var(--color-border)' : 'none', background: 'var(--color-surface)' }}
      >
        <div className="flex gap-2">
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
            onClick={() => setShowExtra(v => !v)}
            style={{ fontSize: 'var(--text-xs)', padding: '6px 8px', color: 'var(--color-text-dim)' }}
          >
            ⋯
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
        {showExtra && (
          <div className="flex gap-2">
            <input
              type="text"
              value={titleDraft}
              onChange={e => setTitleDraft(e.target.value)}
              placeholder="title (optional)..."
              style={{ flex: 1, fontSize: 'var(--text-sm)' }}
            />
            <input
              type="text"
              value={languageDraft}
              onChange={e => setLanguageDraft(e.target.value)}
              placeholder="lang..."
              style={{ width: '80px', fontSize: 'var(--text-sm)' }}
            />
          </div>
        )}
      </div>
    </div>
  )
}
