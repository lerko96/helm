import { parseWidgetConfig } from '../../lib/widget-config'
import type { WidgetProps } from '../layout/Shell'

// Config shape for the iframe widget. URL is public (it has to be — the
// browser makes the request), but the server still validates it against
// iframe_allowed_hosts at config load + sets a matching CSP frame-src so
// a widget pointing at an un-allowlisted host won't render.
interface IframeConfig extends Record<string, unknown> {
  url: string
  height: string
  sandbox: string
}

const SCHEMA = {
  url: { kind: 'string' as const, default: '' },
  height: { kind: 'string' as const, default: '480px' },
  // Mirrors config.DefaultIframeSandbox. Narrow by default; operators widen
  // per-widget in config.yml only when the embedded app needs it.
  sandbox: { kind: 'string' as const, default: 'allow-same-origin allow-scripts' },
}

export default function IframeWidget({ config }: WidgetProps) {
  const cfg = parseWidgetConfig<IframeConfig>(config, SCHEMA)

  if (!cfg.url) {
    return (
      <div style={{ padding: '12px' }}>
        <span className="status status-alert" style={{ fontSize: 'var(--text-xs)' }}>
          iframe: url missing in config
        </span>
      </div>
    )
  }

  return (
    <iframe
      src={cfg.url}
      sandbox={cfg.sandbox}
      referrerPolicy="no-referrer"
      loading="lazy"
      style={{
        width: '100%',
        height: cfg.height,
        border: 'none',
        background: 'var(--color-bg)',
        display: 'block',
      }}
    />
  )
}
