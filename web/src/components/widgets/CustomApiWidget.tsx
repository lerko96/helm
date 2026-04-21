import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getToken } from '../../lib/auth'
import { parseWidgetConfig } from '../../lib/widget-config'
import { renderTemplate, escapeHTML } from '../../lib/template'
import type { WidgetProps } from '../layout/Shell'

// Config shape for the custom-api widget. URL + headers stay server-side and
// are stripped by sanitizeWidgetConfig on the backend — only rendering fields
// reach the browser.
interface CustomApiConfig extends Record<string, unknown> {
  template: string
  refresh: string
}

const SCHEMA = {
  template: { kind: 'string' as const, default: '' },
  refresh: { kind: 'string' as const, default: '' },
}

// parseDurationMs turns a Go-style duration ("30s", "5m", "1h") into ms.
// Used to drive React Query's refetchInterval so the frontend re-polls
// roughly in step with the backend cache expiry.
function parseDurationMs(s: string): number | false {
  const m = /^(\d+)(ms|s|m|h)$/.exec(s.trim())
  if (!m) return false
  const n = Number(m[1])
  switch (m[2]) {
    case 'ms': return n
    case 's': return n * 1000
    case 'm': return n * 60_000
    case 'h': return n * 3_600_000
  }
  return false
}

async function fetchProxy(widgetID: string): Promise<unknown> {
  const token = getToken()
  const res = await fetch(`/api/proxy?widget_id=${encodeURIComponent(widgetID)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    throw new Error(`upstream fetch failed (${res.status})`)
  }
  const ct = res.headers.get('Content-Type') ?? ''
  if (ct.includes('application/json')) {
    return res.json()
  }
  // Non-JSON passthrough: return as text so templates can still substitute.
  return res.text()
}

export default function CustomApiWidget({ id, config }: WidgetProps) {
  const cfg = parseWidgetConfig<CustomApiConfig>(config, SCHEMA)
  const refetchInterval = useMemo(() => parseDurationMs(cfg.refresh), [cfg.refresh])

  const { data, isLoading, error } = useQuery({
    queryKey: ['custom-api', id],
    queryFn: () => fetchProxy(id ?? ''),
    enabled: Boolean(id),
    refetchInterval: refetchInterval === false ? false : refetchInterval,
  })

  if (!id) {
    return <Message tone="alert">widget id missing</Message>
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        <div className="skeleton" style={{ height: '16px', width: '40%' }} />
        <div className="skeleton" style={{ height: '16px', width: '80%' }} />
        <div className="skeleton" style={{ height: '16px', width: '60%' }} />
      </div>
    )
  }

  if (error) {
    return <Message tone="alert">{(error as Error).message}</Message>
  }

  const body = cfg.template
    ? renderTemplate(cfg.template, data)
    : escapeHTML(typeof data === 'string' ? data : JSON.stringify(data, null, 2))

  return (
    <div
      style={{
        padding: '12px',
        fontSize: 'var(--text-xs)',
        color: 'var(--color-text-primary)',
        whiteSpace: 'pre-wrap',
        lineHeight: 1.5,
      }}
      // Template output is HTML-escaped by renderTemplate/escapeHTML before
      // reaching here — dangerouslySetInnerHTML is safe as long as the
      // escaping invariant in lib/template.ts holds.
      dangerouslySetInnerHTML={{ __html: body }}
    />
  )
}

function Message({ tone, children }: { tone: 'alert' | 'dim'; children: React.ReactNode }) {
  return (
    <div style={{ padding: '12px' }}>
      <span className={tone === 'alert' ? 'status status-alert' : ''} style={{ fontSize: 'var(--text-xs)', color: tone === 'dim' ? 'var(--color-text-dim)' : undefined }}>
        {children}
      </span>
    </div>
  )
}
