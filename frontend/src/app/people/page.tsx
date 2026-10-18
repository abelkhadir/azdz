'use client'

import { useState } from 'react'
import Link from 'next/link'
import Avatar from '@/components/Avatar'
import Shell from '@/components/Shell'
import { user } from '@/lib/adapters'
import { errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import type { User } from '@/lib/types'

const loadPeople = async () => (await request<{ users: unknown[] }>('/api/users/search?q=')).users.map(user)

export default function PeoplePage() {
  const { data: people, error: loadError, reload } = useData(loadPeople)
  const [search, setSearch] = useState('')
  const [actionError, setActionError] = useState('')
  const visible = people?.filter(person => `${person.first_name} ${person.last_name} ${person.nickname}`.toLowerCase().includes(search.toLowerCase())) || []

  async function toggleFollow(person: User) {
    const method = person.follow_status === 'accepted' ? 'DELETE' : 'PUT'
    try { await request(`/api/users/${person.id}/follow`, { method }); setActionError(''); await reload() }
    catch (problem) { setActionError(errorMessage(problem)) }
  }

  return <Shell><div className="narrow">
    <div className="page-heading"><h1>People</h1></div>
    {(loadError || actionError) && <p className="error">{loadError || actionError}</p>}
    <input className="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Search" />
    <div className="people-grid">{visible.map(person => <article className="person-card card" key={person.id}>
      <Link href={`/profile/${person.id}`}><Avatar user={person} size="lg" /><h3>{person.first_name} {person.last_name}</h3><p>@{person.nickname || person.first_name.toLowerCase()}</p></Link>
      <span className="privacy">{person.is_public ? 'Public' : 'Private'}</span>
      <button className={person.follow_status === 'accepted' ? '' : 'primary'} disabled={person.follow_status === 'pending'} onClick={() => toggleFollow(person)}>{person.follow_status === 'accepted' ? 'Unfollow' : person.follow_status === 'pending' ? 'Requested' : 'Follow'}</button>
    </article>)}</div>
  </div></Shell>
}
