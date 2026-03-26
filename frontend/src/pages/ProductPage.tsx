import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { ProductGrid } from '@/components/catalog/ProductGrid'
import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { RatingStars } from '@/components/common/RatingStars'
import { productService } from '@/services/productService'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { addCartItemThunk } from '@/store/slices/cartSlice'
import { addFavoriteThunk, removeFavoriteThunk } from '@/store/slices/favoritesSlice'
import { fetchPlacesThunk } from '@/store/slices/placesSlice'
import { fetchProductThunk } from '@/store/slices/productsSlice'
import type { Product, Review } from '@/types/domain'
import { formatCurrency, formatDate, formatReviewCount, formatUnitLabel } from '@/utils/format'
import { getErrorMessage } from '@/utils/error'
import { resolveProductImage, resolveProductImageFallback, swapImageToFallback } from '@/utils/media'
import { formatProductSpecLabel, formatProductSpecValue, getProductSpecEntries } from '@/utils/productSpecs'

import styles from '@/pages/ProductPage.module.scss'

export const ProductPage = () => {
  const { productRef = '' } = useParams()
  const navigate = useNavigate()
  const dispatch = useAppDispatch()

  const auth = useAppSelector((state) => state.auth)
  const product = useAppSelector((state) => state.products.selectedProduct)
  const detailStatus = useAppSelector((state) => state.products.detailStatus)
  const productError = useAppSelector((state) => state.products.error)
  const favoriteItems = useAppSelector((state) => state.favorites.items)
  const favoriteMutationStatus = useAppSelector((state) => state.favorites.mutationStatus)
  const cartMutationStatus = useAppSelector((state) => state.cart.mutationStatus)
  const places = useAppSelector((state) => state.places.items)
  const placesStatus = useAppSelector((state) => state.places.status)

  const [selectedImages, setSelectedImages] = useState<Record<string, string>>({})
  const [quantities, setQuantities] = useState<Record<string, number>>({})
  const [relatedItems, setRelatedItems] = useState<Product[]>([])
  const [relatedStatus, setRelatedStatus] = useState<'idle' | 'loading' | 'succeeded' | 'failed'>('idle')
  const [reviews, setReviews] = useState<Review[]>([])
  const [reviewsStatus, setReviewsStatus] = useState<'idle' | 'loading' | 'succeeded' | 'failed'>('idle')
  const [reviewsError, setReviewsError] = useState<string | null>(null)
  const [reviewForm, setReviewForm] = useState({ rating: 5, comment: '' })
  const [reviewSubmitStatus, setReviewSubmitStatus] = useState<'idle' | 'loading' | 'succeeded' | 'failed'>('idle')
  const [reviewSubmitError, setReviewSubmitError] = useState<string | null>(null)

  useEffect(() => {
    if (!productRef) {
      return
    }
    dispatch(fetchProductThunk(productRef))
  }, [dispatch, productRef])

  useEffect(() => {
    if (!auth.isAuthenticated || placesStatus !== 'idle') {
      return
    }
    dispatch(fetchPlacesThunk())
  }, [auth.isAuthenticated, dispatch, placesStatus])

  useEffect(() => {
    let cancelled = false

    const loadReviews = async () => {
      if (!product?.id) {
        if (!cancelled) {
          setReviews([])
          setReviewsStatus('idle')
          setReviewsError(null)
        }
        return
      }

      setReviewsStatus('loading')
      setReviewsError(null)

      try {
        const items = await productService.getProductReviews(product.id)
        if (cancelled) {
          return
        }
        setReviews(items)
        setReviewsStatus('succeeded')
      } catch (error) {
        if (cancelled) {
          return
        }
        setReviews([])
        setReviewsStatus('failed')
        setReviewsError(getErrorMessage(error, 'Не удалось загрузить отзывы'))
      }
    }

    void loadReviews()

    return () => {
      cancelled = true
    }
  }, [product?.id])

  useEffect(() => {
    let cancelled = false
    const loadRelatedItems = async () => {
      if (!product?.categoryId) {
        if (!cancelled) {
          setRelatedItems([])
          setRelatedStatus('idle')
        }
        return
      }

      setRelatedStatus('loading')

      try {
        const response = await productService.getProducts({
          page: 1,
          limit: 8,
          category_id: product.categoryId,
        })

        if (cancelled) {
          return
        }

        setRelatedItems(response.items.filter((item) => item.id !== product.id).slice(0, 6))
        setRelatedStatus('succeeded')
      } catch {
        if (cancelled) {
          return
        }

        setRelatedItems([])
        setRelatedStatus('failed')
      }
    }

    void loadRelatedItems()

    return () => {
      cancelled = true
    }
  }, [product?.categoryId, product?.id])

  const isFavorite = useMemo(
    () => (product ? favoriteItems.some((item) => item.id === product.id) : false),
    [favoriteItems, product],
  )

  const gallery = useMemo(() => {
    if (!product) {
      return []
    }

    const rawGallery = product.images.length > 0 ? product.images : [product.imageUrl ?? '']
    return rawGallery.map((image, index) => ({
      primary: resolveProductImage(product, index, image),
      fallback: resolveProductImageFallback(product, index),
    }))
  }, [product])
  const selectedImage = product ? selectedImages[product.id] ?? '' : ''
  const activeImage = gallery.find((image) => image.primary === selectedImage) ?? gallery[0]
  const quantity = product ? quantities[product.id] ?? 1 : 1
  const safeQuantity = Math.min(quantity, Math.max(product?.stock || 1, 1))
  const specEntries = getProductSpecEntries(product?.specs)
  const oldPrice = product ? Math.round(product.price * 1.14) : 0
  const savings = product ? oldPrice - product.price : 0
  const isAvailable = Boolean(product?.stock && product.stock > 0)
  const deliveryPlace = places[0]
  const sellerLabel = product?.sellerName || 'Партнёрский магазин'
  const unitLabel = formatUnitLabel(product?.unit) || 'шт.'

  const reviewsCount = Math.max(product?.reviewsCount ?? 0, reviews.length)
  const averageRating =
    reviews.length > 0
      ? Number((reviews.reduce((sum, item) => sum + item.rating, 0) / reviews.length).toFixed(1))
      : Number((product?.rating ?? 0).toFixed(1))

  const handleToggleFavorite = async () => {
    if (!product) {
      return
    }

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
    if (!product) {
      return
    }

    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    await dispatch(addCartItemThunk({ productId: product.id, quantity: safeQuantity }))
    navigate('/cart')
  }

  const handleReviewSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!product) {
      return
    }

    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    setReviewSubmitStatus('loading')
    setReviewSubmitError(null)

    try {
      const created = await productService.createProductReview(product.id, reviewForm)
      setReviews((current) => [created, ...current])
      setReviewForm({ rating: 5, comment: '' })
      setReviewSubmitStatus('succeeded')
    } catch (error) {
      setReviewSubmitStatus('failed')
      setReviewSubmitError(getErrorMessage(error, 'Не удалось отправить отзыв'))
    }
  }

  if (detailStatus === 'loading') {
    return <AppLoader label="Загружаем карточку товара..." />
  }

  if (productError) {
    return <ErrorMessage message={productError} />
  }

  if (!product) {
    return <ErrorMessage message="Товар не найден" />
  }

  return (
    <div className={styles.page}>
      <nav className={styles.breadcrumbs} aria-label="Хлебные крошки">
        <Link to="/">Главная</Link>
        <span>/</span>
        <Link to={product.categoryId ? `/?category_id=${product.categoryId}` : '/'}>{product.categoryName ?? 'Каталог'}</Link>
        <span>/</span>
        <span>{product.title}</span>
      </nav>

      <section className={styles.layout}>
        <div className={`${styles.galleryPanel} page-card`}>
          <div className={styles.galleryRail}>
            <div className={styles.thumbList}>
              {gallery.map((image, index) => (
                <button
                  type="button"
                  key={`${image.primary}-${index}`}
                  className={image.primary === activeImage?.primary ? styles.thumbActive : styles.thumb}
                  onClick={() =>
                    setSelectedImages((current) => ({
                      ...current,
                      [product.id]: image.primary,
                    }))
                  }
                >
                  <img
                    src={image.primary}
                    alt={`${product.title} ${index + 1}`}
                    onError={(event) => swapImageToFallback(event.currentTarget, image.fallback)}
                  />
                </button>
              ))}
            </div>

            <div className={styles.mainStage}>
              <div className={styles.visualBadges}>
                {product.categoryName ? <span className="badge-pill">{product.categoryName}</span> : null}
                {savings > 0 ? <span className={`${styles.discountBadge} badge-pill`}>Экономия {formatCurrency(savings, product.currency)}</span> : null}
              </div>
              <img
                src={activeImage?.primary}
                alt={product.title}
                className={styles.mainImage}
                onError={(event) => swapImageToFallback(event.currentTarget, activeImage?.fallback ?? '')}
              />
            </div>
          </div>
        </div>

        <div className={styles.infoColumn}>
          <section className={`${styles.infoPanel} page-card`}>
            <div className={styles.badges}>
              {product.brand ? <span className={styles.badgeAlt}>{product.brand}</span> : null}
              <span className={styles.badge}>Артикул {product.sku || product.id.slice(0, 8)}</span>
              {product.unit ? <span className={styles.badge}>Единица: {unitLabel}</span> : null}
            </div>

            <h1 className={styles.title}>{product.title}</h1>

            <div className={styles.metaRow}>
              <span>{reviewsCount > 0 ? `Рейтинг ${averageRating.toFixed(1)} / 5 - ${formatReviewCount(reviewsCount)}` : 'Пока нет отзывов'}</span>
              <span>{product.categoryName ?? 'Категория каталога'}</span>
              <span>{sellerLabel}</span>
              <span>{isAvailable ? `В наличии: ${product.stock}` : 'Под заказ'}</span>
            </div>

            <div className={styles.priceRow}>
              <strong className={styles.currentPrice}>{formatCurrency(product.price, product.currency)}</strong>
              {savings > 0 ? <span className={styles.oldPrice}>{formatCurrency(oldPrice, product.currency)}</span> : null}
            </div>

            <div className={styles.reviewSummaryBar}>
              <RatingStars rating={reviewsCount > 0 ? averageRating : 0} size="md" />
              <strong>{reviewsCount > 0 ? `${averageRating.toFixed(1)} / 5` : 'Новый товар'}</strong>
              <span>{reviewsCount > 0 ? formatReviewCount(reviewsCount) : 'Пока без отзывов и комментариев'}</span>
            </div>

            <div className={isAvailable ? styles.stockChipActive : styles.stockChip}>
              {isAvailable ? 'Можно оформить доставку или самовывоз' : 'Товар временно недоступен для быстрой покупки'}
            </div>

            <p className={styles.description}>
              {product.description || 'Описание пока не добавлено. Откройте характеристики и детали получения ниже.'}
            </p>

            <div className={styles.statsGrid}>
              <article className={styles.statCard}>
                <span>Магазин</span>
                <strong>{sellerLabel}</strong>
              </article>
              <article className={styles.statCard}>
                <span>Категория</span>
                <strong>{product.categoryName || 'Каталог'}</strong>
              </article>
              <article className={styles.statCard}>
                <span>Артикул</span>
                <strong>{product.sku || product.id.slice(0, 8)}</strong>
              </article>
              <article className={styles.statCard}>
                <span>Формат продажи</span>
                <strong>{unitLabel}</strong>
              </article>
            </div>
          </section>

          <section className={`${styles.sectionCard} page-card`}>
            <div className={styles.sectionHeader}>
              <div>
                <span className="badge-pill">Доставка</span>
                <h2>Условия получения</h2>
              </div>
            </div>
            <div className={styles.deliveryGrid}>
              <article className={styles.deliveryCard}>
                <strong>Курьером или в пункт выдачи</strong>
                <p>
                  {deliveryPlace
                    ? `${deliveryPlace.title}: ${deliveryPlace.addressText}`
                    : 'Выберите адрес в профиле, чтобы видеть персональный сценарий доставки.'}
                </p>
              </article>
              <article className={styles.deliveryCard}>
                <strong>Оплата и оформление</strong>
                <p>Проверьте корзину, подтвердите адрес и завершите покупку в пару шагов.</p>
              </article>
            </div>
          </section>

          {specEntries.length > 0 ? (
            <section className={`${styles.sectionCard} page-card`}>
              <div className={styles.sectionHeader}>
                <div>
                  <span className="badge-pill">Характеристики</span>
                  <h2>Что важно знать перед заказом</h2>
                </div>
              </div>
              <dl className={styles.specList}>
                {specEntries.map(([key, value]) => (
                  <div key={key} className={styles.specRow}>
                    <dt>{formatProductSpecLabel(key)}</dt>
                    <dd>{formatProductSpecValue(value as string | number | boolean | null, key)}</dd>
                  </div>
                ))}
              </dl>
            </section>
          ) : null}

          <section className={`${styles.sectionCard} page-card`}>
            <div className={styles.sectionHeader}>
              <div>
                <span className="badge-pill">Отзывы</span>
                <h2>Отзывы и комментарии</h2>
                <p>
                  {reviewsCount > 0
                    ? `Средняя оценка ${averageRating.toFixed(1)} из 5 на основе ${formatReviewCount(reviewsCount)}.`
                    : 'Станьте первым, кто оставит отзыв об этом товаре.'}
                </p>
              </div>
            </div>

            <div className={styles.reviewsStats}>
              <article className={styles.reviewStatCard}>
                <RatingStars rating={reviewsCount > 0 ? averageRating : 0} size="md" />
                <span>Средняя оценка</span>
                <strong>{reviewsCount > 0 ? averageRating.toFixed(1) : '0.0'}</strong>
              </article>
              <article className={styles.reviewStatCard}>
                <span>Всего отзывов</span>
                <strong>{reviewsCount > 0 ? formatReviewCount(reviewsCount) : 'Пока нет'}</strong>
              </article>
            </div>

            {auth.isAuthenticated ? (
              <form className={styles.reviewForm} onSubmit={handleReviewSubmit}>
                <div className={styles.reviewFormGrid}>
                  <label className={styles.reviewField}>
                    <span>Оценка</span>
                    <select
                      value={reviewForm.rating}
                      onChange={(event) =>
                        setReviewForm((current) => ({
                          ...current,
                          rating: Number(event.target.value),
                        }))
                      }
                    >
                      <option value={5}>5 / 5</option>
                      <option value={4}>4 / 5</option>
                      <option value={3}>3 / 5</option>
                      <option value={2}>2 / 5</option>
                      <option value={1}>1 / 5</option>
                    </select>
                    <RatingStars rating={reviewForm.rating} size="md" />
                  </label>

                  <label className={`${styles.reviewField} ${styles.reviewFieldWide}`}>
                    <span>Комментарий</span>
                    <textarea
                      rows={4}
                      maxLength={1500}
                      value={reviewForm.comment}
                      onChange={(event) =>
                        setReviewForm((current) => ({
                          ...current,
                          comment: event.target.value,
                        }))
                      }
                      placeholder="Что понравилось, как товар показал себя в использовании, что стоит учесть."
                    />
                  </label>
                </div>

                <div className={styles.reviewFormActions}>
                  <button
                    type="submit"
                    className="action-primary"
                    disabled={reviewSubmitStatus === 'loading' || reviewForm.comment.trim().length < 3}
                  >
                    {reviewSubmitStatus === 'loading' ? 'Отправляем...' : 'Оставить отзыв'}
                  </button>
                  {reviewSubmitStatus === 'succeeded' ? (
                    <span className={styles.reviewSuccess}>Спасибо, отзыв опубликован.</span>
                  ) : null}
                  {reviewSubmitError ? <span className={styles.reviewError}>{reviewSubmitError}</span> : null}
                </div>
              </form>
            ) : (
              <div className={styles.reviewGuestCard}>
                <p>Войдите в аккаунт, чтобы оставить отзыв и комментарий к товару.</p>
                <button type="button" className="action-secondary" onClick={() => navigate('/login')}>
                  Войти
                </button>
              </div>
            )}

            {reviewsStatus === 'loading' ? (
              <AppLoader label="Загружаем отзывы..." />
            ) : reviewsError ? (
              <ErrorMessage message={reviewsError} />
            ) : reviews.length > 0 ? (
              <div className={styles.reviewList}>
                {reviews.map((review) => (
                  <article key={review.id} className={styles.reviewCard}>
                    <div className={styles.reviewCardHeader}>
                      <div>
                        <strong>{review.userName}</strong>
                        <span>{formatDate(review.createdAt)}</span>
                      </div>
                      <div className={styles.reviewRatingGroup}>
                        <RatingStars rating={review.rating} size="sm" />
                        <span className={styles.reviewRating}>{review.rating} / 5</span>
                      </div>
                    </div>
                    <p>{review.comment}</p>
                  </article>
                ))}
              </div>
            ) : (
              <div className={styles.emptyReviews}>
                <strong>Пока нет отзывов</strong>
                <p>Первый комментарий появится здесь сразу после публикации.</p>
              </div>
            )}
          </section>
        </div>

        <aside className={`${styles.purchasePanel} summary-card`}>
          <div className={styles.purchaseTop}>
            <span className="badge-pill">К покупке</span>
            <strong className={styles.sidebarPrice}>{formatCurrency(product.price, product.currency)}</strong>
            {savings > 0 ? <span className={styles.sidebarOldPrice}>{formatCurrency(oldPrice, product.currency)}</span> : null}
          </div>

          <div className={styles.quantityCard}>
            <span>Количество</span>
            <div className={styles.quantityControls}>
              <button
                type="button"
                onClick={() =>
                  setQuantities((current) => ({
                    ...current,
                    [product.id]: Math.max(1, safeQuantity - 1),
                  }))
                }
                disabled={safeQuantity <= 1}
              >
                -
              </button>
              <span>{safeQuantity}</span>
              <button
                type="button"
                onClick={() =>
                  setQuantities((current) => ({
                    ...current,
                    [product.id]: Math.min(product.stock || 1, safeQuantity + 1),
                  }))
                }
                disabled={!product.stock || safeQuantity >= product.stock}
              >
                +
              </button>
            </div>
          </div>

          <button
            type="button"
            className="action-primary"
            onClick={handleAddToCart}
            disabled={!product.stock || cartMutationStatus === 'loading'}
          >
            {cartMutationStatus === 'loading' ? 'Добавляем...' : 'Добавить в корзину'}
          </button>

          <button
            type="button"
            className="action-secondary"
            onClick={handleToggleFavorite}
            disabled={favoriteMutationStatus === 'loading'}
          >
            {isFavorite ? 'Убрать из избранного' : 'Сохранить в избранное'}
          </button>

          <div className={styles.purchaseMeta}>
            <div>
              <span>Адрес</span>
              <strong>{deliveryPlace?.title || 'Гостевой режим'}</strong>
            </div>
            <div>
              <span>Магазин</span>
              <strong>{sellerLabel}</strong>
            </div>
            <div>
              <span>Статус</span>
              <strong>{isAvailable ? 'Готов к оформлению' : 'Ожидает пополнения'}</strong>
            </div>
          </div>
        </aside>
      </section>

      {relatedItems.length > 0 ? (
        <section className={styles.recommendationsSection}>
          <div className={styles.sectionHeader}>
            <div>
              <span className="badge-pill">Похожие товары</span>
              <h2>Часто смотрят вместе с этой карточкой</h2>
              <p>Подборка собрана по текущей категории, чтобы проще продолжить выбор.</p>
            </div>
          </div>
          <ProductGrid products={relatedItems} />
        </section>
      ) : relatedStatus === 'loading' ? (
        <section className={`${styles.recommendationsSection} page-card`}>
          <AppLoader label="Подбираем похожие товары..." />
        </section>
      ) : null}
    </div>
  )
}
