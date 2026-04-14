import { useState, useEffect, useCallback, useRef } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Shell, { type Page } from './components/layout/Shell'
import LoginPage from './components/LoginPage'
import { isAuthenticated, clearToken } from './lib/auth'
import { apiFetch } from './lib/api'
import { startSSE, type ReminderEvent } from './lib/sse'
import MemosWidget from './components/widgets/MemosWidget'
import TodosWidget from './components/widgets/TodosWidget'
import ClipboardWidget from './components/widgets/ClipboardWidget'
import BookmarksWidget from './components/widgets/BookmarksWidget'
import NotesFoldersWidget from './components/widgets/NotesFoldersWidget'
import NotesEditorWidget from './components/widgets/NotesEditorWidget'
import TaskListsWidget from './components/widgets/TaskListsWidget'
import TaskBoardWidget from './components/widgets/TaskBoardWidget'
import CalendarWidget from './components/widgets/CalendarWidget'
import CalendarSourcesWidget from './components/widgets/CalendarSourcesWidget'
import TagsWidget from './components/widgets/TagsWidget'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,
      retry: 1,
    },
  },
})

const WIDGET_COMPONENTS = {
  memos: MemosWidget,
  todos: TodosWidget,
  clipboard: ClipboardWidget,
  bookmarks: BookmarksWidget,
  'notes-folders': NotesFoldersWidget,
  'notes-editor': NotesEditorWidget,
  'task-lists': TaskListsWidget,
  'task-board': TaskBoardWidget,
  'cal-view': CalendarWidget,
  'cal-sources': CalendarSourcesWidget,
  tags: TagsWidget,
}

export default function App() {
  const [authed, setAuthed] = useState(isAuthenticated)
  const [pages, setPages] = useState<Page[] | null>(null)
  const [banner, setBanner] = useState<string | null>(null)
  const bannerTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const notifPermission = useRef<NotificationPermission>('default')

  const showBanner = useCallback((msg: string) => {
    setBanner(msg)
    if (bannerTimer.current) clearTimeout(bannerTimer.current)
    bannerTimer.current = setTimeout(() => setBanner(null), 6000)
  }, [])

  const onReminder = useCallback((r: ReminderEvent) => {
    const title = `REMINDER — ${r.entity_type.toUpperCase()} #${r.entity_id}`
    const body = `Due: ${new Date(r.remind_at).toLocaleString()}`

    if (notifPermission.current === 'granted') {
      new Notification(title, { body })
    } else if (notifPermission.current === 'default') {
      Notification.requestPermission().then(perm => {
        notifPermission.current = perm
        if (perm === 'granted') {
          new Notification(title, { body })
        } else {
          showBanner(`${title} — ${body}`)
        }
      })
    } else {
      showBanner(`${title} — ${body}`)
    }
  }, [showBanner])

  useEffect(() => {
    if (!authed) return
    apiFetch<Page[]>('/api/config/pages')
      .then(setPages)
      .catch(() => setPages([]))
  }, [authed])

  useEffect(() => {
    if (!authed) return
    const stop = startSSE(onReminder)
    return stop
  }, [authed, onReminder])

  if (!authed) {
    return <LoginPage onSuccess={() => setAuthed(true)} />
  }

  if (pages === null) {
    return (
      <div
        className="min-h-screen flex items-center justify-center"
        style={{ background: 'var(--color-bg)', fontFamily: 'var(--font-mono)' }}
      >
        <span style={{ fontSize: 'var(--text-xs)', letterSpacing: '0.2em', color: 'var(--color-text-dim)' }}>
          LOADING...
        </span>
      </div>
    )
  }

  function handleLogout() {
    clearToken()
    setAuthed(false)
    setPages(null)
    setBanner(null)
  }

  const bannerEl = banner ? (
    <span
      className="status status-alert"
      style={{ fontSize: 'var(--text-xs)', letterSpacing: '0.05em', cursor: 'pointer' }}
      onClick={() => setBanner(null)}
    >
      {banner}
    </span>
  ) : undefined

  return (
    <QueryClientProvider client={queryClient}>
      <Shell pages={pages} header={bannerEl} widgetComponents={WIDGET_COMPONENTS} onLogout={handleLogout} />
    </QueryClientProvider>
  )
}
