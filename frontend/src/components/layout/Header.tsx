import { useEffect, useMemo, useState } from 'react'
import { Link, NavLink, useNavigate } from 'react-router-dom'

import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { logoutThunk } from '@/store/slices/authSlice'
import { fetchCartThunk } from '@/store/slices/cartSlice'

import styles from '@/components/layout/Header.module.scss'

interface NavItem {
  to: string
  label: string
}

const publicItems: NavItem[] = [
  { to: '/', label: 'Catalog' },
]

const privateItems: NavItem[] = [
  { to: '/cart', label: 'Cart' },
  { to: '/favorites', label: 'Favorites' },
  { to: '/account/orders', label: 'Orders' },
  { to: '/account', label: 'Account' },
  { to: '/account/places', label: 'Places' },
]

const adminItems: NavItem[] = [
  { to: '/admin', label: 'Admin' },
]

export const Header = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const auth = useAppSelector((state) => state.auth)
  const cartTotalItems = useAppSelector((state) => state.cart.totalItems)

  const navItems = useMemo(() => {
    if (!auth.isAuthenticated) {
      return publicItems
    }

    if (auth.user?.role === 'admin') {
      return [...publicItems, ...privateItems, ...adminItems]
    }

    return [...publicItems, ...privateItems]
  }, [auth.isAuthenticated, auth.user?.role])

  useEffect(() => {
    if (auth.isAuthenticated) {
      dispatch(fetchCartThunk())
    }
  }, [auth.isAuthenticated, dispatch])

  const handleLogout = async () => {
    await dispatch(logoutThunk())
    navigate('/login')
  }

  return (
    <header className={styles.header}>
      <div className={styles.inner}>
        <Link to="/" className={styles.brand}>
          Marketplace
        </Link>

        <button
          type="button"
          className={styles.mobileToggle}
          aria-label="Open menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((prev) => !prev)}
        >
          {menuOpen ? 'Close' : 'Menu'}
        </button>

        <nav
          className={`${styles.nav} ${menuOpen ? styles.navOpen : ''}`}
          aria-label="Main menu"
        >
          {navItems.map((item) => (
            item.to === '/cart' ? (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) => `${styles.navLink} ${styles.cartWrapper} ${isActive ? styles.active : ''}`}
                onClick={() => setMenuOpen(false)}
              >
                Cart
                {cartTotalItems > 0 ? <span className={styles.cartCount}>{cartTotalItems}</span> : null}
              </NavLink>
            ) : (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) => `${styles.navLink} ${isActive ? styles.active : ''}`}
                onClick={() => setMenuOpen(false)}
              >
                {item.label}
              </NavLink>
            )
          ))}
        </nav>

        <div className={styles.actions}>
          {auth.isAuthenticated ? (
            <button type="button" className={styles.authButton} onClick={handleLogout}>
              Logout
            </button>
          ) : (
            <>
              <Link to="/login" className={styles.authButton}>
                Login
              </Link>
              <Link to="/register" className={styles.authButtonSecondary}>
                Register
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
