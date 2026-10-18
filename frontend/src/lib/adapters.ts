import type { Comment, Conversation, Event, Group, Message, Notification, Post, User } from './types'

type Data = Record<string, unknown>
const number = (value: unknown) => Number(value) || 0
const text = (value: unknown) => typeof value === 'string' ? value : ''

export function user(value: unknown): User {
  const item = (value || {}) as Data
  return { id: number(item.id), email: text(item.email), first_name: text(item.firstName), last_name: text(item.lastName), birth_date: text(item.dateOfBirth), avatar: text(item.avatar ?? item.avatarPath), nickname: text(item.nickname), about: text(item.aboutMe), is_public: Boolean(item.isPublic), follow_status: text(item.followStatus) }
}

export function comment(value: unknown): Comment {
  const item = (value || {}) as Data
  return { id: number(item.id), post_id: number(item.postId), content: text(item.content), image: text(item.imagePath), created_at: text(item.createdAt), user: user(item.author) }
}

export function post(value: unknown): Post {
  const item = (value || {}) as Data
  const comments = Array.isArray(item.comments) ? item.comments.map(comment) : []
  return { id: number(item.id), title: text(item.title), content: text(item.content), image: text(item.imagePath), privacy: text(item.privacy) || 'group', created_at: text(item.createdAt), group_id: number(item.groupId) || undefined, user: user(item.author), comments, comment_count: comments.length }
}

export function group(value: unknown): Group {
  const item = (value || {}) as Data
  const status = Boolean(item.isMember) ? 'accepted' : text(item.requestStatus) === 'pending' ? 'requested' : ''
  return { id: number(item.id), creator_id: number(item.creatorId), title: text(item.title), description: text(item.description), member_count: number(item.memberCount), status }
}

export function groupEvent(value: unknown): Event {
  const item = (value || {}) as Data
  return { id: number(item.id), title: text(item.title), description: text(item.description), event_time: text(item.dateTime), response: text(item.response), going_count: number(item.goingCount), not_going_count: number(item.notGoingCount) }
}

export function message(value: unknown): Message {
  const item = (value || {}) as Data
  return { id: number(item.id), sender_id: number(item.senderId), recipient_id: number(item.receiverId) || undefined, group_id: number(item.groupId) || undefined, content: text(item.content), created_at: text(item.createdAt), sender: text(item.senderName) }
}

export function conversation(value: unknown): Conversation {
  const item = (value || {}) as Data
  return { ...user({ id: item.userId, firstName: item.firstName, lastName: item.lastName, nickname: item.nickname, avatarPath: item.avatarPath }), last_message: text(item.lastMessage), last_message_at: text(item.lastMessageAt), unread_count: 0 }
}

export function notification(value: unknown): Notification {
  const item = (value || {}) as Data
  return { id: number(item.id), actor_id: number(item.actorId), type: text(item.type), reference_id: number(item.entityId ?? item.referenceId), message: text(item.content), is_read: Boolean(item.isRead), created_at: text(item.createdAt) }
}
