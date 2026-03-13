import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { loginThunk } from '@/store/slices/authSlice'

import styles from '@/pages/AuthPage.module.scss'

export const LoginPage = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const location = useLocation()
  const authState = useAppSelector((state) => state.auth)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const from = (location.state as { from?: string } | null)?.from ?? '/'

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const result = await dispatch(loginThunk({ email: email.trim(), password }))
    if (loginThunk.fulfilled.match(result)) {
      navigate(from, { replace: true })
    }
  }

  return (
    <div className={styles.page}>
      <section className={styles.card}>
        <h1>????</h1>
        <p>???????, ????? ???????? ? ?????????, ???????? ? ????????.</p>
        <form className={styles.form} onSubmit={handleSubmit}>
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              autoComplete="email"
              maxLength={254}
              required
            />
          </label>
          <label>
            ??????
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              required
            />
          </label>
          {authState.error ? <ErrorMessage message={authState.error} /> : null}
          <button type="submit" disabled={authState.status === 'loading'}>
            {authState.status === 'loading' ? '??????...' : '?????'}
          </button>
        </form>
        <p>
          ??? ????????? <Link to="/register">??????????????????</Link>
        </p>
      </section>
    </div>
  )
}
