import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../../lib/api'
import { parseWidgetConfig } from '../../lib/widget-config'
import type { WidgetProps } from '../layout/Shell'

interface DockerConfig extends Record<string, unknown> {
  refresh: string
  name_filter: string
}

interface Container {
  id: string
  name: string
  image: string
  state: string
  status: string
  created: number
  ports?: string[]
}

const SCHEMA = {
  refresh: { kind: 'string' as const, default: '30s' },
  name_filter: { kind: 'string' as const, default: '' },
}

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

// stateDot returns the color-token for the status dot. Kept tiny; the engine
// uses a handful of canonical state strings and anything else reads as dim.
function stateColor(state: string): string {
  switch (state) {
    case 'running':
      return 'var(--color-accent, #7cb37c)'
    case 'paused':
    case 'restarting':
      return 'var(--color-warn, #c7a96b)'
    case 'exited':
    case 'dead':
      return 'var(--color-alert, #c97a7a)'
    default:
      return 'var(--color-text-dim)'
  }
}

export default function DockerStatusWidget({ config }: WidgetProps) {
  const cfg = parseWidgetConfig<DockerConfig>(config, SCHEMA)
  const refetchInterval = useMemo(() => parseDurationMs(cfg.refresh), [cfg.refresh])

  const { data, isLoading, error } = useQuery({
    queryKey: ['docker-containers'],
    queryFn: () => apiFetch<Container[]>('/api/docker/containers'),
    refetchInterval: refetchInterval === false ? 30_000 : refetchInterval,
  })

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2" style={{ padding: '12px' }}>
        {[0, 1, 2].map(i => <div key={i} className="skeleton" style={{ height: '20px' }} />)}
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ padding: '12px' }}>
        <span className="status status-alert" style={{ fontSize: 'var(--text-xs)' }}>
          {(error as Error).message}
        </span>
      </div>
    )
  }

  const containers = (data ?? []).filter(c =>
    cfg.name_filter ? c.name.includes(cfg.name_filter) : true,
  )

  if (containers.length === 0) {
    return (
      <div className="flex items-center justify-center" style={{ height: '80px' }}>
        <span style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-dim)', letterSpacing: '0.1em' }}>
          NO CONTAINERS
        </span>
      </div>
    )
  }

  return (
    <div className="flex flex-col" style={{ overflowY: 'auto' }}>
      {containers.map(c => (
        <div
          key={c.id}
          style={{
            padding: '10px 12px',
            borderBottom: '1px solid var(--color-border)',
            display: 'flex',
            gap: '10px',
            alignItems: 'flex-start',
          }}
        >
          <span
            aria-label={c.state}
            style={{
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: stateColor(c.state),
              flexShrink: 0,
              marginTop: '6px',
            }}
          />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 'var(--text-xs)', color: 'var(--color-text-primary)', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {c.name}
            </div>
            <div style={{ fontSize: 'var(--text-xxs, 10px)', color: 'var(--color-text-dim)', marginTop: '2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {c.image}
            </div>
            <div style={{ fontSize: 'var(--text-xxs, 10px)', color: 'var(--color-text-dim)', letterSpacing: '0.05em', marginTop: '2px' }}>
              {c.status}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
