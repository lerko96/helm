import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Bookmark } from '../../lib/types'

function useBookmarks() {
  return useQuery({
    queryKey: ['bookmarks'],
    queryFn: () => apiFetch<Bookmark[]>('/api/bookmarks'),
  })
}

function useCreateBookmark() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ url, title }: { url: string; title: string }) =>
      apiFetch<Bookmark>('/api/bookmarks', {
        method: 'POST',
        body: JSON.stringify({ url, title }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bookmarks'] }),
  })
}

function getDomain(url: string) {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

export default function BookmarksWidget() {
  const { data, isLoading, error } = useBookmarks()
  const create = useCreateBookmark()
  const [urlDraft, setUrlDraft] = useState('')
  const [titleDraft, setTitleDraft] = useState('')

  function handleAdd() {
    const url = urlDraft.trim()
    if (!url) return
    create.mutate({ url, title: titleDraft.trim() }, {
      onSuccess: () => { setUrlDraft(''); setTitleDraft('') },
    })
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

  const bookmarks = data ?? []

  return (
    <div className="flex flex-col">
      {bookmarks.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '80px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
        </div>
      )}
      {bookmarks.map(bm => (
        <div
          key={bm.id}
          className="flex items-center gap-3"
          style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)' }}
        >
          <span style={{ color: 'var(--color-text-label)', flexShrink: 0, fontSize: 'var(--text-sm)' }}>◈</span>
          <div className="flex flex-col" style={{ flex: 1, minWidth: 0 }}>
            <a
              href={bm.url}
              target="_blank"
              rel="noopener noreferrer"
              style={{
                fontSize: 'var(--text-sm)',
                color: 'var(--color-text-primary)',
                textDecoration: 'none',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                display: 'block',
              }}
              onMouseEnter={e => (e.currentTarget.style.textDecoration = 'underline')}
              onMouseLeave={e => (e.currentTarget.style.textDecoration = 'none')}
            >
              {bm.title ?? getDomain(bm.url)}
            </a>
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)' }}>
              {getDomain(bm.url)}
            </span>
          </div>
        </div>
      ))}

      <div
        className="flex flex-col gap-2"
        style={{ padding: '8px 12px', borderTop: bookmarks.length > 0 ? '1px solid var(--color-border)' : 'none', background: 'var(--color-surface)' }}
      >
        <input
          type="text"
          value={urlDraft}
          onChange={e => setUrlDraft(e.target.value)}
          placeholder="url..."
          style={{ width: '100%', fontSize: 'var(--text-sm)' }}
          onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
        />
        <div className="flex gap-2">
          <input
            type="text"
            value={titleDraft}
            onChange={e => setTitleDraft(e.target.value)}
            placeholder="title (optional)..."
            style={{ flex: 1, fontSize: 'var(--text-sm)' }}
            onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
          />
          <button
            className="btn-ghost"
            onClick={handleAdd}
            disabled={!urlDraft.trim() || create.isPending}
            style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
          >
            +
          </button>
        </div>
      </div>
    </div>
  )
}
