import { describe, it, expect } from 'vitest'
import { renderTemplate, lookup, escapeHTML } from './template'

describe('renderTemplate', () => {
  it('substitutes top-level keys', () => {
    expect(renderTemplate('hi {{.name}}', { name: 'world' })).toBe('hi world')
  })

  it('substitutes nested paths', () => {
    const data = { user: { profile: { name: 'Ada' } } }
    expect(renderTemplate('{{.user.profile.name}}', data)).toBe('Ada')
  })

  it('substitutes array indices', () => {
    const data = { items: [{ title: 'first' }, { title: 'second' }] }
    expect(renderTemplate('{{.items[0].title}} / {{.items[1].title}}', data)).toBe('first / second')
  })

  it('renders missing keys as empty string', () => {
    expect(renderTemplate('{{.missing}}', {})).toBe('')
    expect(renderTemplate('a={{.a}} b={{.b.c}}', { a: 1 })).toBe('a=1 b=')
  })

  it('HTML-escapes substituted values', () => {
    const data = { x: '<script>alert(1)</script>' }
    expect(renderTemplate('{{.x}}', data)).toBe('&lt;script&gt;alert(1)&lt;/script&gt;')
  })

  it('HTML-escapes quotes and ampersands', () => {
    expect(renderTemplate('{{.x}}', { x: `"a"&'b'` })).toBe('&quot;a&quot;&amp;&#39;b&#39;')
  })

  it('stringifies non-string scalars', () => {
    expect(renderTemplate('{{.n}}/{{.b}}', { n: 42, b: true })).toBe('42/true')
  })

  it('stringifies objects as JSON (still escaped)', () => {
    const data = { obj: { '<k>': 'v' } }
    expect(renderTemplate('{{.obj}}', data)).toBe('{&quot;&lt;k&gt;&quot;:&quot;v&quot;}')
  })

  it('ignores tokens without leading dot', () => {
    // Template engine only recognizes {{.foo}} — bare {{foo}} must pass through.
    expect(renderTemplate('{{foo}}', { foo: 'bar' })).toBe('{{foo}}')
  })

  it('tolerates whitespace inside placeholders', () => {
    expect(renderTemplate('{{ .name }}', { name: 'Ada' })).toBe('Ada')
  })
})

describe('lookup', () => {
  it('returns undefined on missing path', () => {
    expect(lookup({ a: 1 }, 'b')).toBeUndefined()
    expect(lookup({ a: { b: 1 } }, 'a.c')).toBeUndefined()
  })

  it('returns undefined for out-of-bounds array index', () => {
    expect(lookup({ xs: [1] }, 'xs[5]')).toBeUndefined()
  })

  it('returns null values as-is', () => {
    expect(lookup({ a: null }, 'a')).toBeNull()
  })
})

describe('escapeHTML', () => {
  it('escapes all dangerous characters', () => {
    expect(escapeHTML(`<>&"'`)).toBe('&lt;&gt;&amp;&quot;&#39;')
  })

  it('leaves safe text alone', () => {
    expect(escapeHTML('hello world 123')).toBe('hello world 123')
  })
})
