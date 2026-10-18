'use client'

import { useState, type FormEvent } from 'react'
import Link from 'next/link'
import { comment as readComment } from '@/lib/adapters'
import { errorMessage, imageUrl, request } from '@/lib/api'
import type { Comment, Post } from '@/lib/types'
import Avatar from './Avatar'
import InfiniteTrigger from './InfiniteTrigger'

export default function PostCard({ post, group = false }: { post: Post; group?: boolean; onChange?: () => void | Promise<void> }) {
  const [comments, setComments] = useState<Comment[]>(post.comments || [])
  const [commentsOpen, setCommentsOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [error, setError] = useState('')

  async function loadComments(reset = true) {
    if (loading) return
    setLoading(true)
    try {
      const after = !reset && comments.length ? `&afterId=${comments[comments.length - 1].id}` : ''
      const path = group ? `/api/group-comments?postId=${post.id}&limit=10${after}` : `/api/comments?postId=${post.id}&limit=10${after}`
      const result = await request<unknown>(path)
      const values = Array.isArray(result) ? result : ((result as { comments?: unknown[] }).comments || [])
      const page = values.map(readComment)
      setComments(current => reset ? page : [...current, ...page.filter(item => !current.some(old => old.id === item.id))])
      setHasMore(page.length === 10)
      setCommentsOpen(true)
      setError('')
    } catch (problem) { setError(errorMessage(problem)) }
    finally { setLoading(false) }
  }

  async function addComment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const body = new FormData(form)
    const content = String(body.get('content') || '').trim()
    const image = body.get('image')
    if (!content && (!(image instanceof File) || image.size === 0)) return
    setBusy(true)
    try {
      let created: unknown
      if (group) {
        const result = await request<{ comment: unknown }>(`/api/group-comments?postId=${post.id}`, { method: 'POST', body: JSON.stringify({ content }) })
        created = result.comment
      } else {
        body.set('postId', String(post.id))
        created = await request('/api/comments', { method: 'POST', body })
      }
      setComments(current => [...current, readComment(created)])
      setCommentsOpen(true)
      setError('')
      form.reset()
    } catch (problem) { setError(errorMessage(problem)) }
    finally { setBusy(false) }
  }

  return <article className="post card">
    <div className="post-head"><Avatar user={post.user} /><div><Link href={`/profile/${post.user.id}`}><b>{post.user.first_name} {post.user.last_name}</b></Link><small>{new Date(post.created_at).toLocaleString()} · {post.privacy}</small></div></div>
    {post.title && <h2 className="post-title">{post.title}</h2>}
    {post.content && <p className="post-copy">{post.content}</p>}
    {post.image && <img className="post-image" src={imageUrl(post.image)} alt="Post attachment" />}
    <button className="mini" type="button" onClick={() => loadComments(true)}>{commentsOpen ? 'Refresh comments' : 'Comments'}</button>
    {commentsOpen && <div className="comments">{comments.map(item => <div className="comment" key={item.id}><Avatar user={item.user} size="sm" /><div><b>{item.user.first_name} {item.user.last_name}</b>{item.content && <p>{item.content}</p>}{item.image && <img src={imageUrl(item.image)} alt="Comment attachment" />}</div></div>)}{comments.length === 0 && !loading && <small>No comments yet.</small>}<InfiniteTrigger hasMore={hasMore} loading={loading} onLoad={() => loadComments(false)} /></div>}
    {error && <p className="error compact">{error}</p>}
    <form className="comment-form" onSubmit={addComment}><input name="content" maxLength={2000} placeholder="Comment" />{!group && <label className="attach">＋<input name="image" type="file" accept="image/jpeg,image/png,image/gif" /></label>}<button disabled={busy}>Send</button></form>
  </article>
}
