import { useState } from 'react'
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

const PAGES: Page[] = [
  {
    id: 'overview',
    label: 'Overview',
    slug: '/',
    columns: [
      {
        id: 'overview-left',
        size: 'small',
        widgets: [
          { id: 'calendar', type: 'cal-view', title: 'Calendar' },
          { id: 'todos', type: 'todos', title: 'Tasks' },
        ],
      },
      {
        id: 'overview-main',
        size: 'full',
        widgets: [
          { id: 'memos', type: 'memos', title: 'Memos' },
          { id: 'bookmarks', type: 'bookmarks', title: 'Bookmarks' },
        ],
      },
      {
        id: 'overview-right',
        size: 'small',
        widgets: [
          { id: 'clipboard', type: 'clipboard', title: 'Clipboard' },
        ],
      },
    ],
  },
  {
    id: 'notes',
    label: 'Notes',
    slug: '/notes',
    columns: [
      {
        id: 'notes-sidebar',
        size: 'small',
        widgets: [
          { id: 'notes-folders', type: 'notes-folders', title: 'Folders' },
        ],
      },
      {
        id: 'notes-main',
        size: 'full',
        widgets: [
          { id: 'notes-editor', type: 'notes-editor', title: 'Editor' },
        ],
      },
    ],
  },
  {
    id: 'tasks',
    label: 'Tasks',
    slug: '/tasks',
    columns: [
      {
        id: 'tasks-sidebar',
        size: 'small',
        widgets: [
          { id: 'task-lists', type: 'task-lists', title: 'Lists' },
        ],
      },
      {
        id: 'tasks-main',
        size: 'full',
        widgets: [
          { id: 'task-board', type: 'task-board', title: 'Active' },
        ],
      },
    ],
  },
  {
    id: 'calendar',
    label: 'Calendar',
    slug: '/calendar',
    columns: [
      {
        id: 'cal-main',
        size: 'full',
        widgets: [
          { id: 'cal-view', type: 'cal-view', title: 'Schedule' },
        ],
      },
    ],
  },
]

export default function App() {
  const [authed, setAuthed] = useState(isAuthenticated)

  if (!authed) {
    return <LoginPage onSuccess={() => setAuthed(true)} />
  }

  return (
    <QueryClientProvider client={queryClient}>
      <Shell pages={PAGES} widgetComponents={WIDGET_COMPONENTS} />
    </QueryClientProvider>
  )
}
