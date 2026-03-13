import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { AdminNav } from '@/components/admin/AdminNav'
import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { categoryService } from '@/services/categoryService'
import { productService } from '@/services/productService'
import { getErrorMessage } from '@/utils/error'

import styles from '@/pages/AdminPage.module.scss'

interface AdminDashboardState {
  categoriesTotal: number
  productsTotal: number
  activeProducts: number
  hiddenProducts: number
}

export const AdminDashboardPage = () => {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<AdminDashboardState>({
    categoriesTotal: 0,
    productsTotal: 0,
    activeProducts: 0,
    hiddenProducts: 0,
  })

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      setError(null)

      try {
        const [categories, productsAll, productsActive, productsHidden] = await Promise.all([
          categoryService.getCategories(),
          productService.getAdminProducts({ page: 1, limit: 1 }),
          productService.getAdminProducts({ page: 1, limit: 1, is_active: true }),
          productService.getAdminProducts({ page: 1, limit: 1, is_active: false }),
        ])

        setStats({
          categoriesTotal: categories.length,
          productsTotal: productsAll.total,
          activeProducts: productsActive.total,
          hiddenProducts: productsHidden.total,
        })
      } catch (loadError) {
        setError(getErrorMessage(loadError, 'Failed to load admin dashboard'))
      } finally {
        setLoading(false)
      }
    }

    void load()
  }, [])

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div>
          <h1>Admin backoffice</h1>
          <p>Manage categories, products, and inventory from one place.</p>
        </div>
        <AdminNav />
      </section>

      {loading ? <AppLoader label="Loading admin dashboard..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {!loading && !error ? (
        <>
          <section className={styles.statsGrid}>
            <article>
              <h2>Categories</h2>
              <p className={styles.statsValue}>{stats.categoriesTotal}</p>
            </article>
            <article>
              <h2>Total products</h2>
              <p className={styles.statsValue}>{stats.productsTotal}</p>
            </article>
            <article>
              <h2>Active products</h2>
              <p className={styles.statsValue}>{stats.activeProducts}</p>
            </article>
          </section>

          <section className={styles.contentGrid}>
            <article className={styles.panel}>
              <h2>Catalog operations</h2>
              <p>Create and edit categories, keep slugs consistent, and avoid orphan branches.</p>
              <div className={styles.formActions}>
                <Link to="/admin/categories" className={styles.primaryButton}>
                  Manage categories
                </Link>
              </div>
            </article>

            <article className={styles.panel}>
              <h2>Inventory operations</h2>
              <p>
                Update product cards, prices, and stock. Hidden products: {stats.hiddenProducts}.
              </p>
              <div className={styles.formActions}>
                <Link to="/admin/products" className={styles.primaryButton}>
                  Manage products
                </Link>
              </div>
            </article>
          </section>
        </>
      ) : null}
    </div>
  )
}
