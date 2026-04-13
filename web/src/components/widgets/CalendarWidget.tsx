import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import type { CalendarEvent } from '../../lib/types'

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

export default function CalendarWidget() {
  const { data, isLoading, error } = useCalendarEvents()

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => (
          <div key={i} style={{ height: '48px', background: 'var(--color-surface-raised)' }} />
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
    <div className="flex flex-col">
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
    </div>
  )
}
