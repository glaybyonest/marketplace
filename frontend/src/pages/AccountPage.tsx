import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchProfileThunk, updateProfileThunk } from '@/store/slices/userSlice'

import styles from '@/pages/AccountPage.module.scss'

export const AccountPage = () => {
  const dispatch = useAppDispatch()
  const authUser = useAppSelector((state) => state.auth.user)
  const { profile, status, updateStatus, error } = useAppSelector((state) => state.user)

  const [fullNameDraft, setFullNameDraft] = useState<string | undefined>(undefined)

  useEffect(() => {
    dispatch(fetchProfileThunk())
  }, [dispatch])

  const fullName = fullNameDraft ?? profile?.fullName ?? authUser?.fullName ?? authUser?.name ?? ''

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await dispatch(updateProfileThunk({ fullName }))
  }

  const email = profile?.email ?? authUser?.email ?? '-'

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1>Account</h1>
          <p>Manage your profile and quick links.</p>
        </div>
      </header>

      {status === 'loading' ? <AppLoader label="Loading profile..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.card}>
        <form className={styles.form} onSubmit={handleSubmit}>
          <label>
            Full name
            <input
              value={fullName}
              onChange={(event) => setFullNameDraft(event.target.value)}
            />
          </label>
          <label>
            Email
            <input value={email} disabled />
          </label>
          <button type="submit" disabled={updateStatus === 'loading'}>
            {updateStatus === 'loading' ? 'Saving...' : 'Save'}
          </button>
        </form>

        <nav className={styles.quickLinks}>
          <Link to="/cart">Cart</Link>
          <Link to="/account/orders">Orders</Link>
          <Link to="/favorites">Favorites</Link>
          <Link to="/account/places">My places</Link>
          <Link to="/account/sessions">Sessions</Link>
          {authUser?.role === 'admin' ? <Link to="/admin">Admin</Link> : null}
        </nav>
      </section>
    </div>
  )
}
