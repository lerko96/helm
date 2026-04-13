import { create } from 'zustand'

interface NotesStore {
  selectedFolderId: number | null
  selectedNoteId: number | null
  setFolder: (id: number | null) => void
  setNote: (id: number | null) => void
}

export const useNotesStore = create<NotesStore>(set => ({
  selectedFolderId: null,
  selectedNoteId: null,
  setFolder: id => set({ selectedFolderId: id, selectedNoteId: null }),
  setNote: id => set({ selectedNoteId: id }),
}))
