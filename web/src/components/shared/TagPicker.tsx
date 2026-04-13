import { useState, useRef, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { Tag } from '../../lib/types'

interface Props {
  entityType: string
  entityId: number
}

function useAllTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: () => apiFetch<Tag[]>('/api/tags'),
    staleTime: 1000 * 30,
  })
}

function useEntityTags(entityType: string, entityId: number) {
  return useQuery({
    queryKey: ['entity-tags', entityType, entityId],
    queryFn: () =>
      apiFetch<Tag[]>(`/api/tags?entity_type=${entityType}&entity_id=${entityId}`),
    staleTime: 1000 * 30,
  })
}

function useAttachTag(entityType: string, entityId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tagId: number) =>
      apiFetch<void>(`/api/tags/${tagId}/attach`, {
        method: 'POST',
        body: JSON.stringify({ entity_type: entityType, entity_id: entityId }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['entity-tags', entityType, entityId] })
    },
  })
}

function useDetachTag(entityType: string, entityId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tagId: number) =>
      apiFetch<void>(`/api/tags/${tagId}/detach`, {
        method: 'DELETE',
        body: JSON.stringify({ entity_type: entityType, entity_id: entityId }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['entity-tags', entityType, entityId] })
    },
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
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tags'] })
    },
  })
}

export default function TagPicker({ entityType, entityId }: Props) {
  const [open, setOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState('#6b7280')
  const dropdownRef = useRef<HTMLDivElement>(null)

  const { data: allTags = [] } = useAllTags()
  const { data: currentTags = [] } = useEntityTags(entityType, entityId)
  const attach = useAttachTag(entityType, entityId)
  const detach = useDetachTag(entityType, entityId)
  const createTag = useCreateTag()
  const qc = useQueryClient()

  const currentIds = new Set(currentTags.map(t => t.id))
  const unattached = allTags.filter(t => !currentIds.has(t.id))

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  function handleAttach(tag: Tag) {
    attach.mutate(tag.id, { onSuccess: () => setOpen(false) })
  }

  function handleDetach(e: React.MouseEvent, tagId: number) {
    e.stopPropagation()
    detach.mutate(tagId)
  }

  async function handleCreate() {
    const name = newName.trim()
    if (!name) return
    const result = await createTag.mutateAsync({ name, color: newColor })
    // invalidate all tags then attach the new one
    await qc.invalidateQueries({ queryKey: ['tags'] })
    attach.mutate(result.id, { onSuccess: () => { setNewName(''); setOpen(false) } })
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '4px', flexWrap: 'wrap', position: 'relative' }}>
      {currentTags.map(tag => (
        <span
          key={tag.id}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '3px',
            fontSize: '9px',
            letterSpacing: '0.05em',
            padding: '1px 4px',
            border: '1px solid var(--color-border)',
            color: 'var(--color-text-label)',
          }}
        >
          <span
            style={{
              width: '8px',
              height: '8px',
              background: tag.color,
              flexShrink: 0,
              display: 'inline-block',
            }}
          />
          {tag.name}
          <button
            onClick={e => handleDetach(e, tag.id)}
            style={{
              background: 'transparent',
              border: 'none',
              padding: '0 1px',
              cursor: 'pointer',
              fontSize: '10px',
              lineHeight: 1,
              color: 'var(--color-text-dim)',
            }}
            onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
            onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
          >
            ×
          </button>
        </span>
      ))}

      <div ref={dropdownRef} style={{ position: 'relative' }}>
        <button
          onClick={() => setOpen(v => !v)}
          style={{
            background: 'transparent',
            border: '1px solid var(--color-border)',
            padding: '1px 5px',
            cursor: 'pointer',
            fontSize: '10px',
            color: 'var(--color-text-dim)',
            lineHeight: 1.4,
          }}
          onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-text-primary)')}
          onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
        >
          +tag
        </button>

        {open && (
          <div
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              zIndex: 50,
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border-bright)',
              minWidth: '160px',
              maxHeight: '220px',
              overflowY: 'auto',
            }}
          >
            {unattached.length === 0 && (
              <div style={{ padding: '6px 8px', fontSize: '10px', color: 'var(--color-text-dim)' }}>
                no tags to attach
              </div>
            )}
            {unattached.map(tag => (
              <div
                key={tag.id}
                onClick={() => handleAttach(tag)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '5px 8px',
                  cursor: 'pointer',
                  fontSize: '11px',
                  color: 'var(--color-text-label)',
                }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--color-surface)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
              >
                <span style={{ width: '8px', height: '8px', background: tag.color, flexShrink: 0, display: 'inline-block' }} />
                {tag.name}
              </div>
            ))}

            {/* Create new tag */}
            <div
              style={{
                borderTop: '1px solid var(--color-border)',
                padding: '6px 8px',
                display: 'flex',
                flexDirection: 'column',
                gap: '4px',
              }}
            >
              <input
                type="text"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                placeholder="new tag..."
                style={{ fontSize: '10px', width: '100%' }}
                onKeyDown={e => { if (e.key === 'Enter') handleCreate() }}
              />
              <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                <input
                  type="color"
                  value={newColor}
                  onChange={e => setNewColor(e.target.value)}
                  style={{ width: '24px', height: '20px', padding: '1px', border: '1px solid var(--color-border)', background: 'transparent', cursor: 'pointer' }}
                />
                <button
                  className="btn-ghost"
                  onClick={handleCreate}
                  disabled={!newName.trim() || createTag.isPending}
                  style={{ flex: 1, fontSize: '9px', padding: '3px 6px' }}
                >
                  create + attach
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
