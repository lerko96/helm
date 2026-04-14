import { useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getToken } from '../../lib/auth'
import type { Attachment } from '../../lib/types'

interface Props {
  entityType: string
  entityId: number
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

export default function AttachmentList({ entityType, entityId }: Props) {
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['attachments', entityType, entityId],
    queryFn: async () => {
      const token = getToken()
      const res = await fetch(
        `/api/attachments?entity_type=${encodeURIComponent(entityType)}&entity_id=${entityId}`,
        { headers: token ? { Authorization: `Bearer ${token}` } : {} },
      )
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return res.json() as Promise<Attachment[]>
    },
  })

  const upload = useMutation({
    mutationFn: async (file: File) => {
      const token = getToken()
      const form = new FormData()
      form.append('file', file)
      form.append('entity_type', entityType)
      form.append('entity_id', String(entityId))
      const res = await fetch('/api/attachments', {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: form,
      })
      if (!res.ok) {
        const d = await res.json()
        throw new Error(d.error ?? `HTTP ${res.status}`)
      }
      return res.json() as Promise<Attachment>
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['attachments', entityType, entityId] }),
  })

  const remove = useMutation({
    mutationFn: async (id: number) => {
      const token = getToken()
      const res = await fetch(`/api/attachments/${id}`, {
        method: 'DELETE',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (res.status !== 204 && !res.ok) throw new Error(`HTTP ${res.status}`)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['attachments', entityType, entityId] }),
  })

  async function handleDownload(a: Attachment) {
    const token = getToken()
    const res = await fetch(`/api/attachments/${a.id}/download`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) return
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = a.original_name
    anchor.click()
    URL.revokeObjectURL(url)
  }

  const attachments = data ?? []

  return (
    <div style={{ borderTop: '1px solid var(--color-border)', padding: '8px 12px' }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: attachments.length > 0 ? '6px' : '0',
        }}
      >
        <span
          style={{
            fontSize: '10px',
            letterSpacing: '0.1em',
            color: 'var(--color-text-label)',
          }}
        >
          ATTACHMENTS
        </span>
        <button
          className="btn-ghost"
          onClick={() => fileRef.current?.click()}
          disabled={upload.isPending}
          style={{ fontSize: '10px', padding: '2px 8px', letterSpacing: '0.1em' }}
        >
          {upload.isPending ? 'UPLOADING...' : 'ATTACH FILE'}
        </button>
        <input
          ref={fileRef}
          type="file"
          style={{ display: 'none' }}
          onChange={e => {
            const file = e.target.files?.[0]
            if (file) {
              upload.mutate(file)
              e.target.value = ''
            }
          }}
        />
      </div>

      {upload.isError && (
        <span className="status status-alert" style={{ fontSize: '10px', display: 'block', marginBottom: '4px' }}>
          {(upload.error as Error).message}
        </span>
      )}

      {isLoading && (
        <div style={{ height: '20px', background: 'var(--color-surface-raised)' }} />
      )}

      {attachments.map(a => (
        <div
          key={a.id}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '3px 0',
            fontSize: 'var(--text-xs)',
          }}
        >
          <button
            onClick={() => handleDownload(a)}
            style={{
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--color-text-primary)',
              textDecoration: 'underline',
              fontSize: 'var(--text-xs)',
              padding: 0,
              flex: 1,
              textAlign: 'left',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {a.original_name}
          </button>
          <span style={{ color: 'var(--color-text-dim)', flexShrink: 0 }}>
            {formatSize(a.size)}
          </span>
          <button
            onClick={() => remove.mutate(a.id)}
            disabled={remove.isPending}
            style={{
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--color-text-dim)',
              fontSize: '11px',
              padding: '0 2px',
              flexShrink: 0,
              lineHeight: 1,
            }}
            onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-accent-red)')}
            onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-dim)')}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  )
}
