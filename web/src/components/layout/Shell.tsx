import { useState, useEffect } from 'react'

export interface Page {
  id: string
  label: string
  slug: string
  columns: Column[]
}

export interface Column {
  id: string
  size: 'small' | 'full' | 'large'
  widgets: Widget[]
}

export interface Widget {
  id: string
  type: string
  title: string
  content?: React.ReactNode
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type WidgetComponentMap = Record<string, React.ComponentType<any>>

interface ShellProps {
  pages: Page[]
  header?: React.ReactNode
  widgetComponents?: WidgetComponentMap
  onLogout?: () => void
}

const COLUMN_WIDTHS: Record<string, string> = {
  small: 'w-64 shrink-0',
  medium: 'w-80 shrink-0',
  large: 'w-96 shrink-0',
  full: 'flex-1 min-w-0',
}

export default function Shell({ pages, header, widgetComponents = {}, onLogout }: ShellProps) {
  const [activePage, setActivePage] = useState(pages[0]?.id ?? '')

  const currentPage = pages.find(p => p.id === activePage) ?? pages[0]

  return (
    <div className="min-h-screen flex flex-col" style={{ background: 'var(--color-bg)', color: 'var(--color-text-primary)', fontFamily: 'var(--font-mono)' }}>
      {/* Top bar */}
      <header style={{ borderBottom: '1px solid var(--color-border)', background: 'var(--color-surface)' }}>
        <div className="flex items-center justify-between px-4" style={{ height: '44px' }}>
          {/* Wordmark */}
          <div className="flex items-center gap-4">
            <span style={{ fontSize: 'var(--text-sm)', letterSpacing: '0.2em', textTransform: 'uppercase', color: 'var(--color-text-primary)' }}>
              HELM
            </span>
            <span style={{ color: 'var(--color-border-bright)', fontSize: 'var(--text-xs)' }}>◆</span>
          </div>

          {/* Page tabs */}
          <nav className="flex items-stretch h-full">
            {pages.map(page => (
              <button
                key={page.id}
                className="nav-tab"
                data-active={page.id === activePage}
                onClick={() => setActivePage(page.id)}
              >
                {page.label}
              </button>
            ))}
          </nav>

          {/* Right slot */}
          <div className="flex items-center gap-3">
            {header}
            <Clock />
            {onLogout && (
              <button
                onClick={onLogout}
                style={{
                  background: 'transparent',
                  border: 'none',
                  color: 'var(--color-text-dim)',
                  fontSize: 'var(--text-xs)',
                  letterSpacing: 'var(--letter-spacing-label)',
                  textTransform: 'uppercase',
                  cursor: 'pointer',
                  padding: '4px 0',
                }}
              >
                logout
              </button>
            )}
          </div>
        </div>
      </header>

      {/* Page content */}
      <main className="flex-1 flex overflow-hidden">
        {currentPage?.columns.map((col, i) => (
          <div
            key={col.id}
            className={`flex flex-col gap-0 overflow-y-auto ${COLUMN_WIDTHS[col.size]}`}
            style={{
              borderRight: i < (currentPage.columns.length - 1) ? '1px solid var(--color-border)' : 'none',
            }}
          >
            {col.widgets.map(widget => (
              <WidgetWrapper key={widget.id} widget={widget} widgetComponents={widgetComponents} />
            ))}
          </div>
        ))}
      </main>
    </div>
  )
}

function WidgetWrapper({ widget, widgetComponents }: { widget: Widget; widgetComponents: WidgetComponentMap }) {
  const Component = widgetComponents[widget.type]
  return (
    <div style={{ borderBottom: '1px solid var(--color-border)' }}>
      <div
        className="flex items-center justify-between"
        style={{
          padding: '8px 12px',
          borderBottom: '1px solid var(--color-border)',
          background: 'var(--color-surface)',
        }}
      >
        <span style={{ fontSize: 'var(--text-xs)', letterSpacing: 'var(--letter-spacing-label)', textTransform: 'uppercase', color: 'var(--color-text-label)' }}>
          {widget.title}
        </span>
        <span style={{ color: 'var(--color-border-bright)', fontSize: '10px' }}>—</span>
      </div>
      <div style={{ background: 'var(--color-bg)', minHeight: '80px' }}>
        {Component ? <Component /> : widget.content ?? <EmptyWidget />}
      </div>
    </div>
  )
}

function EmptyWidget() {
  return (
    <div className="flex items-center justify-center" style={{ height: '80px' }}>
      <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>NO DATA</span>
    </div>
  )
}

function Clock() {
  const [time, setTime] = useState(() => formatTime(new Date()))

  useEffect(() => {
    const id = setInterval(() => setTime(formatTime(new Date())), 1000)
    return () => clearInterval(id)
  }, [])

  return (
    <span style={{ fontSize: 'var(--text-xs)', letterSpacing: '0.1em', color: 'var(--color-text-label)', fontVariantNumeric: 'tabular-nums' }}>
      {time}
    </span>
  )
}

function formatTime(d: Date) {
  return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
}
