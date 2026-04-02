import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { RatingStars } from '@/components/common/RatingStars'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { addCartItemThunk } from '@/store/slices/cartSlice'
import { addFavoriteThunk, removeFavoriteThunk } from '@/store/slices/favoritesSlice'
import type { Product } from '@/types/domain'
import { formatCurrency, formatReviewCount } from '@/utils/format'
import { resolveProductImage } from '@/utils/media'
import { formatProductSpecLabel, formatProductSpecValue, getProductSpecEntries } from '@/utils/productSpecs'
import { getProductPath } from '@/utils/productRef'

import styles from '@/components/catalog/ProductCard.module.scss'

interface ProductCardProps {
  product: Product
}

export const ProductCard = ({ product }: ProductCardProps) => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const [imageLoaded, setImageLoaded] = useState(false)
  const auth = useAppSelector((state) => state.auth)
  const favoriteItems = useAppSelector((state) => state.favorites.items)
  const cartStatus = useAppSelector((state) => state.cart.mutationStatus)
  const image = useMemo(() => resolveProductImage(product), [product])

  const specPreview = useMemo(() => getProductSpecEntries(product.specs, 2), [product.specs])

  const discount = Math.min(32, Math.max(7, Math.round((product.stock ?? 0) / 10)))
  const oldPrice = Math.round(product.price * (1 + discount / 100))
  const isFavorite = favoriteItems.some((item) => item.id === product.id)
  const productPath = getProductPath(product)
  const reviewsCount = product.reviewsCount ?? 0
  const hasReviews = reviewsCount > 0
  const sellerLabel = product.sellerName || 'Партнерский магазин'

  const handleFavorite = async () => {
    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    if (isFavorite) {
      await dispatch(removeFavoriteThunk(product.id))
      return
    }

    await dispatch(addFavoriteThunk(product.id))
  }

  const handleAddToCart = async () => {
    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    await dispatch(addCartItemThunk({ productId: product.id, quantity: 1 }))
  }

  return (
    <article className={styles.card}>
      <div className={styles.badges}>
        <span className={styles.discountBadge}>-{discount}%</span>
        <button
          type="button"
          className={`${styles.favoriteButton} ${isFavorite ? styles.favoriteButtonActive : ''}`}
          onClick={handleFavorite}
          aria-label={isFavorite ? 'Убрать из избранного' : 'Добавить в избранное'}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 20.4 4.9 13.7A4.6 4.6 0 0 1 11.6 7l.4.5.4-.5a4.6 4.6 0 0 1 6.7 6.3L12 20.4Z" />
          </svg>
        </button>
      </div>

      <Link to={productPath} className={styles.imageWrap}>
        {!imageLoaded ? <div className={styles.skeleton} /> : null}
        <img
          src={image}
          alt={product.title}
          loading="lazy"
          className={imageLoaded ? styles.imageLoaded : styles.image}
          onLoad={() => setImageLoaded(true)}
        />
      </Link>

      <div className={styles.body}>
        <div className={styles.metaRow}>
          <span className={styles.category}>{product.categoryName || 'Каталог'}</span>
          {product.brand ? <span className={styles.brand}>{product.brand}</span> : null}
        </div>

        <h3 className={styles.title}>
          <Link to={productPath}>{product.title}</Link>
        </h3>

        <div className={styles.reviewSummary}>
          <RatingStars rating={hasReviews ? product.rating : 0} size="sm" />
          <strong>{hasReviews ? `${product.rating.toFixed(1)} / 5` : 'Новый товар'}</strong>
          <span>{hasReviews ? formatReviewCount(reviewsCount) : 'Пока без отзывов'}</span>
        </div>

        {specPreview.length > 0 ? (
          <ul className={styles.specList}>
            {specPreview.map(([key, value]) => (
              <li key={key}>
                <span>{formatProductSpecLabel(key)}</span>
                <strong>{formatProductSpecValue(value, key)}</strong>
              </li>
            ))}
          </ul>
        ) : (
          <p className={styles.description}>
            {product.description || 'Откройте карточку товара, чтобы посмотреть детали и характеристики.'}
          </p>
        )}

        <div className={styles.priceBlock}>
          <span className={styles.oldPrice}>{formatCurrency(oldPrice, product.currency)}</span>
          <strong className={styles.price}>{formatCurrency(product.price, product.currency)}</strong>
        </div>

        <div className={styles.footerRow}>
          <div>
            <span className={styles.deliveryText}>{sellerLabel}</span>
            <span className={styles.deliveryTextSecondary}>Доставка и самовывоз</span>
          </div>
          <button
            type="button"
            className={styles.cartButton}
            onClick={handleAddToCart}
            disabled={!product.stock || cartStatus === 'loading'}
            aria-label="Добавить в корзину"
            title="Добавить в корзину"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M4 5h2l1.2 8.1a1.7 1.7 0 0 0 1.7 1.4h7.7a1.7 1.7 0 0 0 1.7-1.3L20 8H7.1" />
              <circle cx="10" cy="19" r="1.2" />
              <circle cx="17" cy="19" r="1.2" />
            </svg>
          </button>
        </div>
      </div>
    </article>
  )
}
