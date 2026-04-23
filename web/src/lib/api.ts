import { clearToken, getToken } from './auth'

export const authEvents = new EventTarget()

export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })

  if (res.status === 401) {
    clearToken()
    authEvents.dispatchEvent(new Event('unauth'))
    throw new Error('unauthenticated')
  }

  if (res.status === 204) {
    return undefined as T
  }

  let data: Record<string, unknown>
  try {
    data = await res.json()
  } catch {
    throw new Error(`HTTP ${res.status}`)
  }

  if (!res.ok) {
    throw new Error((data.error as string) ?? `HTTP ${res.status}`)
  }

  return data as T
}
