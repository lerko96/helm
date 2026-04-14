import { getToken } from './auth'

export interface ReminderEvent {
  id: number
  entity_type: string
  entity_id: number
  remind_at: string
}

export function startSSE(onReminder: (r: ReminderEvent) => void): () => void {
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
        const data = JSON.parse(e.data) as ReminderEvent
        if (data.id) onReminder(data)
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
