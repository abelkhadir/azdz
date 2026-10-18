'use client'

import { Suspense, useEffect, useRef, useState, type FormEvent, type UIEvent } from 'react'
import { useSearchParams } from 'next/navigation'
import Avatar from '@/components/Avatar'
import Shell from '@/components/Shell'
import { conversation as readConversation, message, user } from '@/lib/adapters'
import { API, errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import type { Conversation, Message, User } from '@/lib/types'

type ChatData = { me: User; conversations: Conversation[] }

async function loadChat(): Promise<ChatData> {
  const me = user((await request<{ user: unknown }>('/api/auth/me')).user)
  const [recent, following, followers, unread] = await Promise.all([
    request<unknown[]>('/api/messages/conversations'),
    request<{ users: unknown[] }>(`/api/users/${me.id}/following`),
    request<{ users: unknown[] }>(`/api/users/${me.id}/followers`),
    request<{ counts: Record<string, number> }>('/api/notifications/chat'),
  ])
  const people = new Map<number, Conversation>()
  for (const item of [...following.users, ...followers.users]) {
    const person = user(item)
    people.set(person.id, { ...person, last_message: '', last_message_at: '', unread_count: unread.counts[String(person.id)] || 0 })
  }
  for (const item of recent.map(readConversation)) people.set(item.id, { ...item, unread_count: unread.counts[String(item.id)] || 0 })
  return { me, conversations: [...people.values()].sort((a, b) => b.last_message_at.localeCompare(a.last_message_at)) }
}

export default function ChatPage() {
  return <Suspense fallback={<main className="center"><div className="spinner" /></main>}><Chat /></Suspense>
}

function Chat() {
  const search = useSearchParams()
  const { data, error: loadError, reload } = useData(loadChat)
  const [active, setActive] = useState<Conversation | null>(null)
  const [items, setItems] = useState<Message[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const socket = useRef<WebSocket | null>(null)
  const activeID = useRef(0)
  const box = useRef<HTMLDivElement | null>(null)
  const meID = data?.me.id

  useEffect(() => {
    if (!meID) return
    const connection = new WebSocket(`${API.replace(/^http/, 'ws')}/ws`)
    connection.onmessage = event => {
      const packet = JSON.parse(event.data) as { type: string; data: unknown }
      if (packet.type === 'notifications_read') { void reload(); return }
      if (packet.type === 'message_error') {
        setError(String((packet.data as { message?: string }).message || 'Message was not sent.'))
        return
      }
      if (packet.type !== 'private_message') return
      const next = message(packet.data)
      const other = next.sender_id === meID ? next.recipient_id : next.sender_id
      if (other === activeID.current) setItems(current => current.some(item => item.id === next.id) ? current : [...current, next])
      void reload()
    }
    connection.onopen = () => {
      if (activeID.current) connection.send(JSON.stringify({ type: 'chat_view', data: { kind: 'private', id: activeID.current } }))
    }
    socket.current = connection
    return () => connection.close()
  }, [meID, reload])

  useEffect(() => {
    const wanted = Number(search.get('userId'))
    const person = data?.conversations.find(item => item.id === wanted)
    if (person && activeID.current !== person.id) void openConversation(person)
    // Opening from ?userId= only needs to run when the loaded chat list changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, search])

  async function openConversation(person: Conversation) {
    activeID.current = person.id
    setActive(person); setItems([]); setLoading(true); setError('')
    try {
      const page = await request<unknown[]>(`/api/messages/history?withUserId=${person.id}&limit=10`)
      if (activeID.current !== person.id) return
      setItems(page.map(message)); setHasMore(page.length === 10)
      await request('/api/notifications/chat/read', { method: 'POST', body: JSON.stringify({ actor_id: String(person.id) }) })
      if (socket.current?.readyState === WebSocket.OPEN) socket.current.send(JSON.stringify({ type: 'chat_view', data: { kind: 'private', id: person.id } }))
      await reload()
      requestAnimationFrame(() => { if (box.current) box.current.scrollTop = box.current.scrollHeight })
    } catch (problem) { setError(errorMessage(problem)) }
    finally { setLoading(false) }
  }

  async function loadOlder(event: UIEvent<HTMLDivElement>) {
    const element = event.currentTarget
    if (element.scrollTop > 40 || loading || !hasMore || !active || items.length === 0) return
    const oldHeight = element.scrollHeight
    setLoading(true)
    try {
      const page = await request<unknown[]>(`/api/messages/history?withUserId=${active.id}&limit=10&beforeId=${items[0].id}`)
      const older = page.map(message)
      setItems(current => [...older.filter(item => !current.some(old => old.id === item.id)), ...current])
      setHasMore(older.length === 10)
      requestAnimationFrame(() => { element.scrollTop = element.scrollHeight - oldHeight })
    } catch (problem) { setError(errorMessage(problem)) }
    finally { setLoading(false) }
  }

  function send(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const input = event.currentTarget.elements.namedItem('message') as HTMLInputElement
    if (!active || !input.value.trim()) return
    if (socket.current?.readyState !== WebSocket.OPEN) { setError('Chat is connecting. Try again.'); return }
    setError('')
    socket.current.send(JSON.stringify({ type: 'private_message', data: { receiverId: active.id, content: input.value.trim(), clientId: crypto.randomUUID() } }))
    input.value = ''
  }

  return <Shell>{loadError && <p className="error">{loadError}</p>}<div className="chat-layout card">
    <aside className="chat-list"><h2>Messages</h2>{data?.conversations.map(person => <button className={active?.id === person.id ? 'active' : ''} onClick={() => openConversation(person)} key={person.id}><Avatar user={person} /><span><b>{person.first_name} {person.last_name}{person.unread_count > 0 && <em className="chat-unread">{person.unread_count}</em>}</b><small>{person.last_message || `@${person.nickname || person.first_name.toLowerCase()}`}</small></span></button>)}</aside>
    <section className="chat-main">{active ? <><header><Avatar user={active} /><b>{active.first_name} {active.last_name}</b></header><div className="messages" ref={box} onScroll={loadOlder}>{loading && <small className="loading-old">Loading…</small>}{items.map(item => <div className={`bubble ${item.sender_id === data?.me.id ? 'mine' : ''}`} key={item.id}>{item.content}<small>{new Date(item.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</small></div>)}</div>{error && <p className="error compact">{error}</p>}<form className="message-form" onSubmit={send}><input name="message" maxLength={2000} placeholder="Message" /><button className="primary">Send</button></form></> : <div className="empty"><p>Select a conversation.</p></div>}</section>
  </div></Shell>
}
