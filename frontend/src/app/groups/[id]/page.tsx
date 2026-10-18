'use client'

import { useEffect, useRef, useState, type FormEvent, type UIEvent } from 'react'
import { useParams } from 'next/navigation'
import InfiniteTrigger from '@/components/InfiniteTrigger'
import PostCard from '@/components/PostCard'
import Shell from '@/components/Shell'
import { group, groupEvent, message, post, user } from '@/lib/adapters'
import { API, errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import { usePagedList } from '@/lib/usePagedList'
import type { Event, Group, Message, Post, User } from '@/lib/types'

type PageData = { group: Group; events: Event[]; people: User[] }
const readPosts = (value: unknown) => ((value as { posts: unknown[] }).posts || []).map(post)

export default function GroupPage() {
  const { id } = useParams<{ id: string }>()
  const { data, error: loadError, reload } = useData(async (): Promise<PageData> => {
    const detail = group((await request<{ group: unknown }>(`/api/groups/${id}`)).group)
    if (detail.status !== 'accepted') return { group: detail, events: [], people: [] }
    const [events, people] = await Promise.all([request<{ events: unknown[] }>(`/api/groups/${id}/events?limit=50&offset=0`), request<{ users: unknown[] }>('/api/users/search?q=')])
    return { group: detail, events: events.events.map(groupEvent), people: people.users.map(user) }
  }, id)
  const posts = usePagedList<Post>(data?.group.status === 'accepted' ? `/api/groups/${id}/posts` : '', readPosts)
  const [view, setView] = useState<'posts' | 'events' | 'chat' | 'invite'>('posts')
  const [error, setError] = useState('')

  async function createPost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const content = String(new FormData(form).get('content') || '')
    try { await request(`/api/groups/${id}/posts`, { method: 'POST', body: JSON.stringify({ content }) }); form.reset(); setError(''); await posts.reset() }
    catch (problem) { setError(errorMessage(problem)) }
  }

  async function createEvent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const values = new FormData(form)
    const date = new Date(String(values.get('dateTime')))
    if (Number.isNaN(date.getTime())) { setError('Choose a valid date.'); return }
    try { await request(`/api/groups/${id}/events`, { method: 'POST', body: JSON.stringify({ title: values.get('title'), description: values.get('description'), dateTime: date.toISOString() }) }); form.reset(); setError(''); await reload() }
    catch (problem) { setError(errorMessage(problem)) }
  }

  async function vote(eventID: number, value: string) {
    try { await request(`/api/events/${eventID}/response`, { method: 'PUT', body: JSON.stringify({ response: value }) }); setError(''); await reload() }
    catch (problem) { setError(errorMessage(problem)) }
  }

  async function invite(userID: number) {
    try { await request(`/api/groups/${id}/requests`, { method: 'POST', body: JSON.stringify({ type: 'invite', userId: userID }) }); setError('') }
    catch (problem) { setError(errorMessage(problem)) }
  }

  return <Shell><div className="narrow">
    {loadError && <p className="error">{loadError}</p>}
    {data && <><header className="group-hero"><h1>{data.group.title}</h1><p>{data.group.description}</p><small>{data.group.member_count} members</small></header>
      {data.group.status !== 'accepted' ? <div className="empty card"><p>Join this group to view its content.</p></div> : <>
        <div className="tabs">{(['posts', 'events', 'chat', 'invite'] as const).map(tab => <button className={view === tab ? 'active' : ''} onClick={() => setView(tab)} key={tab}>{tab}</button>)}</div>
        {error && <p className="error">{error}</p>}
        {view === 'posts' && <><form className="composer card create-form" onSubmit={createPost}><textarea name="content" required maxLength={5000} placeholder="New post" /><button className="primary">Post</button></form><div className="post-list">{posts.items.map(item => <PostCard key={item.id} post={item} group />)}</div><InfiniteTrigger hasMore={posts.hasMore} loading={posts.loading} onLoad={posts.loadMore} /></>}
        {view === 'events' && <><form className="card create-form" onSubmit={createEvent}><input name="title" required maxLength={100} placeholder="Event name" /><textarea name="description" required maxLength={2000} placeholder="Description" /><input name="dateTime" type="datetime-local" required /><button className="primary">Create event</button></form><div className="event-list">{data.events.map(item => <article className="event card" key={item.id}><small>{new Date(item.event_time).toLocaleString()}</small><h3>{item.title}</h3><p>{item.description}</p><div><button className={item.response === 'going' ? 'primary' : ''} onClick={() => vote(item.id, 'going')}>Going · {item.going_count}</button><button className={item.response === 'not_going' ? 'primary' : ''} onClick={() => vote(item.id, 'not_going')}>Not going · {item.not_going_count}</button></div></article>)}</div></>}
        {view === 'chat' && <GroupChat groupID={Number(id)} />}
        {view === 'invite' && <section className="card invite-list"><h2>Invite people</h2>{data.people.map(person => <div className="person-mini" key={person.id}><span>{person.first_name} {person.last_name}</span><button className="mini" onClick={() => invite(person.id)}>Invite</button></div>)}</section>}
      </>}
    </>}
  </div></Shell>
}

function GroupChat({ groupID }: { groupID: number }) {
  const [items, setItems] = useState<Message[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const socket = useRef<WebSocket | null>(null)
  const box = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    void request<unknown[]>(`/api/group-messages/history?groupId=${groupID}&limit=10`).then(page => {
      setItems(page.map(message)); setHasMore(page.length === 10)
      requestAnimationFrame(() => { if (box.current) box.current.scrollTop = box.current.scrollHeight })
    }).catch(problem => setError(errorMessage(problem))).finally(() => setLoading(false))
    const connection = new WebSocket(`${API.replace(/^http/, 'ws')}/ws`)
    connection.onopen = () => connection.send(JSON.stringify({ type: 'chat_view', data: { kind: 'group', id: groupID } }))
    connection.onmessage = event => {
      const packet = JSON.parse(event.data) as { type: string; data: unknown }
      if (packet.type === 'message_error') {
        setError(String((packet.data as { message?: string }).message || 'Message was not sent.'))
        return
      }
      if (packet.type !== 'group_message') return
      const next = message(packet.data)
      if (next.group_id === groupID) setItems(current => current.some(item => item.id === next.id) ? current : [...current, next])
    }
    socket.current = connection
    return () => connection.close()
  }, [groupID])

  async function loadOlder(event: UIEvent<HTMLDivElement>) {
    const element = event.currentTarget
    if (element.scrollTop > 40 || loading || !hasMore || items.length === 0) return
    const oldHeight = element.scrollHeight
    setLoading(true)
    try {
      const page = await request<unknown[]>(`/api/group-messages/history?groupId=${groupID}&limit=10&beforeId=${items[0].id}`)
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
    if (!input.value.trim()) return
    if (socket.current?.readyState !== WebSocket.OPEN) { setError('Chat is connecting. Try again.'); return }
    setError('')
    socket.current.send(JSON.stringify({ type: 'group_message', data: { groupId: groupID, content: input.value.trim(), clientId: crypto.randomUUID() } }))
    input.value = ''
  }

  return <section className="chat-room card"><div className="messages" ref={box} onScroll={loadOlder}>{loading && <small className="loading-old">Loading…</small>}{items.map(item => <div className="message" key={item.id}><b>{item.sender || 'Member'}</b><p>{item.content}</p></div>)}</div>{error && <p className="error compact">{error}</p>}<form className="message-form" onSubmit={send}><input name="message" maxLength={2000} placeholder="Message" /><button className="primary">Send</button></form></section>
}
