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
import { formatCurrency } from '@/utils/format'

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
  imageUrl: product.imageUrl ?? '',
  imagesText: product.images.join('\n'),
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
      setError(getErrorMessage(loadError, 'Failed to load products'))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadCategories = useCallback(async () => {
    try {
      const nextCategories = await categoryService.getCategories()
      setCategories(nextCategories)
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Failed to load categories'))
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
      setError(getErrorMessage(submitError, 'Failed to save product'))
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
      setError(getErrorMessage(stockError, 'Failed to update stock'))
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
      setError(getErrorMessage(toggleError, 'Failed to update product status'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleArchive = async (product: Product) => {
    if (!window.confirm(`Archive product "${product.title}"?`)) {
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
      setError(getErrorMessage(deleteError, 'Failed to archive product'))
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
          <h1>Products</h1>
          <p>Manage catalog cards, stock, and visibility for customer-facing products.</p>
        </div>
        <AdminNav />
      </section>

      {loading ? <AppLoader label="Loading products..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.contentGrid}>
        <article className={styles.panel}>
          <h2>{editingProduct ? 'Edit product' : 'New product'}</h2>
          <p>Fields match backend product model: category, stock, media, brand, unit, and specs JSON.</p>

          <form className={styles.form} onSubmit={handleSubmit}>
            <div className={styles.fieldGrid}>
              <label>
                Name
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
                  placeholder="Optional"
                />
              </label>
              <label>
                SKU
                <input
                  value={formState.sku}
                  onChange={(event) => setFormState((current) => ({ ...current, sku: event.target.value }))}
                  required
                />
              </label>
              <label>
                Category
                <select
                  value={formState.categoryId}
                  onChange={(event) => setFormState((current) => ({ ...current, categoryId: event.target.value }))}
                  required
                >
                  <option value="">Select category</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Price
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
                Currency
                <input
                  value={formState.currency}
                  onChange={(event) => setFormState((current) => ({ ...current, currency: event.target.value.toUpperCase() }))}
                  maxLength={3}
                />
              </label>
              <label>
                Stock
                <input
                  type="number"
                  min="0"
                  value={formState.stockQty}
                  onChange={(event) => setFormState((current) => ({ ...current, stockQty: event.target.value }))}
                  required
                />
              </label>
              <label>
                Unit
                <input
                  value={formState.unit}
                  onChange={(event) => setFormState((current) => ({ ...current, unit: event.target.value }))}
                  placeholder="bag / piece / sheet"
                />
              </label>
              <label>
                Brand
                <input
                  value={formState.brand}
                  onChange={(event) => setFormState((current) => ({ ...current, brand: event.target.value }))}
                />
              </label>
              <label>
                Cover image URL
                <input
                  value={formState.imageUrl}
                  onChange={(event) => setFormState((current) => ({ ...current, imageUrl: event.target.value }))}
                  placeholder="https://..."
                />
              </label>
            </div>

            <label>
              Description
              <textarea
                value={formState.description}
                onChange={(event) => setFormState((current) => ({ ...current, description: event.target.value }))}
              />
            </label>

            <label>
              Gallery URLs
              <textarea
                value={formState.imagesText}
                onChange={(event) => setFormState((current) => ({ ...current, imagesText: event.target.value }))}
                placeholder={'One URL per line'}
              />
            </label>

            <label>
              Specs JSON
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
              Visible in public catalog
            </label>

            <div className={styles.formActions}>
              <button type="submit" className={styles.primaryButton} disabled={submitting}>
                {submitting ? 'Saving...' : editingProduct ? 'Update product' : 'Create product'}
              </button>
              <button type="button" className={styles.secondaryButton} onClick={resetForm} disabled={submitting}>
                Reset
              </button>
            </div>
          </form>
        </article>

        <article className={styles.panel}>
          <div className={styles.toolbar}>
            <div>
              <h2>Catalog inventory</h2>
              <p>{total} products found.</p>
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
                placeholder="Search by name, brand, SKU..."
              />
              <select
                value={filters.category_id ?? ''}
                onChange={(event) => applyFilters({ category_id: event.target.value || undefined })}
              >
                <option value="">All categories</option>
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
                <option value="all">All statuses</option>
                <option value="active">Active</option>
                <option value="hidden">Hidden</option>
              </select>
              <button type="submit" className={styles.primaryButton}>
                Apply
              </button>
            </form>
          </div>

          {items.length === 0 && !loading ? (
            <div className={styles.empty}>
              <h2>No products yet</h2>
              <p>Create a product or relax the current filters.</p>
            </div>
          ) : (
            <div className={styles.list}>
              {items.map((product) => (
                <article key={product.id} className={styles.listCard}>
                  <div className={styles.productCardTop}>
                    <img
                      className={styles.imageThumb}
                      src={product.imageUrl || 'https://placehold.co/160x160/e5e7eb/6b7280?text=No+Image'}
                      alt={product.title}
                    />
                    <div className={styles.stack}>
                      <div className={styles.listHeader}>
                        <div>
                          <h3>{product.title}</h3>
                          <p className={styles.listMeta}>
                            {categoryMap.get(product.categoryId) ?? 'Unknown category'} · SKU {product.sku ?? '-'}
                          </p>
                        </div>
                        <div className={styles.badgeRow}>
                          <span className={product.isActive ? styles.badge : styles.badgeDanger}>
                            {product.isActive ? 'Active' : 'Hidden'}
                          </span>
                          <span className={styles.badgeMuted}>Stock: {product.stock ?? 0}</span>
                        </div>
                      </div>

                      <div className={styles.productSummary}>
                        <p className={styles.muted}>
                          {product.brand ? `${product.brand} · ` : ''}
                          {product.unit ? `${product.unit} · ` : ''}
                          {formatCurrency(product.price, product.currency ?? 'RUB')}
                        </p>
                        <div className={styles.rowActions}>
                          <button type="button" className={styles.ghostButton} onClick={() => handleEdit(product)}>
                            Edit
                          </button>
                          <button
                            type="button"
                            className={styles.secondaryButton}
                            onClick={() => handleToggleActive(product)}
                            disabled={submitting}
                          >
                            {product.isActive ? 'Hide' : 'Restore'}
                          </button>
                          <button
                            type="button"
                            className={styles.dangerButton}
                            onClick={() => handleArchive(product)}
                            disabled={submitting}
                          >
                            Archive
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
                          Stock
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
                          Save stock
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
                Prev
              </button>
              <span className={styles.helperText}>
                Page {page} of {totalPages}
              </span>
              <button
                type="button"
                className={styles.secondaryButton}
                disabled={page >= totalPages}
                onClick={() => applyFilters({ page: page + 1 })}
              >
                Next
              </button>
            </div>
          ) : null}
        </article>
      </section>
    </div>
  )
}
