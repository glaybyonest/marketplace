import clsx from 'clsx'
import { NavLink } from 'react-router-dom'

import styles from '@/components/seller/SellerNav.module.scss'

const items = [
  { to: '/seller', label: 'Главная', end: true },
  { to: '/seller/products', label: 'Товары' },
  { to: '/seller/storefront', label: 'Магазин' },
  { to: '/seller/messages', label: 'Сообщения' },
  { to: '/seller/orders', label: 'Заказы' },
]

interface SellerNavProps {
  variant?: 'hero' | 'surface'
}

export const SellerNav = ({ variant = 'hero' }: SellerNavProps) => {
  const isSurface = variant === 'surface'

  return (
    <nav
      className={clsx(styles.nav, isSurface && styles.surface)}
      aria-label="Разделы кабинета продавца"
    >
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.end}
          className={({ isActive }) =>
            clsx(
              styles.link,
              isSurface && styles.surfaceLink,
              isActive && (isSurface ? styles.surfaceActive : styles.active),
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
