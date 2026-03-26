import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { AdminNav } from '@/components/admin/AdminNav'
import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { categoryService } from '@/services/categoryService'
import { productService } from '@/services/productService'
import type { ProductFilters } from '@/types/api'
import type { Category, Product } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { formatCurrency, formatUnitLabel } from '@/utils/format'
import { isGeneratedMediaSource, resolveProductImage, resolveProductImageFallback, swapImageToFallback } from '@/utils/media'

import styles from '@/pages/AdminPage.module.scss'

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

const initialFilters: ProductFilters = { page: 1, limit: 24, sort: 'new' }

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

const productToPayload = (product: Product, overrides?: Partial<{ stock: number; isActive: boolean }>) => ({
  name: product.name ?? product.title,
  slug: product.slug,
  description: product.description,
  price: product.price,
  categoryId: product.categoryId,
  currency: product.currency ?? 'RUB',
  sku: product.sku ?? '',
  imageUrl: product.imageUrl,
  images: product.images,
  brand: product.brand,
  unit: product.unit,
  specs: product.specs ?? {},
  stock: overrides?.stock ?? product.stock ?? 0,
  isActive: overrides?.isActive ?? product.isActive ?? true,
})

export const AdminProductsPage = () => {
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

  const categoryMap = useMemo(
    () => new Map(categories.map((category) => [category.id, category.name])),
    [categories],
  )

  const loadProducts = useCallback(async (nextFilters: ProductFilters) => {
    setLoading(true)
    setError(null)
    try {
      const response = await productService.getAdminProducts(nextFilters)
      setItems(response.items)
      setPage(response.page)
      setTotalPages(response.totalPages)
      setTotal(response.total)
      setStockDrafts(
        Object.fromEntries(
          response.items.map((product) => [product.id, String(product.stock ?? 0)]),
        ),
      )
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Не удалось загрузить товары'))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadCategories = useCallback(async () => {
    try {
      const nextCategories = await categoryService.getCategories()
      setCategories(nextCategories)
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
        await productService.updateProduct(editingProduct.id, payload)
      } else {
        await productService.createProduct(payload)
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
      await productService.updateProductStock(product.id, nextStock)
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
      await productService.updateProduct(
        product.id,
        productToPayload(product, { isActive: !(product.isActive ?? true) }),
      )
      await loadProducts(filters)
      if (editingProduct?.id === product.id) {
        setEditingProduct({
          ...product,
          isActive: !(product.isActive ?? true),
        })
        setFormState((current) => ({ ...current, isActive: !(product.isActive ?? true) }))
      }
    } catch (toggleError) {
      setError(getErrorMessage(toggleError, 'Не удалось обновить статус товара'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleArchive = async (product: Product) => {
    if (!window.confirm(`Отправить товар «${product.title}» в архив?`)) {
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await productService.deleteProduct(product.id)
      await loadProducts(filters)
      if (editingProduct?.id === product.id) {
        resetForm()
      }
    } catch (deleteError) {
      setError(getErrorMessage(deleteError, 'Не удалось архивировать товар'))
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
        <div>
          <span className="badge-pill">Товары</span>
          <h1>Карточки каталога и остатки</h1>
          <p>Управляйте контентом карточек, характеристиками, ценами, медиа, остатками и видимостью товаров на публичной витрине.</p>
        </div>
        <AdminNav />
      </section>

      {loading ? <AppLoader label="Загружаем товары..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.contentGrid}>
        <article className={styles.panel}>
          <h2>{editingProduct ? 'Редактирование товара' : 'Новый товар'}</h2>
          <p>Заполните категорию, цену, остаток, бренд, единицу продажи, медиа и характеристики товара в одном месте.</p>

          <form className={styles.form} onSubmit={handleSubmit}>
            <div className={styles.fieldGrid}>
              <label>
                Название
                <input
                  value={formState.name}
                  onChange={(event) => setFormState((current) => ({ ...current, name: event.target.value }))}
                  required
                />
              </label>
              <label>
                Slug
                <input
                  value={formState.slug}
                  onChange={(event) => setFormState((current) => ({ ...current, slug: event.target.value }))}
                  placeholder="Необязательно"
                />
              </label>
              <label>
                Артикул
                <input
                  value={formState.sku}
                  onChange={(event) => setFormState((current) => ({ ...current, sku: event.target.value }))}
                  required
                />
              </label>
              <label>
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
              <label>
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
              <label>
                Валюта
                <input
                  value={formState.currency}
                  onChange={(event) => setFormState((current) => ({ ...current, currency: event.target.value.toUpperCase() }))}
                  maxLength={3}
                />
              </label>
              <label>
                Остаток
                <input
                  type="number"
                  min="0"
                  value={formState.stockQty}
                  onChange={(event) => setFormState((current) => ({ ...current, stockQty: event.target.value }))}
                  required
                />
              </label>
              <label>
                Единица продажи
                <input
                  value={formState.unit}
                  onChange={(event) => setFormState((current) => ({ ...current, unit: event.target.value }))}
                  placeholder="шт. / уп. / кг"
                />
              </label>
              <label>
                Бренд
                <input
                  value={formState.brand}
                  onChange={(event) => setFormState((current) => ({ ...current, brand: event.target.value }))}
                />
              </label>
              <label>
                URL обложки
                  <input
                    value={formState.imageUrl}
                    onChange={(event) => setFormState((current) => ({ ...current, imageUrl: event.target.value }))}
                    placeholder="Оставьте пустым для локальной обложки"
                  />
                </label>
            </div>

            <p className={styles.helperText}>Если изображение не указано, витрина создаст локальную обложку без внешних сервисов.</p>

            <label>
              Описание
              <textarea
                value={formState.description}
                onChange={(event) => setFormState((current) => ({ ...current, description: event.target.value }))}
              />
            </label>

            <label>
              Галерея изображений
              <textarea
                value={formState.imagesText}
                onChange={(event) => setFormState((current) => ({ ...current, imagesText: event.target.value }))}
                placeholder="По одному URL в строке"
              />
            </label>

            <label>
              JSON характеристик
              <textarea
                value={formState.specsText}
                onChange={(event) => setFormState((current) => ({ ...current, specsText: event.target.value }))}
              />
            </label>

            <label className={styles.checkboxRow}>
              <input
                type="checkbox"
                checked={formState.isActive}
                onChange={(event) => setFormState((current) => ({ ...current, isActive: event.target.checked }))}
              />
              Показывать товар на публичной витрине
            </label>

            <div className={styles.formActions}>
              <button type="submit" className={styles.primaryButton} disabled={submitting}>
                {submitting ? 'Сохраняем...' : editingProduct ? 'Обновить товар' : 'Создать товар'}
              </button>
              <button type="button" className={styles.secondaryButton} onClick={resetForm} disabled={submitting}>
                Сбросить форму
              </button>
            </div>
          </form>
        </article>

        <article className={styles.panel}>
          <div className={styles.toolbar}>
            <div>
              <h2>Каталог товаров</h2>
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
                placeholder="Поиск по названию, бренду, артикулу..."
              />
              <select
                value={filters.category_id ?? ''}
                onChange={(event) => applyFilters({ category_id: event.target.value || undefined })}
              >
                <option value="">Все категории</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
              <select
                value={
                  filters.is_active === undefined
                    ? 'all'
                    : filters.is_active
                      ? 'active'
                      : 'hidden'
                }
                onChange={(event) => {
                  const value = event.target.value
                  applyFilters({
                    is_active: value === 'all' ? undefined : value === 'active',
                  })
                }}
              >
                <option value="all">Все статусы</option>
                <option value="active">Активные</option>
                <option value="hidden">Скрытые</option>
              </select>
              <button type="submit" className={styles.primaryButton}>
                Применить
              </button>
            </form>
          </div>

          {items.length === 0 && !loading ? (
            <div className={styles.empty}>
              <h2>Товары не найдены</h2>
              <p>Создайте товар или ослабьте текущие фильтры.</p>
            </div>
          ) : (
            <div className={styles.list}>
              {items.map((product) => (
                <article key={product.id} className={styles.listCard}>
                  <div className={styles.productCardTop}>
                    <img
                      className={styles.imageThumb}
                      src={resolveProductImage(product)}
                      alt={product.title}
                      onError={(event) => swapImageToFallback(event.currentTarget, resolveProductImageFallback(product))}
                    />
                    <div className={styles.stack}>
                      <div className={styles.listHeader}>
                        <div>
                          <h3>{product.title}</h3>
                          <p className={styles.listMeta}>
                            {categoryMap.get(product.categoryId) ?? 'Неизвестная категория'} • Артикул {product.sku ?? '-'}
                          </p>
                        </div>
                        <div className={styles.badgeRow}>
                          <span className={product.isActive ? styles.badge : styles.badgeDanger}>
                            {product.isActive ? 'Активен' : 'Скрыт'}
                          </span>
                          <span className={styles.badgeMuted}>Остаток: {product.stock ?? 0}</span>
                        </div>
                      </div>

                      <div className={styles.productSummary}>
                        <p className={styles.muted}>
                          {product.brand ? `${product.brand} • ` : ''}
                          {product.unit ? `${formatUnitLabel(product.unit)} • ` : ''}
                          {formatCurrency(product.price, product.currency ?? 'RUB')}
                        </p>
                        <div className={styles.rowActions}>
                          <button type="button" className={styles.ghostButton} onClick={() => handleEdit(product)}>
                            Изменить
                          </button>
                          <button
                            type="button"
                            className={styles.secondaryButton}
                            onClick={() => handleToggleActive(product)}
                            disabled={submitting}
                          >
                            {product.isActive ? 'Скрыть' : 'Вернуть'}
                          </button>
                          <button
                            type="button"
                            className={styles.dangerButton}
                            onClick={() => handleArchive(product)}
                            disabled={submitting}
                          >
                            В архив
                          </button>
                        </div>
                      </div>

                      {product.description ? <p className={styles.muted}>{product.description}</p> : null}

                      {product.specs && Object.keys(product.specs).length > 0 ? (
                        <div className={styles.specList}>
                          {Object.entries(product.specs)
                            .slice(0, 4)
                            .map(([key, value]) => (
                              <span key={key} className={styles.specItem}>
                                {key}: {String(value)}
                              </span>
                            ))}
                        </div>
                      ) : null}

                      <form
                        className={styles.inlineForm}
                        onSubmit={(event) => {
                          event.preventDefault()
                          void handleStockSave(product)
                        }}
                      >
                        <label>
                          Остаток
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
                        </label>
                        <button type="submit" className={styles.primaryButton} disabled={submitting}>
                          Сохранить остаток
                        </button>
                      </form>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          )}

          {totalPages > 1 ? (
            <div className={styles.formActions}>
              <button
                type="button"
                className={styles.secondaryButton}
                disabled={page <= 1}
                onClick={() => applyFilters({ page: page - 1 })}
              >
                Назад
              </button>
              <span className={styles.helperText}>
                Страница {page} из {totalPages}
              </span>
              <button
                type="button"
                className={styles.secondaryButton}
                disabled={page >= totalPages}
                onClick={() => applyFilters({ page: page + 1 })}
              >
                Дальше
              </button>
            </div>
          ) : null}
        </article>
      </section>
    </div>
  )
}
