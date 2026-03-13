import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { registerThunk } from '@/store/slices/authSlice'

import styles from '@/pages/AuthPage.module.scss'

const PASSWORD_MIN_LEN = 8
const PASSWORD_MAX_LEN = 72

const hasLatinLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value)

const validateForm = (name: string, email: string, password: string): string | null => {
  const trimmedName = name.trim()
  const trimmedEmail = email.trim()

  if (!trimmedName) {
    return '??????? ???'
  }

  if (trimmedName.length > 120) {
    return '??? ?????? ???? ?? ??????? 120 ????????'
  }

  if (!trimmedEmail) {
    return '??????? email'
  }

  if (password.length < PASSWORD_MIN_LEN || password.length > PASSWORD_MAX_LEN) {
    return '?????? ?????? ???? ?? 8 ?? 72 ????????'
  }

  if (!hasLatinLetterAndDigit(password)) {
    return '?????? ?????? ????????? ??????? ???? ????????? ????? ? ???? ?????'
  }

  return null
}

export const RegisterPage = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const authState = useAppSelector((state) => state.auth)

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const validationError = validateForm(name, email, password)
    if (validationError) {
      setLocalError(validationError)
      return
    }

    setLocalError(null)

    const result = await dispatch(
      registerThunk({
        name: name.trim(),
        email: email.trim(),
        password,
      }),
    )

    if (registerThunk.fulfilled.match(result)) {
      navigate('/')
    }
  }

  const shownError = localError ?? authState.error

  return (
    <div className={styles.page}>
      <section className={styles.card}>
        <h1>???????????</h1>
        <p>???????? ??????? ??????????.</p>
        <form className={styles.form} onSubmit={handleSubmit}>
          <label>
            ???
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              autoComplete="name"
              maxLength={120}
              required
            />
          </label>
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
              minLength={PASSWORD_MIN_LEN}
              maxLength={PASSWORD_MAX_LEN}
              autoComplete="new-password"
              required
            />
          </label>
          <p className={styles.hint}>??????? 8 ????????, ????????? ????? ? ?????.</p>
          {shownError ? <ErrorMessage message={shownError} /> : null}
          <button type="submit" disabled={authState.status === 'loading'}>
            {authState.status === 'loading' ? '??????? ???????...' : '??????????????????'}
          </button>
        </form>
        <p>
          ??? ???? ???????? <Link to="/login">?????</Link>
        </p>
      </section>
    </div>
  )
}
