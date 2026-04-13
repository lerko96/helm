import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { CalendarSource } from '../../lib/types'

function useSources() {
  return useQuery({
    queryKey: ['calendar-sources'],
    queryFn: () => apiFetch<CalendarSource[]>('/api/calendar/sources'),
  })
}

function formatSynced(dt: string | null) {
  if (!dt) return 'NEVER'
  return new Date(dt).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).toUpperCase()
}

export default function CalendarSourcesWidget() {
  const qc = useQueryClient()
  const { data, isLoading, error } = useSources()

  const [queuedIds, setQueuedIds] = useState<Set<number>>(new Set())
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [color, setColor] = useState('#3b82f6')
  const [isLocal, setIsLocal] = useState(false)
  const [formError, setFormError] = useState('')

  const syncSource = useMutation({
    mutationFn: (id: number) =>
      apiFetch(`/api/calendar/sources/${id}/sync`, { method: 'POST' }),
    onMutate: (id) => setQueuedIds(prev => new Set(prev).add(id)),
    onError: (_e, id) => setQueuedIds(prev => { const s = new Set(prev); s.delete(id); return s }),
  })

  const deleteSource = useMutation({
    mutationFn: (id: number) =>
      apiFetch(`/api/calendar/sources/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['calendar-sources'] }),
  })

  const createSource = useMutation({
    mutationFn: (body: object) =>
      apiFetch('/api/calendar/sources', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-sources'] })
      setName(''); setUrl(''); setUsername(''); setPassword('')
      setColor('#3b82f6'); setIsLocal(false); setFormError('')
      setShowForm(false)
    },
    onError: (e: Error) => setFormError(e.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) { setFormError('name required'); return }
    if (!isLocal && !url.trim()) { setFormError('url required for remote sources'); return }
    createSource.mutate({
      name: name.trim(),
      url: isLocal ? null : url.trim(),
      username: username.trim() || null,
      password: password || null,
      color,
      is_local: isLocal,
    })
  }

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div
        className="flex items-center justify-between"
        style={{ padding: '6px 12px', borderBottom: '1px solid var(--color-border)' }}
      >
        <span
          style={{
            fontSize: 'var(--text-xs)',
            letterSpacing: 'var(--letter-spacing-label)',
            color: 'var(--color-text-label)',
          }}
        >
          CALENDAR SOURCES
        </span>
        <button
          onClick={() => setShowForm(v => !v)}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            fontSize: 'var(--text-xs)',
            color: showForm ? 'var(--color-text-primary)' : 'var(--color-text-label)',
            letterSpacing: 'var(--letter-spacing-label)',
            padding: '2px 4px',
          }}
        >
          {showForm ? '×' : '+'}
        </button>
      </div>

      {/* Create form */}
      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-2"
          style={{
            padding: '10px 12px',
            borderBottom: '1px solid var(--color-border)',
            background: 'var(--color-surface)',
          }}
        >
          <input
            type="text"
            placeholder="NAME"
            value={name}
            onChange={e => setName(e.target.value)}
            autoFocus
            style={{
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border)',
              color: 'var(--color-text-primary)',
              fontFamily: 'var(--font-mono)',
              fontSize: 'var(--text-sm)',
              padding: '4px 8px',
              width: '100%',
            }}
          />
          <label
            style={{
              fontSize: 'var(--text-xs)',
              color: 'var(--color-text-label)',
              letterSpacing: 'var(--letter-spacing-label)',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <input
              type="checkbox"
              checked={isLocal}
              onChange={e => setIsLocal(e.target.checked)}
              style={{ accentColor: 'var(--color-accent-red)' }}
            />
            LOCAL (no URL)
          </label>
          {!isLocal && (
            <>
              <input
                type="text"
                placeholder="CALDAV URL"
                value={url}
                onChange={e => setUrl(e.target.value)}
                style={{
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)',
                  padding: '4px 8px',
                  width: '100%',
                }}
              />
              <input
                type="text"
                placeholder="USERNAME (optional)"
                value={username}
                onChange={e => setUsername(e.target.value)}
                style={{
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)',
                  padding: '4px 8px',
                  width: '100%',
                }}
              />
              <input
                type="password"
                placeholder="PASSWORD (optional)"
                value={password}
                onChange={e => setPassword(e.target.value)}
                style={{
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)',
                  padding: '4px 8px',
                  width: '100%',
                }}
              />
            </>
          )}
          <div className="flex items-center gap-2">
            <span
              style={{
                fontSize: 'var(--text-xs)',
                color: 'var(--color-text-label)',
                letterSpacing: 'var(--letter-spacing-label)',
              }}
            >
              COLOR
            </span>
            <input
              type="color"
              value={color}
              onChange={e => setColor(e.target.value)}
              style={{ width: '32px', height: '20px', border: 'none', cursor: 'pointer', background: 'none' }}
            />
          </div>
          {formError && <span className="status status-alert">{formError}</span>}
          <button
            type="submit"
            className="btn-solid"
            disabled={createSource.isPending}
            style={{ alignSelf: 'flex-start' }}
          >
            {createSource.isPending ? 'ADDING...' : 'ADD SOURCE'}
          </button>
        </form>
      )}

      {/* Source list */}
      {isLoading && (
        <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
          {[0, 1].map(i => (
            <div key={i} style={{ height: '40px', background: 'var(--color-surface-raised)' }} />
          ))}
        </div>
      )}

      {error && (
        <div style={{ padding: '12px' }}>
          <span className="status status-alert">{(error as Error).message}</span>
        </div>
      )}

      {!isLoading && !error && (data ?? []).length === 0 && (
        <div className="flex items-center justify-center" style={{ height: '60px' }}>
          <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>
            NO SOURCES
          </span>
        </div>
      )}

      {(data ?? []).map(src => (
        <div
          key={src.id}
          className="flex items-center gap-2"
          style={{
            padding: '8px 12px',
            borderBottom: '1px solid var(--color-border)',
          }}
        >
          {/* Color dot */}
          <span
            style={{
              width: '8px',
              height: '8px',
              background: src.color,
              flexShrink: 0,
              display: 'inline-block',
            }}
          />

          {/* Name + meta */}
          <div className="flex flex-col" style={{ flex: 1, minWidth: 0 }}>
            <span
              style={{
                fontSize: 'var(--text-sm)',
                color: 'var(--color-text-primary)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {src.name}
            </span>
            <span
              style={{
                fontSize: 'var(--text-xs)',
                color: 'var(--color-text-dim)',
                letterSpacing: 'var(--letter-spacing-label)',
              }}
            >
              {src.is_local ? 'LOCAL' : src.url ?? ''} · SYNCED {formatSynced(src.last_synced_at)}
            </span>
          </div>

          {/* Queued pill or sync button */}
          {src.is_local ? null : queuedIds.has(src.id) ? (
            <span className="status status-neutral">QUEUED</span>
          ) : (
            <button
              onClick={() => syncSource.mutate(src.id)}
              disabled={syncSource.isPending}
              style={{
                background: 'none',
                border: '1px solid var(--color-border)',
                cursor: 'pointer',
                fontSize: 'var(--text-xs)',
                color: 'var(--color-text-label)',
                letterSpacing: 'var(--letter-spacing-label)',
                padding: '2px 6px',
                fontFamily: 'var(--font-mono)',
              }}
            >
              SYNC
            </button>
          )}

          {/* Delete */}
          <button
            onClick={() => deleteSource.mutate(src.id)}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: 'var(--text-sm)',
              color: 'var(--color-text-dim)',
              padding: '0 2px',
              fontFamily: 'var(--font-mono)',
              flexShrink: 0,
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
