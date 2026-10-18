'use client'

import Link from 'next/link'
import { useEffect, useState, type ReactNode } from 'react'
import InfiniteTrigger from './InfiniteTrigger'
import { API, errorMessage, request } from '@/lib/api'
import { notification, user } from '@/lib/adapters'
import { useData } from '@/lib/useData'
import { usePagedList } from '@/lib/usePagedList'
import type { Notification as Note, User } from '@/lib/types'
import Avatar from './Avatar'

const links = [['/', 'Home'], ['/people', 'People'], ['/groups', 'Groups'], ['/chat', 'Chat'], ['/profile', 'Profile']]
type NotificationResponse = { notifications?: unknown[]; totalUnread?: number }
const readNotifications = (value: unknown) => ((value as NotificationResponse).notifications || []).map(notification)

async function loadShell(): Promise<{ me: User; unread: number }> {
  const [me, notes] = await Promise.all([
    request<{ user: unknown }>('/api/auth/me'),
    request<NotificationResponse>('/api/notifications?limit=1'),
  ])
  return { me: user(me.user), unread: notes.totalUnread || 0 }
}

export default function Shell({ children }: { children: ReactNode }) {
  const { data, error, reload } = useData(loadShell)
  const notifications = usePagedList<Note>('/api/notifications', readNotifications)
  const resetNotifications = notifications.reset
  const ready = Boolean(data)
  const [open, setOpen] = useState(false)
  const [actionError, setActionError] = useState('')

  useEffect(() => {
    if (!ready) return
    const socket = new WebSocket(`${API.replace(/^http/, 'ws')}/ws`)
    socket.onmessage = event => {
      const update = JSON.parse(event.data) as { type?: string }
      if (update.type === 'notification' || update.type === 'notifications_read') {
        void reload()
        void resetNotifications()
      }
    }
    return () => socket.close()
  }, [ready, reload, resetNotifications])

  if (error) return <main className="center"><Link className="button primary" href="/login">Log in</Link></main>
  if (!data) return <main className="center"><div className="spinner" /></main>

  async function toggleNotifications() {
    const next = !open
    setOpen(next)
    if (!next) return
    try {
      setActionError('')
      await request('/api/notifications/read', { method: 'POST', body: JSON.stringify({ all: true }) })
      await Promise.all([reload(), notifications.reset()])
    } catch (problem) { setActionError(errorMessage(problem)) }
  }

  async function logout() {
    try {
      await request('/api/auth/logout', { method: 'POST' })
      window.location.href = '/login'
    } catch (problem) { setActionError(errorMessage(problem)) }
  }

  const messageNotes = notifications.items.filter(note => ['private_message', 'group_message'].includes(note.type))
  const otherNotes = notifications.items.filter(note => !['private_message', 'group_message'].includes(note.type))
  const refresh = async () => { await Promise.all([reload(), notifications.reset()]) }

  return <div className="app-shell">
    <aside className="sidebar">
      <Link href="/" className="brand"><span className="brand-mark">c</span><b>circle</b></Link>
      <nav>{links.map(([href, label]) => <Link href={href} key={href}>{label}</Link>)}</nav>
      <div className="side-profile"><Avatar user={data.me} /><div><b>{data.me.first_name} {data.me.last_name}</b><small>@{data.me.nickname || data.me.first_name.toLowerCase()}</small></div></div>
    </aside>
    <section className="main-wrap">
      <header className="topbar">
        <Link href="/" className="mobile-brand">circle</Link>
        <div className="top-actions"><button className="icon-button" onClick={toggleNotifications} aria-label="Notifications">●{data.unread > 0 && <em>{data.unread}</em>}</button><button className="quiet" onClick={logout}>Log out</button></div>
        {open && <div className="notification-panel">
          <h3>Notifications</h3>
          {(actionError || notifications.error) && <p className="error compact">{actionError || notifications.error}</p>}
          {notifications.items.length === 0 && !notifications.loading && <p className="muted">No notifications</p>}
          {messageNotes.length > 0 && <><h4>Messages</h4>{messageNotes.map(note => <NotificationItem key={note.id} note={note} refresh={refresh} />)}</>}
          {otherNotes.length > 0 && <><h4>Updates</h4>{otherNotes.map(note => <NotificationItem key={note.id} note={note} refresh={refresh} />)}</>}
          <InfiniteTrigger hasMore={notifications.hasMore} loading={notifications.loading} onLoad={notifications.loadMore} />
        </div>}
      </header>
      <main className="page">{children}</main>
    </section>
  </div>
}

function NotificationItem({ note, refresh }: { note: Note; refresh: () => Promise<void> }) {
  const [error, setError] = useState('')
  const href = note.type === 'private_message' ? `/chat?userId=${note.actor_id}` : note.type === 'group_message' || note.type === 'group_event' ? `/groups/${note.reference_id}` : note.type.startsWith('follow') ? `/profile/${note.actor_id}` : '/groups'
  const actionable = ['follow_request', 'group_invite', 'group_join_request'].includes(note.type)

  async function answer(accept: boolean) {
    try {
      setError('')
      if (note.type === 'follow_request') {
        await request(`/api/follow-requests/${note.actor_id}`, { method: accept ? 'PUT' : 'DELETE' })
      } else {
        await request(`/api/group-requests/${note.reference_id}`, { method: 'PATCH', body: JSON.stringify({ status: accept ? 'accepted' : 'declined' }) })
      }
      await refresh()
    } catch (problem) { setError(errorMessage(problem)) }
  }

  return <div className="notification"><Link href={href}><span>{note.message}</span></Link>{error && <small className="error compact">{error}</small>}{actionable && <div><button className="mini primary" onClick={() => answer(true)}>Accept</button><button className="mini" onClick={() => answer(false)}>Decline</button></div>}</div>
}
