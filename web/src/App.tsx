import { useState, useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Shell, { type Page } from './components/layout/Shell'
import LoginPage from './components/LoginPage'
import { isAuthenticated } from './lib/auth'
import MemosWidget from './components/widgets/MemosWidget'
import TodosWidget from './components/widgets/TodosWidget'
import ClipboardWidget from './components/widgets/ClipboardWidget'
import BookmarksWidget from './components/widgets/BookmarksWidget'
import NotesFoldersWidget from './components/widgets/NotesFoldersWidget'
import NotesEditorWidget from './components/widgets/NotesEditorWidget'
import TaskListsWidget from './components/widgets/TaskListsWidget'
import TaskBoardWidget from './components/widgets/TaskBoardWidget'
import CalendarWidget from './components/widgets/CalendarWidget'

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
}

export default function App() {
  const [authed, setAuthed] = useState(isAuthenticated)
  const [pages, setPages] = useState<Page[] | null>(null)

  useEffect(() => {
    if (!authed) return
    fetch('/api/config/pages')
      .then(r => r.json())
      .then(setPages)
      .catch(() => setPages([]))
  }, [authed])

  if (!authed) {
    return <LoginPage onSuccess={() => setAuthed(true)} />
  }

  if (pages === null) {
    return null
  }

  return (
    <QueryClientProvider client={queryClient}>
      <Shell pages={pages} widgetComponents={WIDGET_COMPONENTS} />
    </QueryClientProvider>
  )
}
