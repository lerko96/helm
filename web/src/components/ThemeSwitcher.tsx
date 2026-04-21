import { useRef } from 'react'
import { THEMES, useThemeStore, type Theme } from '../stores/themeStore'

const LABELS: Record<Theme, { short: string; full: string }> = {
  noir: { short: 'N', full: 'Noir (dark)' },
  light: { short: 'L', full: 'Light' },
  gruvbox: { short: 'G', full: 'Gruvbox' },
}

/**
 * Compact segmented switch for the header. Arrow keys cycle; each option is a
 * separate <button> so screen readers announce the three choices distinctly.
 */
export default function ThemeSwitcher() {
  const theme = useThemeStore(s => s.theme)
  const setTheme = useThemeStore(s => s.setTheme)
  const refs = useRef<(HTMLButtonElement | null)[]>([])

  function handleKey(e: React.KeyboardEvent<HTMLButtonElement>, idx: number) {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
    e.preventDefault()
    const delta = e.key === 'ArrowRight' ? 1 : -1
    const next = (idx + delta + THEMES.length) % THEMES.length
    const nextTheme = THEMES[next]
    setTheme(nextTheme)
    refs.current[next]?.focus()
  }

  return (
    <div
      role="radiogroup"
      aria-label="Color theme"
      style={{
        display: 'inline-flex',
        border: '1px solid var(--color-border)',
      }}
    >
      {THEMES.map((t, i) => {
        const active = t === theme
        return (
          <button
            key={t}
            ref={el => {
              refs.current[i] = el
            }}
            role="radio"
            aria-checked={active}
            aria-label={LABELS[t].full}
            title={LABELS[t].full}
            onClick={() => setTheme(t)}
            onKeyDown={e => handleKey(e, i)}
            style={{
              background: active ? 'var(--color-text-primary)' : 'transparent',
              color: active ? 'var(--color-bg)' : 'var(--color-text-label)',
              border: 'none',
              borderLeft: i > 0 ? '1px solid var(--color-border)' : 'none',
              fontSize: 'var(--text-xs)',
              padding: '3px 7px',
              letterSpacing: '0.1em',
              cursor: 'pointer',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {LABELS[t].short}
          </button>
        )
      })}
    </div>
  )
}
