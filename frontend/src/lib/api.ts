export const API = (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080').replace(/\/$/, '')

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  // The browser sets the multipart boundary when the body is FormData.
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')

  const response = await fetch(API + path, {
    ...options,
    headers,
    credentials: 'include', // Send the session cookie to the Go API.
  })
  const text = await response.text()
  let data: Record<string, unknown> = {}
  try { data = text ? JSON.parse(text) : {} } catch { /* Some errors are plain text. */ }
  if (!response.ok) throw new Error(String(data.message || data.error || text || 'Request failed'))
  return data as T
}

export function imageUrl(path?: string) {
  if (!path) return ''
  return /^https?:\/\//.test(path) ? path : API + (path.startsWith('/') ? path : `/${path}`)
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Request failed'
}

