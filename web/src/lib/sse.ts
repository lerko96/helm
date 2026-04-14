import { getToken } from './auth'

export interface ReminderEvent {
  id: number
  entity_type: string
  entity_id: number
  remind_at: string
}

export interface CaldavSyncedEvent {
  source_id: number
  error: boolean
}

export interface SSEHandlers {
  onReminder: (r: ReminderEvent) => void
  onCaldavSynced?: (e: CaldavSyncedEvent) => void
}

export function startSSE(handlers: SSEHandlers): () => void {
  const token = getToken()
  if (!token) return () => {}

  const url = `/api/events?token=${encodeURIComponent(token)}`
  let es: EventSource | null = null
  let stopped = false
  let retryTimer: ReturnType<typeof setTimeout> | null = null

  function connect() {
    if (stopped) return
    es = new EventSource(url)

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data) as Record<string, unknown>
        if (data.type === 'caldav_synced') {
          handlers.onCaldavSynced?.({ source_id: data.source_id as number, error: data.error as boolean })
        } else if (data.id) {
          handlers.onReminder(data as unknown as ReminderEvent)
        }
      } catch {
        // ignore non-JSON pings
      }
    }

    es.onerror = () => {
      es?.close()
      es = null
      if (!stopped) {
        retryTimer = setTimeout(connect, 5000)
      }
    }
  }

  connect()

  return () => {
    stopped = true
    if (retryTimer) clearTimeout(retryTimer)
    es?.close()
  }
}
