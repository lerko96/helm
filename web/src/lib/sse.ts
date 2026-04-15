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

export interface MutationEvent {
  entity_type: string
  action: 'create' | 'update' | 'delete'
}

export interface SSEHandlers {
  onReminder: (r: ReminderEvent) => void
  onCaldavSynced?: (e: CaldavSyncedEvent) => void
  onMutation?: (e: MutationEvent) => void
}

const MIN_BACKOFF_MS = 1000
const MAX_BACKOFF_MS = 60_000

export function startSSE(handlers: SSEHandlers): () => void {
  const token = getToken()
  if (!token) return () => {}

  const url = `/api/events?token=${encodeURIComponent(token)}`
  let es: EventSource | null = null
  let stopped = false
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let backoff = MIN_BACKOFF_MS

  function connect() {
    if (stopped) return
    es = new EventSource(url)

    es.onopen = () => {
      backoff = MIN_BACKOFF_MS
    }

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data) as Record<string, unknown>
        if (data.type === 'caldav_synced') {
          handlers.onCaldavSynced?.({ source_id: data.source_id as number, error: data.error as boolean })
        } else if (data.type === 'mutation') {
          handlers.onMutation?.({ entity_type: data.entity_type as string, action: data.action as MutationEvent['action'] })
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
        retryTimer = setTimeout(() => {
          backoff = Math.min(backoff * 2, MAX_BACKOFF_MS)
          connect()
        }, backoff)
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
