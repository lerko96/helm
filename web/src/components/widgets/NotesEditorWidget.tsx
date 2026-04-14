import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Note } from '../../lib/types'
import { useNotesStore } from '../../stores/notesStore'
import { useSearchStore } from '../../stores/searchStore'
import TagPicker from '../shared/TagPicker'
import MarkdownRenderer from '../shared/MarkdownRenderer'
import AttachmentList from '../shared/AttachmentList'

function useNotes(folderId: number | null, query: string) {
  return useQuery({
    queryKey: ['notes', folderId, query],
    queryFn: () => {
      const params = new URLSearchParams()
      if (folderId != null) params.set('folder_id', String(folderId))
      if (query) params.set('q', query)
      const qs = params.toString() ? `?${params}` : ''
      return apiFetch<Note[]>(`/api/notes${qs}`)
    },
  })
}

function useUpdateNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, content }: { id: number; content: string }) =>
      apiFetch<Note>(`/api/notes/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notes'] })
    },
  })
}

function useToggleNotePin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (note: Note) =>
      apiFetch<Note>(`/api/notes/${note.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          title: note.title,
          content: note.content,
          folder_id: note.folder_id,
          is_pinned: !note.is_pinned,
          due_date: note.due_date,
        }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })
}

function useCreateNote(folderId: number | null) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (title: string) =>
      apiFetch<Note>('/api/notes', {
        method: 'POST',
        body: JSON.stringify({ title, folder_id: folderId }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })
}

function NoteEditor({ note }: { note: Note }) {
  const [content, setContent] = useState(note.content ?? '')
  const [viewMode, setViewMode] = useState(false)
  const update = useUpdateNote()

  useEffect(() => {
    setContent(note.content ?? '')
    setViewMode(false)
  }, [note.id])

  useEffect(() => {
    if (!viewMode) setContent(note.content ?? '')
  }, [note.content])

  function handleBlur() {
    if (content !== (note.content ?? '')) {
      update.mutate({ id: note.id, content })
    }
  }

  return (
    <>
      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
          padding: '4px 12px',
          borderTop: '1px solid var(--color-border)',
          borderBottom: '1px solid var(--color-border)',
          background: 'var(--color-surface)',
        }}
      >
        <button
          className="btn-ghost"
          onClick={() => setViewMode(v => !v)}
          style={{ fontSize: '10px', padding: '2px 8px', letterSpacing: '0.1em' }}
        >
          {viewMode ? 'EDIT' : 'VIEW'}
        </button>
      </div>
      {viewMode ? (
        <div style={{ padding: '12px', minHeight: '300px', borderTop: '1px solid var(--color-border)' }}>
          <MarkdownRenderer content={content} />
        </div>
      ) : (
        <>
          <textarea
            value={content}
            onChange={e => setContent(e.target.value)}
            onBlur={handleBlur}
            style={{
              width: '100%',
              minHeight: '300px',
              resize: 'none',
              fontSize: 'var(--text-sm)',
              lineHeight: '1.6',
              padding: '12px',
              border: 'none',
              borderTop: '1px solid var(--color-border)',
              background: 'var(--color-bg)',
            }}
          />
          <AttachmentList entityType="note" entityId={note.id} />
        </>
      )}
    </>
  )
}

export default function NotesEditorWidget() {
  const { selectedFolderId, selectedNoteId, setNote } = useNotesStore()
  const { query } = useSearchStore()
  const { data, isLoading, error } = useNotes(selectedFolderId, query)
  const createNote = useCreateNote(selectedFolderId)
  const togglePin = useToggleNotePin()
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

  const notes = data ?? []
  const activeNote = notes.find(n => n.id === selectedNoteId) ?? null

  function handleAdd() {
    const title = draft.trim()
    if (!title) return
    createNote.mutate(title, {
      onSuccess: note => { setDraft(''); setNote(note.id) },
    })
  }

  return (
    <div className="flex flex-col">
      {/* Note list */}
      <div style={{ borderBottom: '1px solid var(--color-border)' }}>
        {notes.length === 0 && !activeNote && (
          <div className="flex items-center justify-center" style={{ height: '60px' }}>
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>
              {query ? `NO RESULTS FOR "${query}"` : 'NO NOTES'}
            </span>
          </div>
        )}
        {notes.map(note => (
          <div
            key={note.id}
            onClick={() => setNote(note.id === selectedNoteId ? null : note.id)}
            style={{
              padding: '8px 12px',
              borderBottom: '1px solid var(--color-border)',
              cursor: 'pointer',
              background: note.id === selectedNoteId ? 'var(--color-surface-raised)' : 'transparent',
              fontSize: 'var(--text-sm)',
              color: note.id === selectedNoteId ? 'var(--color-text-primary)' : 'var(--color-text-label)',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            {note.is_pinned && (
              <span style={{ fontSize: '10px', color: 'var(--color-text-primary)', flexShrink: 0 }}>◆</span>
            )}
            <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{note.title}</span>
            <button
              onClick={e => { e.stopPropagation(); togglePin.mutate(note) }}
              title="pin"
              style={{
                background: 'transparent',
                border: 'none',
                fontSize: '10px',
                cursor: 'pointer',
                padding: '0 2px',
                flexShrink: 0,
                color: note.is_pinned ? 'var(--color-text-primary)' : 'var(--color-text-dim)',
              }}
            >
              ◆
            </button>
          </div>
        ))}
        <div className="flex gap-2" style={{ padding: '8px 12px', background: 'var(--color-surface)' }}>
          <input
            type="text"
            value={draft}
            onChange={e => setDraft(e.target.value)}
            placeholder="new note..."
            style={{ flex: 1, fontSize: 'var(--text-sm)' }}
            onKeyDown={e => { if (e.key === 'Enter') handleAdd() }}
          />
          <button
            className="btn-ghost"
            onClick={handleAdd}
            disabled={!draft.trim() || createNote.isPending}
            style={{ fontSize: 'var(--text-xs)', padding: '6px 10px' }}
          >
            +
          </button>
        </div>
      </div>

      {/* Editor */}
      {activeNote && (
        <>
          <div style={{ padding: '6px 12px', borderBottom: '1px solid var(--color-border)' }}>
            <TagPicker entityType="note" entityId={activeNote.id} />
          </div>
          <NoteEditor note={activeNote} />
        </>
      )}

    </div>
  )
}
