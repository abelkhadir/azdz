'use client'

import { useParams } from 'next/navigation'
import { ProfileView } from '../page'

export default function ProfilePage() {
  const { id } = useParams<{ id: string }>()
  return <ProfileView id={id} />
}
