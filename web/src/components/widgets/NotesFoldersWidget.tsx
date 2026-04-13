import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { NoteFolder } from '../../lib/types'
import { useNotesStore } from '../../stores/notesStore'

function useNoteFolders() {
  return useQuery({
    queryKey: ['note-folders'],
    queryFn: () => apiFetch<NoteFolder[]>('/api/note-folders'),
  })
}

function useCreateNoteFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<NoteFolder>('/api/note-folders', {
        method: 'POST',
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['note-folders'] }),
  })
}

function useDeleteNoteFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/api/note-folders/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['note-folders'] }),
  })
}

export default function NotesFoldersWidget() {
  const { data, isLoading, error } = useNoteFolders()
  const create = useCreateNoteFolder()
  const del = useDeleteNoteFolder()
  const { selectedFolderId, setFolder } = useNotesStore()
  const [draft, setDraft] = useState('')

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

  const folders = data ?? []

  function handleAdd() {
    const name = draft.trim()
    if (!name) return
    create.mutate(name, { onSuccess: () => setDraft('') })
  }

  return (
    <div className="flex flex-col">
      {folders.length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '80px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
        </div>
      )}
      {folders.map(folder => {
        const active = folder.id === selectedFolderId
        return (
          <div
            key={folder.id}
            onClick={() => setFolder(active ? null : folder.id)}
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
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)' }}>{active ? '▶' : '▷'}</span>
            <span style={{ flex: 1 }}>{folder.name}</span>
            <button
              onClick={e => { e.stopPropagation(); del.mutate(folder.id) }}
              disabled={del.isPending}
              style={{ background: 'transparent', border: 'none', color: 'var(--color-text-dim)', fontSize: '12px', padding: '0 2px', cursor: 'pointer' }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
            >
              ×
            </button>
          </div>
        )
      })}

      <div className="flex gap-2" style={{ padding: '8px 12px', borderTop: folders.length > 0 ? '1px solid var(--color-border)' : 'none', background: 'var(--color-surface)' }}>
        <input
          type="text"
          value={draft}
          onChange={e => setDraft(e.target.value)}
          placeholder="new folder..."
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
