'use client'

import { useState, type FormEvent } from 'react'
import Link from 'next/link'
import Shell from '@/components/Shell'
import { group } from '@/lib/adapters'
import { errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import type { Group } from '@/lib/types'

type GroupRequest = { id: number; groupId: number; type: string }

async function loadGroups() {
  const [groups, requests] = await Promise.all([
    request<{ groups: unknown[] }>('/api/groups?limit=50&offset=0'),
    request<{ requests: GroupRequest[] }>('/api/group-requests'),
  ])
  return groups.groups.map(group).map(item => {
    const invite = requests.requests.find(request => request.groupId === item.id && request.type === 'invite')
    return invite ? { ...item, status: 'invited', request_id: invite.id } : item
  })
}

export default function GroupsPage() {
  const { data: groups, error: loadError, reload } = useData<Group[]>(loadGroups)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  async function createGroup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    try {
      await request('/api/groups', { method: 'POST', body: JSON.stringify({ title: form.get('title'), description: form.get('description') }) })
      setCreating(false); setError(''); await reload()
    } catch (problem) { setError(errorMessage(problem)) }
  }

  async function join(id: number) {
    try { await request(`/api/groups/${id}/requests`, { method: 'POST', body: JSON.stringify({ type: 'join_request' }) }); setError(''); await reload() }
    catch (problem) { setError(errorMessage(problem)) }
  }

  async function respondInvite(requestID: number, accept: boolean) {
    try { await request(`/api/group-requests/${requestID}`, { method: 'PATCH', body: JSON.stringify({ status: accept ? 'accepted' : 'declined' }) }); setError(''); await reload() }
    catch (problem) { setError(errorMessage(problem)) }
  }

  return <Shell><div className="narrow">
    <div className="page-heading heading-row"><h1>Groups</h1><button className="primary" onClick={() => setCreating(!creating)}>Create group</button></div>
    {(loadError || error) && <p className="error">{loadError || error}</p>}
    {creating && <form className="card create-form" onSubmit={createGroup}><label>Name<input required name="title" maxLength={100} /></label><label>Description<textarea required name="description" maxLength={2000} /></label><button className="primary">Create</button></form>}
    <div className="group-grid">{groups?.map(item => <article className="group-card card" key={item.id}>
      <div className="group-art">{item.title[0]}</div><small>{item.member_count} members</small><h2>{item.title}</h2><p>{item.description}</p>
      {item.status === 'accepted' ? <Link className="button primary" href={`/groups/${item.id}`}>Open</Link> : item.status === 'invited' ? <div className="form-row"><button className="primary" onClick={() => respondInvite(item.request_id!, true)}>Accept</button><button onClick={() => respondInvite(item.request_id!, false)}>Decline</button></div> : <button onClick={() => join(item.id)} disabled={item.status === 'requested'}>{item.status === 'requested' ? 'Requested' : 'Join'}</button>}
    </article>)}</div>
  </div></Shell>
}
