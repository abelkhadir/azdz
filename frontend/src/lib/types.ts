export type User = {
  id: number
  email?: string
  first_name: string
  last_name: string
  birth_date?: string
  avatar: string
  nickname: string
  about?: string
  is_public: boolean
  is_following?: boolean
  is_follower?: boolean
  can_chat?: boolean
  follow_status?: string
}

export type Comment = { id: number; post_id: number; content: string; image: string; created_at: string; user: User }
export type Post = { id: number; title: string; content: string; image: string; privacy: string; created_at: string; group_id?: number; user: User; comments: Comment[]; comment_count: number }
export type Notification = { id: number; actor_id: number; type: string; reference_id: number; message: string; is_read: boolean; created_at: string }
export type Group = { id: number; creator_id: number; title: string; description: string; member_count: number; status: string; request_id?: number; created_at?: string }
export type Event = { id: number; title: string; description: string; event_time: string; response: string; going_count: number; not_going_count: number }
export type Message = { id: number; sender_id: number; recipient_id?: number; group_id?: number; content: string; created_at: string; sender?: string; type?: string }
export type Conversation = User & { last_message: string; last_message_at: string; unread_count: number }
export type ProfileData = { user: User; followers: User[]; following: User[]; post_count: number; posts: Post[] }
