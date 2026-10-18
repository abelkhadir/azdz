'use client'

import { useState, type FormEvent } from 'react'
import Avatar from '@/components/Avatar'
import InfiniteTrigger from '@/components/InfiniteTrigger'
import PostCard from '@/components/PostCard'
import Shell from '@/components/Shell'
import { post, user } from '@/lib/adapters'
import { errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import { usePagedList } from '@/lib/usePagedList'
import type { Post, User } from '@/lib/types'

const readPosts = (value: unknown) => (value as unknown[]).map(post)

async function loadFeed(): Promise<{ me: User; people: User[]; followers: User[] }> {
  const meResult = await request<{ user: unknown }>('/api/auth/me')
  const me = user(meResult.user)
  const [peopleResult, followersResult] = await Promise.all([request<{ users: unknown[] }>('/api/users/search?q='), request<{ users: unknown[] }>(`/api/users/${me.id}/followers`)])
  return { me, people: peopleResult.users.map(user), followers: followersResult.users.map(user) }
}

export default function HomePage() {
  const { data, error, reload } = useData(loadFeed)
  const posts = usePagedList<Post>('/api/posts', readPosts)
  const [composer, setComposer] = useState({ open: false, privacy: 'public' })
  const [actionError, setActionError] = useState('')

  async function createPost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const body = new FormData(form)
    body.set('allowedUserIds', JSON.stringify(body.getAll('viewer_id').map(Number)))
    try {
      await request('/api/posts', { method: 'POST', body })
      form.reset()
      setComposer({ open: false, privacy: 'public' })
      setActionError('')
      await posts.reset()
    } catch (problem) { setActionError(errorMessage(problem)) }
  }

  return <Shell><div className="feed-layout"><section>
    <div className="page-heading"><h1>Feed</h1></div>
    <div className="composer card"><Avatar user={data?.me} /><button onClick={() => setComposer(current => ({ ...current, open: true }))}>Create post</button></div>
    {composer.open && <form className="create-form card" onSubmit={createPost}>
      <input required name="title" maxLength={100} placeholder="Title" /><textarea name="content" maxLength={5000} placeholder="Write something" />
      <div className="form-row"><select name="privacy" value={composer.privacy} onChange={event => setComposer(current => ({ ...current, privacy: event.target.value }))}><option value="public">Public</option><option value="followers">Followers</option><option value="private">Selected followers</option></select><input type="file" name="image" accept="image/jpeg,image/png,image/gif" /></div>
      {composer.privacy === 'private' && <fieldset className="viewer-picker"><legend>Who can see this?</legend>{data?.followers.map(person => <label key={person.id}><input type="checkbox" name="viewer_id" value={person.id} /> {person.first_name} {person.last_name}</label>)}</fieldset>}
      <div className="row-end"><button type="button" onClick={() => setComposer({ open: false, privacy: 'public' })}>Cancel</button><button className="primary">Post</button></div>
    </form>}
    {(error || posts.error || actionError) && <p className="error">{error || posts.error || actionError}</p>}
    <div className="post-list">{posts.items.map(item => <PostCard post={item} key={item.id} />)}</div>
    {posts.items.length === 0 && !posts.loading && <div className="empty card"><p>No posts yet.</p></div>}
    <InfiniteTrigger hasMore={posts.hasMore} loading={posts.loading} onLoad={posts.loadMore} />
  </section><aside className="right-rail card"><h3>People</h3>{data?.people.slice(0, 5).map(person => <PersonRow key={person.id} person={person} reload={reload} />)}</aside></div></Shell>
}

function PersonRow({ person, reload }: { person: User; reload: () => Promise<void> }) {
  const [error, setError] = useState('')
  async function follow() {
    const method = person.follow_status === 'accepted' ? 'DELETE' : 'PUT'
    try { await request(`/api/users/${person.id}/follow`, { method }); setError(''); await reload() }
    catch (problem) { setError(errorMessage(problem)) }
  }
  return <div className="person-mini"><Avatar user={person} /><div><b>{person.first_name} {person.last_name}</b><small>{error || `@${person.nickname || person.first_name.toLowerCase()}`}</small></div><button className="mini" disabled={person.follow_status === 'pending'} onClick={follow}>{person.follow_status === 'accepted' ? 'Unfollow' : person.follow_status === 'pending' ? 'Requested' : 'Follow'}</button></div>
}
