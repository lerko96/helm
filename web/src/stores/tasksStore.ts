import { create } from 'zustand'

interface TasksStore {
  selectedListId: number | null
  setList: (id: number | null) => void
}

export const useTasksStore = create<TasksStore>(set => ({
  selectedListId: null,
  setList: id => set({ selectedListId: id }),
}))
