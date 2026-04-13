import { useState, type FormEvent } from 'react'
import { setToken } from '../lib/auth'

interface LoginPageProps {
  onSuccess: () => void
}

export default function LoginPage({ onSuccess }: LoginPageProps) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error ?? 'Authentication failed')
        return
      }
      setToken(data.token)
      onSuccess()
    } catch {
      setError('Connection error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="min-h-screen flex items-center justify-center"
      style={{ background: 'var(--color-bg)', fontFamily: 'var(--font-mono)' }}
    >
      <div className="panel" style={{ width: '320px', padding: '32px' }}>
        <div className="flex flex-col items-center gap-8">
          <span
            style={{
              fontSize: 'var(--text-xl)',
              letterSpacing: '0.3em',
              textTransform: 'uppercase',
              color: 'var(--color-text-primary)',
            }}
          >
            HELM
          </span>

          <form onSubmit={handleSubmit} className="flex flex-col gap-3 w-full">
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="password"
              autoFocus
              style={{ width: '100%' }}
            />

            {error && (
              <span
                style={{
                  fontSize: 'var(--text-xs)',
                  color: 'var(--color-accent-red)',
                  letterSpacing: 'var(--letter-spacing-label)',
                }}
              >
                {error}
              </span>
            )}

            <button
              type="submit"
              className="btn-solid"
              disabled={loading || !password}
            >
              {loading ? 'authenticating...' : 'enter'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
