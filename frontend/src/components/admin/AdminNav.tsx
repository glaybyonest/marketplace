import { NavLink } from 'react-router-dom'

import styles from '@/components/admin/AdminNav.module.scss'

const items = [
  { to: '/admin', label: 'Overview', end: true },
  { to: '/admin/categories', label: 'Categories' },
  { to: '/admin/products', label: 'Products' },
]

export const AdminNav = () => (
  <nav className={styles.nav} aria-label="Admin sections">
    {items.map((item) => (
      <NavLink
        key={item.to}
        to={item.to}
        end={item.end}
        className={({ isActive }) => `${styles.link} ${isActive ? styles.active : ''}`}
      >
        {item.label}
      </NavLink>
    ))}
  </nav>
)
