import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Bookmark, BookmarkCollection } from '../../lib/types'
import { useSearchStore } from '../../stores/searchStore'
import TagPicker from '../shared/TagPicker'
import ConfirmButton from '../shared/ConfirmButton'

function useBookmarks(collectionId: number | null, query: string) {
  return useQuery({
    queryKey: ['bookmarks', { collectionId, query }],
    queryFn: () => {
      const params = new URLSearchParams()
      if (collectionId != null) params.set('collection_id', String(collectionId))
      if (query) params.set('q', query)
      const qs = params.toString() ? `?${params}` : ''
      return apiFetch<Bookmark[]>(`/api/bookmarks${qs}`)
    },
  })
}

function useBookmarkCollections() {
  return useQuery({
    queryKey: ['bookmark-collections'],
    queryFn: () => apiFetch<BookmarkCollection[]>('/api/bookmark-collections'),
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

function useUpdateBookmark() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: {
      id: number
      url: string
      title: string | null
      description: string | null
      collection_id: number | null
      is_public: boolean
      is_pinned: boolean
    }) =>
      apiFetch<Bookmark>(`/api/bookmarks/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bookmarks'] }),
  })
}

function useDeleteBookmark() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/bookmarks/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bookmarks'] }),
  })
}

function useCreateBookmarkCollection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<BookmarkCollection>('/api/bookmark-collections', {
        method: 'POST',
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bookmark-collections'] }),
  })
}

function useDeleteBookmarkCollection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) =>
      apiFetch<void>(`/api/bookmark-collections/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bookmark-collections'] }),
  })
}

function getDomain(url: string) {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

type EditDraft = {
  title: string
  url: string
  description: string
  is_public: boolean
  is_pinned: boolean
  collection_id: number | null
}

export default function BookmarksWidget() {
  const [collectionFilter, setCollectionFilter] = useState<number | null>(null)
  const [showCollections, setShowCollections] = useState(false)
  const [collectionDraft, setCollectionDraft] = useState('')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editDraft, setEditDraft] = useState<EditDraft>({
    title: '', url: '', description: '', is_public: false, is_pinned: false, collection_id: null,
  })
  const [urlDraft, setUrlDraft] = useState('')
  const [titleDraft, setTitleDraft] = useState('')

  const { query } = useSearchStore()
  const { data, isLoading, error } = useBookmarks(collectionFilter, query)
  const { data: collections } = useBookmarkCollections()
  const create = useCreateBookmark()
  const update = useUpdateBookmark()
  const del = useDeleteBookmark()
  const createCollection = useCreateBookmarkCollection()
  const deleteCollection = useDeleteBookmarkCollection()

  const allCollections = collections ?? []
  const bookmarks = data ?? []

  function handleAdd() {
    const url = urlDraft.trim()
    if (!url) return
    create.mutate({ url, title: titleDraft.trim() }, {
      onSuccess: () => { setUrlDraft(''); setTitleDraft('') },
    })
  }

  function startEdit(bm: Bookmark) {
    setEditingId(bm.id)
    setEditDraft({
      title: bm.title ?? '',
      url: bm.url,
      description: bm.description ?? '',
      is_public: bm.is_public,
      is_pinned: bm.is_pinned,
      collection_id: bm.collection_id,
    })
  }

  function commitEdit(id: number) {
    update.mutate(
      { id, url: editDraft.url, title: editDraft.title || null, description: editDraft.description || null, is_public: editDraft.is_public, is_pinned: editDraft.is_pinned, collection_id: editDraft.collection_id },
      { onSuccess: () => setEditingId(null) }
    )
  }

  function togglePin(bm: Bookmark) {
    update.mutate({ id: bm.id, url: bm.url, title: bm.title, description: bm.description, is_public: bm.is_public, is_pinned: !bm.is_pinned, collection_id: bm.collection_id })
  }

  function handleCreateCollection() {
    const name = collectionDraft.trim()
    if (!name) return
    createCollection.mutate(name, { onSuccess: () => setCollectionDraft('') })
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} className="skeleton" style={{ height: '36px' }} />
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

  return (
    <div className="flex flex-col">
      {/* Collection filter bar */}
      <div style={{ padding: '6px 12px', borderBottom: '1px solid var(--color-border)', display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
        <button
          className="btn-ghost"
          onClick={() => setCollectionFilter(null)}
          style={{ fontSize: 'var(--text-xs)', padding: '3px 8px', background: collectionFilter === null ? 'var(--color-surface-raised)' : 'transparent' }}
        >
          ALL
        </button>
        {allCollections.map(c => (
          <button
            key={c.id}
            className="btn-ghost"
            onClick={() => setCollectionFilter(collectionFilter === c.id ? null : c.id)}
            style={{ fontSize: 'var(--text-xs)', padding: '3px 8px', background: collectionFilter === c.id ? 'var(--color-surface-raised)' : 'transparent' }}
          >
            {c.name}
          </button>
        ))}
        <button
          className="btn-ghost"
          onClick={() => setShowCollections(v => !v)}
          style={{ fontSize: 'var(--text-xs)', padding: '3px 8px', marginLeft: 'auto', color: 'var(--color-text-dim)' }}
        >
          {showCollections ? '− mgr' : '+ mgr'}
        </button>
      </div>

      {/* Collections manager panel */}
      {showCollections && (
        <div style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', background: 'var(--color-surface-raised)' }}>
          {allCollections.length === 0 && (
            <div style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', marginBottom: '6px' }}>no collections</div>
          )}
          {allCollections.map(c => (
            <div key={c.id} className="flex items-center gap-2" style={{ padding: '3px 0' }}>
              <span style={{ flex: 1, fontSize: 'var(--text-xs)', color: 'var(--color-text-label)' }}>{c.name}</span>
              <ConfirmButton onConfirm={() => deleteCollection.mutate(c.id)} disabled={deleteCollection.isPending} style={{ fontSize: '10px', padding: '0 4px' }} />
            </div>
          ))}
          <div className="flex gap-2" style={{ marginTop: '6px' }}>
            <input
              type="text"
              value={collectionDraft}
              onChange={e => setCollectionDraft(e.target.value)}
              placeholder="new collection..."
              style={{ flex: 1, fontSize: 'var(--text-xs)' }}
              onKeyDown={e => { if (e.key === 'Enter') handleCreateCollection() }}
            />
            <button
              className="btn-ghost"
              onClick={handleCreateCollection}
              disabled={!collectionDraft.trim() || createCollection.isPending}
              style={{ fontSize: 'var(--text-xs)', padding: '4px 8px' }}
            >
              +
            </button>
          </div>
        </div>
      )}

      {/* Bookmark list */}
      {bookmarks.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '80px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>
            {query ? `NO RESULTS FOR "${query}"` : 'NO DATA'}
          </span>
        </div>
      )}
      {bookmarks.map(bm => {
        if (editingId === bm.id) {
          return (
            <div key={bm.id} style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <input
                type="text"
                value={editDraft.url}
                onChange={e => setEditDraft(d => ({ ...d, url: e.target.value }))}
                placeholder="url..."
                style={{ fontSize: 'var(--text-sm)', width: '100%' }}
              />
              <input
                type="text"
                value={editDraft.title}
                onChange={e => setEditDraft(d => ({ ...d, title: e.target.value }))}
                placeholder="title..."
                style={{ fontSize: 'var(--text-sm)', width: '100%' }}
              />
              <input
                type="text"
                value={editDraft.description}
                onChange={e => setEditDraft(d => ({ ...d, description: e.target.value }))}
                placeholder="description..."
                style={{ fontSize: 'var(--text-sm)', width: '100%' }}
              />
              <div className="flex items-center gap-3" style={{ flexWrap: 'wrap' }}>
                <select
                  value={editDraft.collection_id ?? ''}
                  onChange={e => setEditDraft(d => ({ ...d, collection_id: e.target.value ? Number(e.target.value) : null }))}
                  style={{ fontSize: 'var(--text-xs)', background: 'var(--color-surface)', color: 'var(--color-text-primary)', border: '1px solid var(--color-border)', padding: '3px 6px' }}
                >
                  <option value="">no collection</option>
                  {allCollections.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                </select>
                <label style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', cursor: 'pointer' }}>
                  <input type="checkbox" checked={editDraft.is_public} onChange={e => setEditDraft(d => ({ ...d, is_public: e.target.checked }))} />
                  PUBLIC
                </label>
                <label style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', cursor: 'pointer' }}>
                  <input type="checkbox" checked={editDraft.is_pinned} onChange={e => setEditDraft(d => ({ ...d, is_pinned: e.target.checked }))} />
                  PIN
                </label>
                <div style={{ marginLeft: 'auto', display: 'flex', gap: '6px' }}>
                  <button
                    className="btn-ghost"
                    onClick={() => commitEdit(bm.id)}
                    disabled={!editDraft.url.trim() || update.isPending}
                    style={{ fontSize: 'var(--text-xs)', padding: '4px 8px' }}
                  >
                    {update.isPending ? 'saving…' : 'save'}
                  </button>
                  <button
                    className="btn-ghost"
                    onClick={() => setEditingId(null)}
                    style={{ fontSize: 'var(--text-xs)', padding: '4px 8px' }}
                  >
                    cancel
                  </button>
                </div>
              </div>
            </div>
          )
        }

        return (
          <div
            key={bm.id}
            className="flex items-center gap-3"
            style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)' }}
          >
            {bm.favicon_url ? (
              <img
                src={bm.favicon_url}
                alt=""
                width={14}
                height={14}
                style={{ flexShrink: 0, imageRendering: 'pixelated' }}
                onError={e => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
              />
            ) : (
              <span style={{ color: 'var(--color-text-label)', flexShrink: 0, fontSize: 'var(--text-sm)' }}>◈</span>
            )}
            <div className="flex flex-col" style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px', minWidth: 0 }}>
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
                  }}
                  onMouseEnter={e => (e.currentTarget.style.textDecoration = 'underline')}
                  onMouseLeave={e => (e.currentTarget.style.textDecoration = 'none')}
                >
                  {bm.title ?? getDomain(bm.url)}
                </a>
                {bm.is_public
                  ? <span className="status status-active" style={{ fontSize: '9px', flexShrink: 0 }}>PUBLIC</span>
                  : <span className="status status-neutral" style={{ fontSize: '9px', flexShrink: 0 }}>PRIVATE</span>
                }
              </div>
              <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-label)', letterSpacing: 'var(--letter-spacing-label)' }}>
                {getDomain(bm.url)}
              </span>
              <div style={{ marginTop: '3px' }}>
                <TagPicker entityType="bookmark" entityId={bm.id} />
              </div>
            </div>
            <button
              onClick={() => togglePin(bm)}
              title="pin"
              style={{ background: 'transparent', border: 'none', fontSize: '11px', cursor: 'pointer', padding: '0 2px', flexShrink: 0, color: bm.is_pinned ? 'var(--color-text-primary)' : 'var(--color-text-dim)' }}
            >
              ◆
            </button>
            <button
              onClick={() => startEdit(bm)}
              style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '11px', padding: '0 2px', cursor: 'pointer', flexShrink: 0 }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-text-primary)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
            >
              ✎
            </button>
            <ConfirmButton onConfirm={() => del.mutate(bm.id)} disabled={del.isPending} style={{ flexShrink: 0 }} />
          </div>
        )
      })}

      {/* Add form */}
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
            {create.isPending ? '…' : '+'}
          </button>
        </div>
      </div>
    </div>
  )
}
