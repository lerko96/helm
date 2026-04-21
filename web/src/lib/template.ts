// Tiny dot-path template renderer for custom-api widgets.
//
// Why hand-rolled: the custom-api widget needs `{{.foo.bar}}` substitution
// against a JSON response. Pulling Handlebars or Mustache for one widget
// with one syntax is overkill. This is ~40 lines, has a narrow surface, and
// always HTML-escapes — no way to accidentally render user-controlled data
// as markup.

const PLACEHOLDER = /\{\{\s*\.([a-zA-Z0-9_.[\]]+)\s*\}\}/g

export function renderTemplate(template: string, data: unknown): string {
  return template.replace(PLACEHOLDER, (_match, path: string) => {
    const value = lookup(data, path)
    if (value === undefined || value === null) return ''
    return escapeHTML(stringify(value))
  })
}

// lookup walks a dot path like `foo.bar` or `items[0].name` on arbitrary JSON.
// Returns undefined on any missing segment — the template renderer substitutes
// an empty string so a missing key reads as "" rather than "undefined".
export function lookup(data: unknown, path: string): unknown {
  const segments = path.split(/\.|\[(\d+)\]/).filter(s => s !== undefined && s !== '')
  let cur: unknown = data
  for (const seg of segments) {
    if (cur === null || cur === undefined) return undefined
    if (/^\d+$/.test(seg)) {
      const idx = Number(seg)
      if (!Array.isArray(cur)) return undefined
      cur = cur[idx]
    } else if (typeof cur === 'object') {
      cur = (cur as Record<string, unknown>)[seg]
    } else {
      return undefined
    }
  }
  return cur
}

function stringify(v: unknown): string {
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return JSON.stringify(v)
}

const HTML_ESCAPES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
}

export function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, ch => HTML_ESCAPES[ch])
}
