import { useState, useEffect, useRef, type CSSProperties } from 'react'

interface Props {
  onConfirm: () => void
  disabled?: boolean
  style?: CSSProperties
}

export default function ConfirmButton({ onConfirm, disabled, style }: Props) {
  const [confirming, setConfirming] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  function handleClick(e: React.MouseEvent) {
    e.stopPropagation()
    if (!confirming) {
      setConfirming(true)
      timer.current = setTimeout(() => setConfirming(false), 3000)
    } else {
      if (timer.current) clearTimeout(timer.current)
      setConfirming(false)
      onConfirm()
    }
  }

  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])

  return (
    <button
      onClick={handleClick}
      disabled={disabled}
      style={{
        background: 'transparent',
        border: 'none',
        fontSize: '12px',
        padding: '0 4px',
        cursor: 'pointer',
        color: confirming ? 'var(--color-accent-red)' : 'var(--color-text-dim)',
        fontFamily: 'var(--font-mono)',
        letterSpacing: confirming ? '0' : undefined,
        ...style,
      }}
      onMouseEnter={e => !confirming && (e.currentTarget.style.color = 'var(--color-accent-red)')}
      onMouseLeave={e => !confirming && (e.currentTarget.style.color = 'var(--color-text-dim)')}
    >
      {confirming ? 'ok?' : '×'}
    </button>
  )
}
