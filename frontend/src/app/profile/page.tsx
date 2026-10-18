'use client'

import { useState } from 'react'
import Link from 'next/link'
import Avatar from '@/components/Avatar'
import InfiniteTrigger from '@/components/InfiniteTrigger'
import PostCard from '@/components/PostCard'
import Shell from '@/components/Shell'
import { post, user } from '@/lib/adapters'
import { errorMessage, request } from '@/lib/api'
import { useData } from '@/lib/useData'
import { usePagedList } from '@/lib/usePagedList'
import type { Post, User } from '@/lib/types'

type ProfileData = { user: User; followers: User[]; following: User[] }
const postPage = (value: unknown) => (value as unknown[]).map(post)

async function loadProfile(id?: string): Promise<ProfileData> {
  let profile: unknown
  if (id) profile = (await request<{ user: unknown }>(`/profile/${id}`)).user
  else profile = (await request<{ user: unknown }>('/api/auth/me')).user
  const profileUser = user(profile)
  const [followers, following] = await Promise.all([
    request<{ users: unknown[] }>(`/api/users/${profileUser.id}/followers`),
    request<{ users: unknown[] }>(`/api/users/${profileUser.id}/following`),
  ])
  return { user: profileUser, followers: followers.users.map(user), following: following.users.map(user) }
}

export default function MyProfilePage() {
  return <ProfileView own />
}

export function ProfileView({ id, own = false }: { id?: string; own?: boolean }) {
  const { data, error: loadError, reload } = useData(() => loadProfile(id), id)
  const [error, setError] = useState('')

  async function togglePrivacy() {
    if (!data) return
    try {
      await request('/api/auth/me', { method: 'PATCH', body: JSON.stringify({ isPublic: !data.user.is_public }) })
      setError('')
      await reload()
    } catch (problem) { setError(errorMessage(problem)) }
  }

  return <Shell><div className="narrow">
    {(loadError || error) && <div className="empty card"><p>{loadError || error}</p></div>}
    {data && <><ProfileHeader data={data} own={own} onPrivacy={togglePrivacy} /><ProfilePosts ownerID={data.user.id} /></>}
  </div></Shell>
}

function ProfilePosts({ ownerID }: { ownerID: number }) {
  const posts = usePagedList<Post>(`/api/posts?ownerId=${ownerID}`, postPage)
  return <><div className="post-list">{posts.items.map(item => <PostCard key={item.id} post={item} />)}</div>{posts.items.length === 0 && !posts.loading && <div className="empty card"><p>{posts.error || 'No visible posts.'}</p></div>}<InfiniteTrigger hasMore={posts.hasMore} loading={posts.loading} onLoad={posts.loadMore} /></>
}

function ProfileHeader({ data, own, onPrivacy }: { data: ProfileData; own: boolean; onPrivacy: () => void }) {
  const profile = data.user
  return <section className="profile-head card">
    <div className="profile-cover" /><Avatar user={profile} size="xl" />
    <div className="profile-info"><div><h1>{profile.first_name} {profile.last_name}</h1><p>@{profile.nickname || profile.first_name.toLowerCase()}</p></div>{own && <button onClick={onPrivacy}>{profile.is_public ? 'Make private' : 'Make public'}</button>}</div>
    <div className="profile-details">{profile.email && <span>{profile.email}</span>}{profile.birth_date && <span>Born {profile.birth_date}</span>}</div>
    {profile.about && <p className="bio">{profile.about}</p>}
    <div className="stats"><span><b>{data.followers.length}</b> followers</span><span><b>{data.following.length}</b> following</span><span className="privacy">{profile.is_public ? 'Public' : 'Private'}</span></div>
    <div className="profile-connections"><ConnectionList title="Followers" users={data.followers} /><ConnectionList title="Following" users={data.following} /></div>
  </section>
}

function ConnectionList({ title, users }: { title: string; users: User[] }) {
  return <div><b>{title}</b>{users.length === 0 ? <small>None</small> : users.map(item => <Link href={`/profile/${item.id}`} key={item.id}><Avatar user={item} size="sm" />{item.first_name} {item.last_name}</Link>)}</div>
}
