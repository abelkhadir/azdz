'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { errorMessage, request } from '@/lib/api'

export default function RegisterPage() {
  const [status, setStatus] = useState({ busy: false, error: '' })

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setStatus({ busy: true, error: '' })
    try {
      await request('/api/auth/register', { method: 'POST', body: new FormData(event.currentTarget) })
      window.location.href = '/feed'
    } catch (error) {
      setStatus({ busy: false, error: errorMessage(error) })
    }
  }

  return <main className="auth-page simple-auth">
    <section className="auth-intro"><h1>circle</h1></section>
    <section className="auth-card">
      <Link href="/" className="brand"><span className="brand-mark">c</span><b>circle</b></Link>
      <div><h2>Create account</h2></div>
      <form onSubmit={submit}>
        <div className="form-row">
          <label>First name<input required name="firstName" maxLength={14} /></label>
          <label>Last name<input required name="lastName" maxLength={14} /></label>
        </div>
        <label>Email<input required type="email" name="email" /></label>
        <div className="form-row">
          <label>Password<input required minLength={8} type="password" name="password" /></label>
          <label>Birth date <small>16–100 years old</small><input required type="date" name="dateOfBirth" /></label>
        </div>
        <div className="form-row">
          <label>Nickname <small>optional</small><input name="nickname" maxLength={31} /></label>
          <label>Avatar <small>optional</small><input type="file" name="avatar" accept="image/jpeg,image/png,image/gif" /></label>
        </div>
        <label>About <small>optional</small><textarea name="aboutMe" maxLength={500} rows={2} /></label>
        {status.error && <p className="error">{status.error}</p>}
        <button className="primary wide" disabled={status.busy}>{status.busy ? 'Creating…' : 'Register'}</button>
      </form>
      <p className="auth-switch">Have an account? <Link href="/login">Log in</Link></p>
    </section>
  </main>
}

