'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { errorMessage, request } from '@/lib/api'

export default function LoginPage() {
  const [status, setStatus] = useState({ busy: false, error: '' })

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    setStatus({ busy: true, error: '' })
    try {
      await request('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email: form.get('email'), password: form.get('password') }),
      })
      window.location.href = '/'
    } catch (error) {
      setStatus({ busy: false, error: errorMessage(error) })
    }
  }

  return <main className="auth-page simple-auth">
    <section className="auth-intro"><h1>circle</h1></section>
    <section className="auth-card">
      <Link href="/" className="brand"><span className="brand-mark">c</span><b>circle</b></Link>
      <div><h2>Log in</h2></div>
      <form onSubmit={submit}>
        <label>Email<input required type="email" name="email" /></label>
        <label>Password<input required minLength={8} type="password" name="password" /></label>
        {status.error && <p className="error">{status.error}</p>}
        <button className="primary wide" disabled={status.busy}>{status.busy ? 'Loading…' : 'Log in'}</button>
      </form>
      <p className="auth-switch">No account? <Link href="/register">Register</Link></p>
    </section>
  </main>
}

