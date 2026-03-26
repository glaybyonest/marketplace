import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { SellerNav } from '@/components/seller/SellerNav'
import { categoryService } from '@/services/categoryService'
import { sellerService } from '@/services/sellerService'
import type { ProductFilters } from '@/types/api'
import type { Category, Product } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { formatCurrency, formatUnitLabel } from '@/utils/format'
import { isGeneratedMediaSource, resolveProductImage, resolveProductImageFallback, swapImageToFallback } from '@/utils/media'

import styles from '@/pages/SellerPage.module.scss'

interface ProductFormState {
  name: string
  slug: string
  sku: string
  categoryId: string
  description: string
  price: string
  currency: string
  stockQty: string
  imageUrl: string
  imagesText: string
  brand: string
  unit: string
  specsText: string
  isActive: boolean
}

const initialFilters: ProductFilters = { page: 1, limit: 18, sort: 'new' }

const createInitialFormState = (): ProductFormState => ({
  name: '',
  slug: '',
  sku: '',
  categoryId: '',
  description: '',
  price: '',
  currency: 'RUB',
  stockQty: '0',
  imageUrl: '',
  imagesText: '',
  brand: '',
  unit: '',
  specsText: '{}',
  isActive: true,
})

const productToFormState = (product: Product): ProductFormState => ({
  name: product.name ?? product.title,
  slug: product.slug ?? '',
  sku: product.sku ?? '',
  categoryId: product.categoryId,
  description: product.description,
  price: String(product.price),
  currency: product.currency ?? 'RUB',
  stockQty: String(product.stock ?? 0),
  imageUrl: isGeneratedMediaSource(product.imageUrl) ? '' : (product.imageUrl ?? ''),
  imagesText: product.images.filter((image) => !isGeneratedMediaSource(image)).join('\n'),
  brand: product.brand ?? '',
  unit: product.unit ?? '',
  specsText: JSON.stringify(product.specs ?? {}, null, 2),
  isActive: product.isActive ?? true,
})

export const SellerProductsPage = () => {
  const [categories, setCategories] = useState<Category[]>([])
  const [items, setItems] = useState<Product[]>([])
  const [filters, setFilters] = useState<ProductFilters>(initialFilters)
  const [searchDraft, setSearchDraft] = useState('')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editingProduct, setEditingProduct] = useState<Product | null>(null)
  const [formState, setFormState] = useState<ProductFormState>(createInitialFormState())
  const [stockDrafts, setStockDrafts] = useState<Record<string, string>>({})
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [total, setTotal] = useState(0)
  const initializedRef = useRef(false)

  const categoryMap = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories])

  const loadProducts = useCallback(async (nextFilters: ProductFilters) => {
    setLoading(true)
    setError(null)
    try {
      const response = await sellerService.getProducts(nextFilters)
      setItems(response.items)
      setPage(response.page)
      setTotalPages(response.totalPages)
      setTotal(response.total)
      setStockDrafts(Object.fromEntries(response.items.map((product) => [product.id, String(product.stock ?? 0)])))
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Не удалось загрузить товары магазина'))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadCategories = useCallback(async () => {
    try {
      const data = await categoryService.getCategories()
      setCategories(data)
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Не удалось загрузить категории'))
    }
  }, [])

  useEffect(() => {
    const loadInitial = async () => {
      await Promise.all([loadCategories(), loadProducts(initialFilters)])
    }

    void loadInitial()
  }, [loadCategories, loadProducts])

  useEffect(() => {
    if (!initializedRef.current) {
      initializedRef.current = true
      return
    }
    void loadProducts(filters)
  }, [filters, loadProducts])

  const resetForm = () => {
    setEditingProduct(null)
    setFormState(createInitialFormState())
  }

  const handleEdit = (product: Product) => {
    setEditingProduct(product)
    setFormState(productToFormState(product))
  }

  const buildPayloadFromForm = () => {
    const parsedPrice = Number(formState.price)
    const parsedStock = Number(formState.stockQty)
    const trimmedSpecs = formState.specsText.trim()
    const parsedSpecs = trimmedSpecs.length > 0 ? JSON.parse(trimmedSpecs) : {}

    return {
      name: formState.name,
      slug: formState.slug || undefined,
      description: formState.description,
      price: parsedPrice,
      categoryId: formState.categoryId,
      currency: formState.currency,
      sku: formState.sku,
      imageUrl: formState.imageUrl || undefined,
      images: formState.imagesText
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean),
      brand: formState.brand || undefined,
      unit: formState.unit || undefined,
      specs: parsedSpecs,
      stock: parsedStock,
      isActive: formState.isActive,
    }
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      const payload = buildPayloadFromForm()

      if (editingProduct) {
        await sellerService.updateProduct(editingProduct.id, payload)
      } else {
        await sellerService.createProduct(payload)
      }

      await loadProducts(filters)
      resetForm()
    } catch (submitError) {
      setError(getErrorMessage(submitError, 'Не удалось сохранить товар'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleStockSave = async (product: Product) => {
    const nextStock = Number(stockDrafts[product.id] ?? product.stock ?? 0)
    setSubmitting(true)
    setError(null)
    try {
      await sellerService.updateProductStock(product.id, nextStock)
      await loadProducts(filters)
    } catch (stockError) {
      setError(getErrorMessage(stockError, 'Не удалось обновить остаток'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleToggleActive = async (product: Product) => {
    setSubmitting(true)
    setError(null)
    try {
      await sellerService.updateProduct(product.id, {
        name: product.name ?? product.title,
        slug: product.slug,
        description: product.description,
        price: product.price,
        categoryId: product.categoryId,
        currency: product.currency,
        sku: product.sku,
        imageUrl: product.imageUrl,
        images: product.images,
        brand: product.brand,
        unit: product.unit,
        specs: product.specs,
        stock: product.stock,
        isActive: !(product.isActive ?? true),
      })
      await loadProducts(filters)
    } catch (toggleError) {
      setError(getErrorMessage(toggleError, 'Не удалось обновить статус товара'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleArchive = async (product: Product) => {
    if (!window.confirm(`Скрыть товар «${product.title}» из витрины магазина?`)) {
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await sellerService.deleteProduct(product.id)
      await loadProducts(filters)
      if (editingProduct?.id === product.id) {
        resetForm()
      }
    } catch (deleteError) {
      setError(getErrorMessage(deleteError, 'Не удалось скрыть товар'))
    } finally {
      setSubmitting(false)
    }
  }

  const applyFilters = (nextFilters: Partial<ProductFilters>) => {
    setFilters((current) => ({
      ...current,
      ...nextFilters,
      page: nextFilters.page ?? 1,
    }))
  }

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroTop}>
          <div className={styles.heroCopy}>
            <span className={styles.heroBadge}>Товары магазина</span>
            <h1>Управляйте ассортиментом продавца</h1>
            <p>Цена, остатки, изображения, характеристики и видимость карточек обновляются из одного раздела.</p>
          </div>
          <SellerNav />
        </div>

        <div className={styles.heroMeta}>
          <div className={styles.heroMetaCard}>
            <span>Всего товаров</span>
            <strong>{total}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>В продаже</span>
            <strong>{items.filter((item) => item.isActive).length}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Требуют внимания</span>
            <strong>{items.filter((item) => (item.stock ?? 0) <= 10).length}</strong>
          </div>
        </div>
      </section>

      {loading ? <AppLoader label="Загружаем товары магазина..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <div className={styles.contentGrid}>
        <section className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">{editingProduct ? 'Редактирование' : 'Новая карточка'}</span>
              <h2>{editingProduct ? 'Обновите карточку товара' : 'Добавьте новый товар'}</h2>
              <p>Новая карточка сразу попадёт в каталог вашего магазина и начнёт работать на витрине.</p>
            </div>
          </div>

          <form className={styles.form} onSubmit={handleSubmit}>
            <div className={styles.formGrid}>
              <label className={styles.field}>
                Название товара
                <input
                  value={formState.name}
                  onChange={(event) => setFormState((current) => ({ ...current, name: event.target.value }))}
                  required
                />
              </label>
              <label className={styles.field}>
                Slug
                <input
                  value={formState.slug}
                  onChange={(event) => setFormState((current) => ({ ...current, slug: event.target.value }))}
                />
              </label>
              <label className={styles.field}>
                Артикул
                <input
                  value={formState.sku}
                  onChange={(event) => setFormState((current) => ({ ...current, sku: event.target.value }))}
                  required
                />
              </label>
              <label className={styles.field}>
                Категория
                <select
                  value={formState.categoryId}
                  onChange={(event) => setFormState((current) => ({ ...current, categoryId: event.target.value }))}
                  required
                >
                  <option value="">Выберите категорию</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.field}>
                Цена
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={formState.price}
                  onChange={(event) => setFormState((current) => ({ ...current, price: event.target.value }))}
                  required
                />
              </label>
              <label className={styles.field}>
                Валюта
                <input
                  value={formState.currency}
                  onChange={(event) => setFormState((current) => ({ ...current, currency: event.target.value.toUpperCase() }))}
                  maxLength={3}
                />
              </label>
              <label className={styles.field}>
                Остаток
                <input
                  type="number"
                  min="0"
                  value={formState.stockQty}
                  onChange={(event) => setFormState((current) => ({ ...current, stockQty: event.target.value }))}
                  required
                />
              </label>
              <label className={styles.field}>
                Единица продажи
                <input
                  value={formState.unit}
                  onChange={(event) => setFormState((current) => ({ ...current, unit: event.target.value }))}
                  placeholder="шт. / набор / кг"
                />
              </label>
              <label className={styles.field}>
                Бренд
                <input
                  value={formState.brand}
                  onChange={(event) => setFormState((current) => ({ ...current, brand: event.target.value }))}
                />
              </label>
              <label className={styles.field}>
                URL обложки
                  <input
                    value={formState.imageUrl}
                    onChange={(event) => setFormState((current) => ({ ...current, imageUrl: event.target.value }))}
                    placeholder="Оставьте пустым для локальной обложки"
                  />
                </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                Описание
                <textarea
                  value={formState.description}
                  onChange={(event) => setFormState((current) => ({ ...current, description: event.target.value }))}
                />
              </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                Галерея изображений
                <textarea
                  value={formState.imagesText}
                  onChange={(event) => setFormState((current) => ({ ...current, imagesText: event.target.value }))}
                  placeholder="По одному URL в строке"
                />
              </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                JSON характеристик
                <textarea
                  value={formState.specsText}
                  onChange={(event) => setFormState((current) => ({ ...current, specsText: event.target.value }))}
                />
              </label>
            </div>

            <p className={styles.helper}>Если вы не добавляете свои фото, маркетплейс покажет аккуратную локальную обложку для карточки.</p>

            <label className={styles.field}>
              <span>Видимость на витрине</span>
              <select
                value={formState.isActive ? 'visible' : 'hidden'}
                onChange={(event) => setFormState((current) => ({ ...current, isActive: event.target.value === 'visible' }))}
              >
                <option value="visible">Показывать в магазине</option>
                <option value="hidden">Скрыть с витрины</option>
              </select>
            </label>

            <div className={styles.inlineActions}>
              <button type="submit" className="action-primary" disabled={submitting}>
                {submitting ? 'Сохраняем...' : editingProduct ? 'Обновить товар' : 'Создать товар'}
              </button>
              <button type="button" className="action-secondary" onClick={resetForm} disabled={submitting}>
                Сбросить форму
              </button>
            </div>
          </form>
        </section>

        <section className={styles.panel}>
          <div className={styles.toolbar}>
            <div>
              <span className="badge-pill">Каталог магазина</span>
              <h2>Текущие товары</h2>
              <p>Найдено: {total}</p>
            </div>
            <form
              className={styles.toolbarFilters}
              onSubmit={(event) => {
                event.preventDefault()
                applyFilters({ q: searchDraft || undefined })
              }}
            >
              <input
                value={searchDraft}
                onChange={(event) => setSearchDraft(event.target.value)}
                placeholder="Поиск по названию, артикулу или бренду"
              />
              <select value={filters.category_id ?? ''} onChange={(event) => applyFilters({ category_id: event.target.value || undefined })}>
                <option value="">Все категории</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
              <select
                value={filters.is_active === undefined ? 'all' : filters.is_active ? 'visible' : 'hidden'}
                onChange={(event) => {
                  const value = event.target.value
                  applyFilters({ is_active: value === 'all' ? undefined : value === 'visible' })
                }}
              >
                <option value="all">Все статусы</option>
                <option value="visible">В продаже</option>
                <option value="hidden">Скрытые</option>
              </select>
              <button type="submit" className="action-secondary">
                Применить
              </button>
            </form>
          </div>

          <div className={styles.list}>
            {items.length === 0 && !loading ? (
              <div className="empty-state">
                <h2>Товары не найдены</h2>
                <p>Добавьте первую карточку или ослабьте фильтры каталога.</p>
              </div>
            ) : (
              items.map((product) => (
                <article key={product.id} className={styles.listCard}>
                  <img
                    className={styles.mediaThumb}
                    src={resolveProductImage(product)}
                    alt={product.title}
                    onError={(event) => swapImageToFallback(event.currentTarget, resolveProductImageFallback(product))}
                  />
                  <div className={styles.listHeader}>
                    <div>
                      <h3>{product.title}</h3>
                      <p className={styles.listMeta}>
                        {categoryMap.get(product.categoryId) ?? 'Категория'} • Артикул {product.sku ?? '-'} •{' '}
                        {formatCurrency(product.price, product.currency)}
                        {product.unit ? ` • ${formatUnitLabel(product.unit)}` : ''}
                      </p>
                    </div>
                    <div className={styles.badgeRow}>
                      <span className={product.isActive ? styles.badge : styles.badgeDanger}>
                        {product.isActive ? 'В продаже' : 'Скрыт'}
                      </span>
                      <span className={(product.stock ?? 0) <= 10 ? styles.badgeWarn : styles.badgeMuted}>
                        Остаток: {product.stock ?? 0}
                      </span>
                    </div>
                  </div>

                  {product.description ? <p className={styles.listMeta}>{product.description}</p> : null}

                  <div className={styles.inlineActions}>
                    <button type="button" className={styles.inlineButton} onClick={() => handleEdit(product)}>
                      Изменить
                    </button>
                    <button type="button" className={styles.inlineButton} onClick={() => handleToggleActive(product)} disabled={submitting}>
                      {product.isActive ? 'Скрыть' : 'Вернуть'}
                    </button>
                    <button
                      type="button"
                      className={`${styles.inlineButton} ${styles.inlineButtonDanger}`}
                      onClick={() => handleArchive(product)}
                      disabled={submitting}
                    >
                      Убрать с витрины
                    </button>
                  </div>

                  <form
                    className={styles.toolbarFilters}
                    onSubmit={(event) => {
                      event.preventDefault()
                      void handleStockSave(product)
                    }}
                  >
                    <input
                      type="number"
                      min="0"
                      value={stockDrafts[product.id] ?? String(product.stock ?? 0)}
                      onChange={(event) =>
                        setStockDrafts((current) => ({
                          ...current,
                          [product.id]: event.target.value,
                        }))
                      }
                    />
                    <button type="submit" className="action-secondary" disabled={submitting}>
                      Сохранить остаток
                    </button>
                  </form>
                </article>
              ))
            )}
          </div>

          {totalPages > 1 ? (
            <div className={styles.pagination}>
              <button type="button" className="action-secondary" disabled={page <= 1} onClick={() => applyFilters({ page: page - 1 })}>
                Назад
              </button>
              <span className={styles.helper}>
                Страница {page} из {totalPages}
              </span>
              <button
                type="button"
                className="action-secondary"
                disabled={page >= totalPages}
                onClick={() => applyFilters({ page: page + 1 })}
              >
                Дальше
              </button>
            </div>
          ) : null}
        </section>
      </div>
    </div>
  )
}
