import { useEffect } from 'react'
import { Link } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchFavoritesThunk, removeFavoriteThunk } from '@/store/slices/favoritesSlice'
import { formatCurrency } from '@/utils/format'
import { resolveProductImage, resolveProductImageFallback, swapImageToFallback } from '@/utils/media'
import { getProductPath } from '@/utils/productRef'

import styles from '@/pages/FavoritesPage.module.scss'

export const FavoritesPage = () => {
  const dispatch = useAppDispatch()
  const { items, status, mutationStatus, error } = useAppSelector((state) => state.favorites)

  useEffect(() => {
    dispatch(fetchFavoritesThunk({ page: 1, limit: 50 }))
  }, [dispatch])

  const handleDelete = (productId: string) => {
    dispatch(removeFavoriteThunk(productId))
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="badge-pill">Избранное</span>
          <h1>Сохранённые товары</h1>
          <p>Возвращайтесь к понравившимся позициям, сравнивайте цены и открывайте карточки товара без повторного поиска.</p>
        </div>
      </header>

      {status === 'loading' ? <AppLoader label="Загружаем избранное..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {items.length === 0 ? (
        <section className={`${styles.empty} empty-state`}>
          <h2>В избранном пока ничего нет</h2>
          <p>Сохраняйте товары с главной или из карточки товара, чтобы быстро вернуться к ним позже.</p>
          <Link to="/" className="action-primary">
            Перейти в каталог
          </Link>
        </section>
      ) : (
        <section className={styles.list}>
          {items.map((item) => (
            <article key={item.id} className={styles.item}>
              <Link to={getProductPath(item)} className={styles.imageWrap}>
                <img
                  src={resolveProductImage(item)}
                  alt={item.title}
                  className={styles.image}
                  onError={(event) => swapImageToFallback(event.currentTarget, resolveProductImageFallback(item))}
                />
              </Link>

              <div className={styles.itemInfo}>
                <span className="badge-pill">{item.categoryName || 'Каталог'}</span>
                <h2>
                  <Link to={getProductPath(item)}>{item.title}</Link>
                </h2>
                <p>{item.brand || 'Товар из вашего каталога'}</p>
              </div>

              <div className={styles.itemActions}>
                <strong>{formatCurrency(item.price, item.currency)}</strong>
                <Link to={getProductPath(item)} className="action-secondary">
                  Открыть товар
                </Link>
                <button type="button" className="action-danger" onClick={() => handleDelete(item.id)} disabled={mutationStatus === 'loading'}>
                  Удалить
                </button>
              </div>
            </article>
          ))}
        </section>
      )}
    </div>
  )
}
