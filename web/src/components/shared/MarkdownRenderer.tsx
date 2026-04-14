import { useMemo } from 'react'

// Inline: `code`, **bold**, _italic_, [text](url)
const INLINE_RE = /(`[^`]+`|\*\*[^*]+\*\*|_[^_]+_|\[[^\]]+\]\([^)]+\))/g

function parseInline(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = []
  let last = 0
  let key = 0
  for (const m of text.matchAll(INLINE_RE)) {
    if (m.index! > last) parts.push(text.slice(last, m.index))
    const raw = m[0]
    if (raw.startsWith('`')) {
      parts.push(
        <code key={key++} style={{ background: 'var(--color-surface-raised)', padding: '0 4px', fontSize: '0.9em' }}>
          {raw.slice(1, -1)}
        </code>
      )
    } else if (raw.startsWith('**')) {
      parts.push(<strong key={key++}>{raw.slice(2, -2)}</strong>)
    } else if (raw.startsWith('_')) {
      parts.push(<em key={key++}>{raw.slice(1, -1)}</em>)
    } else {
      const link = raw.match(/\[([^\]]+)\]\(([^)]+)\)/)!
      parts.push(
        <a
          key={key++}
          href={link[2]}
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: 'var(--color-text-primary)', textDecorationLine: 'underline' }}
          onMouseEnter={e => (e.currentTarget.style.color = 'var(--color-text-label)')}
          onMouseLeave={e => (e.currentTarget.style.color = 'var(--color-text-primary)')}
        >
          {link[1]}
        </a>
      )
    }
    last = m.index! + raw.length
  }
  if (last < text.length) parts.push(text.slice(last))
  return parts
}

function parseBlocks(content: string): React.ReactNode[] {
  const lines = content.split('\n')
  const blocks: React.ReactNode[] = []
  let i = 0
  let bk = 0

  while (i < lines.length) {
    const line = lines[i]

    // Fenced code block
    if (line.startsWith('```')) {
      const codeLines: string[] = []
      i++
      while (i < lines.length && !lines[i].startsWith('```')) {
        codeLines.push(lines[i])
        i++
      }
      i++ // skip closing ```
      blocks.push(
        <pre
          key={bk++}
          style={{
            border: '1px solid var(--color-border)',
            padding: '10px 12px',
            overflowX: 'auto',
            margin: '8px 0',
            background: 'var(--color-surface-raised)',
            lineHeight: '1.5',
          }}
        >
          <code>{codeLines.join('\n')}</code>
        </pre>
      )
      continue
    }

    // Headings
    const hm = line.match(/^(#{1,6})\s+(.+)$/)
    if (hm) {
      blocks.push(
        <h3
          key={bk++}
          style={{
            letterSpacing: '0.12em',
            color: 'var(--color-text-label)',
            margin: '12px 0 4px',
            fontWeight: 'bold',
            fontSize: 'var(--text-sm)',
          }}
        >
          {parseInline(hm[2])}
        </h3>
      )
      i++
      continue
    }

    // Empty line
    if (line.trim() === '') {
      i++
      continue
    }

    // Paragraph — accumulate until empty line / heading / fence
    const paraLines: string[] = []
    while (
      i < lines.length &&
      lines[i].trim() !== '' &&
      !lines[i].startsWith('```') &&
      !lines[i].match(/^#{1,6}\s/)
    ) {
      paraLines.push(lines[i])
      i++
    }
    if (paraLines.length > 0) {
      const children: React.ReactNode[] = []
      paraLines.forEach((l, j) => {
        children.push(...parseInline(l))
        if (j < paraLines.length - 1) children.push(<br key={`br-${j}`} />)
      })
      blocks.push(
        <p key={bk++} style={{ margin: '4px 0', lineHeight: '1.6' }}>
          {children}
        </p>
      )
    }
  }

  return blocks
}

interface Props {
  content: string
  className?: string
}

export default function MarkdownRenderer({ content, className }: Props) {
  const blocks = useMemo(() => parseBlocks(content), [content])
  return (
    <div
      className={className}
      style={{ fontSize: 'var(--text-sm)', color: 'var(--color-text-primary)' }}
    >
      {blocks}
    </div>
  )
}
