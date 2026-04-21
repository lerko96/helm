// Per-widget config types. Source of truth: fields declared in config.yml under
// each widget's `config:` block. Backend forwards the raw map in apiWidget.Config;
// the frontend narrows here, using parseWidgetConfig for lenient validation.
//
// Existing widgets have no config; they declare EmptyConfig so the dispatch type
// stays uniform and adding a new config field is a type-checked change.

export type WidgetConfigRaw = Record<string, unknown> | undefined

export type EmptyConfig = Record<string, never>

type FieldKind = 'string' | 'number' | 'boolean' | 'string[]' | 'record'

interface FieldSpec {
  kind: FieldKind
  default?: unknown
  optional?: boolean
}

type Schema<T> = { [K in keyof T]: FieldSpec }

export function parseWidgetConfig<T extends Record<string, unknown>>(
  raw: WidgetConfigRaw,
  schema: Schema<T>,
): T {
  const src = raw ?? {}
  const out = {} as T
  for (const key of Object.keys(schema) as (keyof T)[]) {
    const spec = schema[key]
    const value = (src as Record<string, unknown>)[key as string]
    const coerced = coerce(value, spec)
    if (coerced === undefined) {
      if (spec.optional) continue
      out[key] = spec.default as T[typeof key]
    } else {
      out[key] = coerced as T[typeof key]
    }
  }
  return out
}

function coerce(value: unknown, spec: FieldSpec): unknown {
  if (value === undefined || value === null) return undefined
  switch (spec.kind) {
    case 'string':
      return typeof value === 'string' ? value : undefined
    case 'number':
      return typeof value === 'number' && Number.isFinite(value) ? value : undefined
    case 'boolean':
      return typeof value === 'boolean' ? value : undefined
    case 'string[]':
      return Array.isArray(value) && value.every(v => typeof v === 'string') ? value : undefined
    case 'record':
      return typeof value === 'object' && !Array.isArray(value) ? value : undefined
  }
}
