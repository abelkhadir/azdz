import { imageUrl } from '@/lib/api'
import type { User } from '@/lib/types'

export default function Avatar({ user, size = 'md' }: { user?: User | null; size?: 'sm' | 'md' | 'lg' | 'xl' }) {
  const src = user?.avatar ? imageUrl(user.avatar) : '/default-avatar.svg'
  return <img className={`avatar ${size}`} src={src} alt="" onError={event => { event.currentTarget.src = '/default-avatar.svg' }} />
}

