export interface Tag {
  id: number
  name: string
  color: string
  created_at: string
}

export interface Memo {
  id: number
  content: string
  visibility: 'private' | 'public'
  share_token: string | null
  is_pinned: boolean
  tags?: Tag[]
  created_at: string
  updated_at: string
}

export interface TodoList {
  id: number
  name: string
  color: string | null
}

export interface Todo {
  id: number
  list_id: number | null
  parent_id: number | null
  title: string
  description: string | null
  status: 'not_started' | 'in_progress' | 'done'
  priority: 'low' | 'medium' | 'high'
  is_pinned: boolean
  due_date: string | null
  tags?: Tag[]
  subtasks?: Todo[]
  created_at: string
  updated_at: string
}

export interface NoteFolder {
  id: number
  name: string
}

export interface Note {
  id: number
  folder_id: number | null
  title: string
  content: string | null
  is_pinned: boolean
  due_date: string | null
  tags?: Tag[]
  created_at: string
  updated_at: string
}

export interface ClipboardItem {
  id: number
  title: string | null
  content: string
  language: string | null
  is_pinned: boolean
  tags?: Tag[]
  created_at: string
  updated_at: string
}

export interface BookmarkCollection {
  id: number
  name: string
}

export interface Bookmark {
  id: number
  collection_id: number | null
  url: string
  title: string | null
  description: string | null
  favicon_url: string | null
  is_pinned: boolean
  is_public: boolean
  tags?: Tag[]
  created_at: string
}

export interface CalendarEvent {
  id: number
  source_id: string | null
  title: string
  description: string | null
  location: string | null
  start_at: string
  end_at: string
  is_all_day: boolean
  rrule: string | null
  created_at: string
}
