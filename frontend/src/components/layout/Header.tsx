import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'

import { SearchBar } from '@/components/catalog/SearchBar'
import { utilityLinks } from '@/config/storefront'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { logoutThunk } from '@/store/slices/authSlice'
import { fetchCategoriesThunk } from '@/store/slices/categoriesSlice'
import { fetchCartThunk } from '@/store/slices/cartSlice'
import { fetchPlacesThunk } from '@/store/slices/placesSlice'

import styles from '@/components/layout/Header.module.scss'

interface NavItem {
  to: string
  label: string
  icon: 'user' | 'orders' | 'heart' | 'cart' | 'account' | 'admin' | 'store'
}

const buyerItems: NavItem[] = [
  { to: '/account', label: 'Кабинет', icon: 'account' },
  { to: '/account/orders', label: 'Заказы', icon: 'orders' },
  { to: '/favorites', label: 'Избранное', icon: 'heart' },
  { to: '/cart', label: 'Корзина', icon: 'cart' },
]

const sellerItems: NavItem[] = [
  { to: '/seller', label: 'Магазин', icon: 'store' },
  { to: '/account', label: 'Кабинет', icon: 'account' },
  { to: '/favorites', label: 'Избранное', icon: 'heart' },
  { to: '/cart', label: 'Корзина', icon: 'cart' },
]

const adminItems: NavItem[] = [{ to: '/admin', label: 'Админка', icon: 'admin' }]

const renderIcon = (icon: NavItem['icon']) => {
  switch (icon) {
    case 'orders':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M7 4.5h10m-10 5h10m-10 5h6M6 3h12a2 2 0 0 1 2 2v14l-4-2.4L12 19l-4-2.4L4 19V5a2 2 0 0 1 2-2Z" />
        </svg>
      )
    case 'heart':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 20.4 4.9 13.7A4.6 4.6 0 0 1 11.6 7l.4.5.4-.5a4.6 4.6 0 0 1 6.7 6.3L12 20.4Z" />
        </svg>
      )
    case 'cart':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 5h2l1.2 8.1a1.7 1.7 0 0 0 1.7 1.4h7.7a1.7 1.7 0 0 0 1.7-1.3L20 8H7.1" />
          <circle cx="10" cy="19" r="1.2" />
          <circle cx="17" cy="19" r="1.2" />
        </svg>
      )
    case 'store':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 8.5 5.4 4h13.2L20 8.5" />
          <path d="M5 9v9.5a1.5 1.5 0 0 0 1.5 1.5h11a1.5 1.5 0 0 0 1.5-1.5V9" />
          <path d="M9 20v-5.5h6V20" />
        </svg>
      )
    case 'account':
    case 'user':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 12.2a4.1 4.1 0 1 0-4.1-4.1 4.1 4.1 0 0 0 4.1 4.1Zm-7 7.1a7 7 0 0 1 14 0" />
        </svg>
      )
    case 'admin':
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3.7 19 6.5v4.1c0 4.3-2.7 8.1-7 9.7-4.3-1.6-7-5.4-7-9.7V6.5L12 3.7Z" />
          <path d="M12 8v4l2.2 2.2" />
        </svg>
      )
    default:
      return null
  }
}

export const Header = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuLocationToken, setMenuLocationToken] = useState<string | null>(null)
  const headerRef = useRef<HTMLElement>(null)

  const auth = useAppSelector((state) => state.auth)
  const cartTotalItems = useAppSelector((state) => state.cart.totalItems)
  const categoriesState = useAppSelector((state) => state.categories)
  const placesState = useAppSelector((state) => state.places)

  const locationToken = `${location.pathname}${location.search}`
  const menuOpen = menuLocationToken === locationToken

  const navItems = useMemo(() => {
    if (!auth.isAuthenticated) {
      return []
    }

    if (auth.user?.role === 'admin') {
      return [...buyerItems, ...adminItems]
    }

    if (auth.user?.role === 'seller') {
      return sellerItems
    }

    return buyerItems
  }, [auth.isAuthenticated, auth.user?.role])

  const topLinks = useMemo(() => {
    if (auth.user?.role === 'seller') {
      return utilityLinks.map((item, index) => (index === 0 ? { href: '/seller', label: 'Кабинет продавца' } : item))
    }

    return utilityLinks
  }, [auth.user?.role])

  useEffect(() => {
    if (categoriesState.status === 'idle') {
      dispatch(fetchCategoriesThunk())
    }
  }, [categoriesState.status, dispatch])

  useEffect(() => {
    if (!auth.isAuthenticated) {
      return
    }

    dispatch(fetchCartThunk())

    if (placesState.status === 'idle') {
      dispatch(fetchPlacesThunk())
    }
  }, [auth.isAuthenticated, dispatch, placesState.status])

  useEffect(() => {
    if (!menuOpen) {
      return
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!headerRef.current?.contains(event.target as Node)) {
        setMenuLocationToken(null)
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setMenuLocationToken(null)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleEscape)

    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [menuOpen])

  const handleLogout = async () => {
    await dispatch(logoutThunk())
    navigate('/login')
  }

  const selectedPlace = placesState.items[0]
  const deliveryLabel = selectedPlace?.title || 'Укажите адрес доставки'
  const searchParams = new URLSearchParams(location.search)
  const searchValue = location.pathname === '/' ? searchParams.get('q') ?? '' : ''
  const quickCategories = categoriesState.items.slice(0, 8)

  const handleSearch = (value: string) => {
    const nextParams = new URLSearchParams()
    if (value.trim()) {
      nextParams.set('q', value.trim())
    }

    navigate({
      pathname: '/',
      search: nextParams.toString() ? `?${nextParams.toString()}` : '',
    })
    setMenuLocationToken(null)
  }

  const openCategory = (categoryId: string) => {
    const nextParams = new URLSearchParams()
    nextParams.set('category_id', categoryId)
    navigate({ pathname: '/', search: `?${nextParams.toString()}` })
    setMenuLocationToken(null)
  }

  return (
    <header className={styles.header} ref={headerRef}>
      <div className={styles.utilityBar}>
        <div className={styles.utilityInner}>
          <Link to={auth.isAuthenticated ? '/account/places' : '/login'} className={styles.delivery}>
            <span className={styles.deliveryBadge} aria-hidden="true">
              <svg viewBox="0 0 28 28" role="presentation" focusable="false">
                <rect x="1" y="1" width="26" height="26" rx="9" fill="#fff" />
                <path d="M1 10.33h26V18H1z" fill="#1f5fff" />
                <path d="M1 18h26v9H1z" fill="#e53935" />
                <rect x="1" y="1" width="26" height="26" rx="9" fill="none" stroke="rgba(15, 23, 42, 0.08)" />
              </svg>
            </span>
            <div>
              <strong>{deliveryLabel}</strong>
              <span>Доставка и самовывоз</span>
            </div>
          </Link>

          <div className={styles.utilityLinks}>
            {topLinks.map((item) => (
              <Link key={item.label} to={item.href}>
                {item.label}
              </Link>
            ))}
          </div>
        </div>
      </div>

      <div className={styles.mainBar}>
        <div className={styles.mainInner}>
          <div className={styles.brandRow}>
            <Link to="/" className={styles.brand} aria-label="На главную">
              <span className={styles.brandMark}>M</span>
              <span className={styles.brandText}>Маркет</span>
            </Link>

            <button
              type="button"
              className={`${styles.catalogButton} ${menuOpen ? styles.catalogButtonActive : ''}`}
              aria-label="Открыть каталог"
              aria-expanded={menuOpen}
              aria-controls="catalog-sheet"
              onClick={() => setMenuLocationToken((current) => (current === locationToken ? null : locationToken))}
            >
              <span className={styles.catalogBurger} aria-hidden="true" />
              Каталог
            </button>
          </div>

          <div className={styles.search}>
            <SearchBar
              initialValue={searchValue}
              onSearch={handleSearch}
              variant="header"
              placeholder="Искать товары и категории"
              submitLabel="Найти"
            />
          </div>

          <div className={styles.actions}>
            {auth.isAuthenticated ? (
              <>
                {navItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      `${styles.actionLink} ${item.to === '/cart' ? styles.cartAction : ''} ${isActive ? styles.active : ''}`
                    }
                  >
                    <span className={styles.actionIcon}>{renderIcon(item.icon)}</span>
                    <span className={styles.actionLabel}>{item.label}</span>
                    {item.to === '/cart' && cartTotalItems > 0 ? <span className={styles.cartCount}>{cartTotalItems}</span> : null}
                  </NavLink>
                ))}
                <button type="button" className={styles.authButton} onClick={handleLogout}>
                  Выйти
                </button>
              </>
            ) : (
              <>
                <Link to="/login" className={styles.actionLink}>
                  <span className={styles.actionIcon}>{renderIcon('user')}</span>
                  <span className={styles.actionLabel}>Войти</span>
                </Link>
                <Link to="/register" className={styles.authButton}>
                  Регистрация
                </Link>
              </>
            )}
          </div>
        </div>
      </div>

      <div className={styles.quickBar}>
        <div className={styles.quickInner}>
          {quickCategories.map((category) => (
            <button key={category.id} type="button" className={styles.quickCategory} onClick={() => openCategory(category.id)}>
              {category.name}
            </button>
          ))}
        </div>
      </div>

      {menuOpen ? (
        <div className={styles.mobileSheet} id="catalog-sheet">
          <div className={styles.mobileContent}>
            <div className={styles.mobileSection}>
              <div className={styles.mobileSectionHeader}>
                <strong>Разделы каталога</strong>
                <span>{categoriesState.items.length} категорий</span>
              </div>
              <div className={`${styles.mobileLinks} ${styles.mobileCatalogGrid}`}>
                {categoriesState.items.map((category) => (
                  <button
                    key={category.id}
                    type="button"
                    className={`${styles.mobileNavButton} ${styles.mobileCatalogButton}`}
                    onClick={() => openCategory(category.id)}
                  >
                    {category.name}
                  </button>
                ))}
              </div>
            </div>

            <div className={styles.mobileSection}>
              <div className={styles.mobileSectionHeader}>
                <strong>Навигация</strong>
                <span>{auth.isAuthenticated ? 'Быстрый доступ' : 'Вход и аккаунт'}</span>
              </div>
              <div className={`${styles.mobileLinks} ${styles.mobileNavList}`}>
                {(auth.isAuthenticated ? navItems : []).map((item) => (
                  <NavLink key={item.to} to={item.to} className={styles.mobileNavButton} onClick={() => setMenuLocationToken(null)}>
                    <span className={styles.mobileNavIcon}>{renderIcon(item.icon)}</span>
                    <span>{item.label}</span>
                  </NavLink>
                ))}
                {auth.user?.role === 'seller' ? (
                  <NavLink to="/seller/orders" className={styles.mobileNavButton} onClick={() => setMenuLocationToken(null)}>
                    <span className={styles.mobileNavIcon}>{renderIcon('orders')}</span>
                    <span>Заказы продавца</span>
                  </NavLink>
                ) : null}
                {auth.user?.role === 'admin' ? (
                  <NavLink to="/admin" className={styles.mobileNavButton} onClick={() => setMenuLocationToken(null)}>
                    <span className={styles.mobileNavIcon}>{renderIcon('admin')}</span>
                    <span>Админка</span>
                  </NavLink>
                ) : null}
                {!auth.isAuthenticated ? (
                  <>
                    <NavLink to="/login" className={styles.mobileNavButton} onClick={() => setMenuLocationToken(null)}>
                      <span className={styles.mobileNavIcon}>{renderIcon('user')}</span>
                      <span>Войти</span>
                    </NavLink>
                    <NavLink to="/register" className={styles.mobileNavButton} onClick={() => setMenuLocationToken(null)}>
                      <span className={styles.mobileNavIcon}>{renderIcon('account')}</span>
                      <span>Создать аккаунт</span>
                    </NavLink>
                  </>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </header>
  )
}
