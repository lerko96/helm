import { useState, useEffect, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { CalendarEvent, CalendarSource } from '../../lib/types'

function useCalendarSources() {
  return useQuery({
    queryKey: ['calendar-sources'],
    queryFn: () => apiFetch<CalendarSource[]>('/api/calendar/sources'),
  })
}

function useCalendarEvents() {
  const { from, to } = useMemo(() => {
    const f = new Date()
    f.setHours(0, 0, 0, 0)
    const t = new Date(f)
    t.setDate(t.getDate() + 30)
    return { from: f.toISOString(), to: t.toISOString() }
  }, [])

  return useQuery({
    queryKey: ['calendar-events', from, to],
    queryFn: () =>
      apiFetch<CalendarEvent[]>(
        `/api/calendar/events?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
      ),
  })
}

function formatDayHeader(dateStr: string) {
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' }).toUpperCase()
}

function formatTime(dt: string) {
  return new Date(dt).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
}

function groupByDay(events: CalendarEvent[]): Map<string, CalendarEvent[]> {
  const map = new Map<string, CalendarEvent[]>()
  for (const ev of events) {
    const day = ev.start_at.slice(0, 10)
    if (!map.has(day)) map.set(day, [])
    map.get(day)!.push(ev)
  }
  return map
}

const today = new Date().toISOString().slice(0, 10)

export default function CalendarWidget() {
  const qc = useQueryClient()
  const { data: sources } = useCalendarSources()
  const { data, isLoading, error } = useCalendarEvents()

  const localSourceId = useMemo(() => sources?.find(s => s.is_local)?.id ?? null, [sources])
  const [showForm, setShowForm] = useState(false)
  const [title, setTitle] = useState('')
  const [date, setDate] = useState(today)
  const [startTime, setStartTime] = useState('09:00')
  const [endTime, setEndTime] = useState('10:00')
  const [isAllDay, setIsAllDay] = useState(false)
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState('')

  const createLocalSource = useMutation({
    mutationFn: () =>
      apiFetch<{ id: number }>('/api/calendar/sources', {
        method: 'POST',
        body: JSON.stringify({ name: 'Local', is_local: true, color: '#3b82f6' }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-sources'] })
    },
  })

  useEffect(() => {
    if (sources === undefined || sources.some(s => s.is_local) || createLocalSource.isPending) return
    createLocalSource.mutate()
  }, [sources, createLocalSource])

  const createEvent = useMutation({
    mutationFn: (body: object) =>
      apiFetch('/api/calendar/events', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar-events'] })
      setTitle('')
      setDate(today)
      setStartTime('09:00')
      setEndTime('10:00')
      setIsAllDay(false)
      setDescription('')
      setFormError('')
      setShowForm(false)
    },
    onError: (e: Error) => setFormError(e.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) { setFormError('title required'); return }
    if (!localSourceId) { setFormError('local calendar not ready'); return }
    const start_at = isAllDay
      ? new Date(date + 'T00:00:00').toISOString()
      : new Date(date + 'T' + startTime + ':00').toISOString()
    const end_at = isAllDay
      ? new Date(date + 'T23:59:59').toISOString()
      : new Date(date + 'T' + endTime + ':00').toISOString()
    createEvent.mutate({
      source_id: localSourceId,
      title: title.trim(),
      start_at,
      end_at,
      is_all_day: isAllDay,
      description: description.trim() || null,
    })
  }

  return (
    <div className="flex flex-col">
      {/* Header row */}
      <div
        className="flex items-center justify-between"
        style={{
          padding: '6px 12px',
          borderBottom: '1px solid var(--color-border)',
        }}
      >
        <span
          style={{
            fontSize: 'var(--text-xs)',
            letterSpacing: 'var(--letter-spacing-label)',
            color: 'var(--color-text-label)',
          }}
        >
          NEXT 30 DAYS
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
            placeholder="TITLE"
            value={title}
            onChange={e => setTitle(e.target.value)}
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
          <div className="flex items-center gap-2">
            <label
              style={{
                fontSize: 'var(--text-xs)',
                color: 'var(--color-text-label)',
                letterSpacing: 'var(--letter-spacing-label)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              <input
                type="checkbox"
                checked={isAllDay}
                onChange={e => setIsAllDay(e.target.checked)}
                style={{ accentColor: 'var(--color-accent-red)' }}
              />
              ALL DAY
            </label>
          </div>
          <input
            type="date"
            value={date}
            onChange={e => setDate(e.target.value)}
            style={{
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border)',
              color: 'var(--color-text-primary)',
              fontFamily: 'var(--font-mono)',
              fontSize: 'var(--text-xs)',
              padding: '4px 8px',
              width: '100%',
              colorScheme: 'dark',
            }}
          />
          {!isAllDay && (
            <div className="flex gap-2">
              <input
                type="time"
                value={startTime}
                onChange={e => setStartTime(e.target.value)}
                style={{
                  flex: 1,
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)',
                  padding: '4px 8px',
                  colorScheme: 'dark',
                }}
              />
              <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--text-xs)', alignSelf: 'center' }}>–</span>
              <input
                type="time"
                value={endTime}
                onChange={e => setEndTime(e.target.value)}
                style={{
                  flex: 1,
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-xs)',
                  padding: '4px 8px',
                  colorScheme: 'dark',
                }}
              />
            </div>
          )}
          <textarea
            placeholder="DESCRIPTION (optional)"
            value={description}
            onChange={e => setDescription(e.target.value)}
            rows={2}
            style={{
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border)',
              color: 'var(--color-text-primary)',
              fontFamily: 'var(--font-mono)',
              fontSize: 'var(--text-xs)',
              padding: '4px 8px',
              resize: 'vertical',
              width: '100%',
            }}
          />
          {formError && (
            <span className="status status-alert">{formError}</span>
          )}
          <button
            type="submit"
            className="btn-solid"
            disabled={createEvent.isPending}
            style={{ alignSelf: 'flex-start' }}
          >
            {createEvent.isPending ? 'ADDING...' : 'ADD EVENT'}
          </button>
        </form>
      )}

      {/* Event list */}
      {isLoading && (
        <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
          {[0, 1, 2].map(i => (
            <div key={i} className="skeleton" style={{ height: '48px' }} />
          ))}
        </div>
      )}

      {error && (
        <div style={{ padding: '12px' }}>
          <span className="status status-alert">{(error as Error).message}</span>
        </div>
      )}

      {!isLoading && !error && (() => {
        const events = data ?? []
        if (events.length === 0) {
          return (
            <div className="flex items-center justify-center" style={{ height: '80px' }}>
              <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
            </div>
          )
        }
        const grouped = groupByDay(events)
        return (
          <>
            {Array.from(grouped.entries()).map(([day, dayEvents]) => (
              <div key={day}>
                <div
                  style={{
                    padding: '6px 12px',
                    background: 'var(--color-surface)',
                    borderBottom: '1px solid var(--color-border)',
                    fontSize: 'var(--text-xs)',
                    letterSpacing: 'var(--letter-spacing-label)',
                    color: 'var(--color-text-label)',
                  }}
                >
                  {formatDayHeader(day + 'T00:00:00')}
                </div>
                {dayEvents.map(ev => (
                  <div
                    key={ev.id}
                    className="flex items-center gap-3"
                    style={{
                      padding: '8px 12px',
                      borderBottom: '1px solid var(--color-border)',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 'var(--text-xs)',
                        color: 'var(--color-text-label)',
                        letterSpacing: 'var(--letter-spacing-label)',
                        flexShrink: 0,
                        fontVariantNumeric: 'tabular-nums',
                        minWidth: '100px',
                      }}
                    >
                      {ev.is_all_day ? 'ALL DAY' : `${formatTime(ev.start_at)} – ${formatTime(ev.end_at)}`}
                    </span>
                    <span style={{ fontSize: 'var(--text-sm)', color: 'var(--color-text-primary)', flex: 1 }}>
                      {ev.title}
                    </span>
                  </div>
                ))}
              </div>
            ))}
          </>
        )
      })()}
    </div>
  )
}
