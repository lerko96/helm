import { create } from 'zustand'

export const THEMES = ['noir', 'light', 'gruvbox'] as const
export type Theme = (typeof THEMES)[number]

export const DEFAULT_THEME: Theme = 'noir'
const STORAGE_KEY = 'helm_theme'

function isTheme(v: unknown): v is Theme {
  return typeof v === 'string' && (THEMES as readonly string[]).includes(v)
}

export function readStoredTheme(): Theme {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (isTheme(raw)) return raw
  } catch {
    // localStorage may throw in private mode / blocked contexts
  }
  return DEFAULT_THEME
}

export function applyTheme(theme: Theme) {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.theme = theme
}

interface ThemeStore {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useThemeStore = create<ThemeStore>(set => {
  const initial = readStoredTheme()
  applyTheme(initial)
  return {
    theme: initial,
    setTheme: (theme: Theme) => {
      try {
        localStorage.setItem(STORAGE_KEY, theme)
      } catch {
        // ignore
      }
      applyTheme(theme)
      set({ theme })
    },
  }
})
