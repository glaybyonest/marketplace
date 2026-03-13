import { useState } from 'react'

import type { Category } from '@/types/domain'

import styles from '@/components/catalog/FilterPanel.module.scss'

export interface FilterValues {
  category_id?: string
  min_price?: number
  max_price?: number
  in_stock?: boolean
  sort?: 'new' | 'price_asc' | 'price_desc'
}

interface FilterPanelProps {
  categories: Category[]
  values: FilterValues
  onChange: (values: FilterValues) => void
  onReset: () => void
  mobileOpen?: boolean
  onCloseMobile?: () => void
}

export const FilterPanel = ({
  categories,
  values,
  onChange,
  onReset,
  mobileOpen = false,
  onCloseMobile,
}: FilterPanelProps) => {
  const handleChange = (next: Partial<FilterValues>) => onChange({ ...values, ...next })

  return (
    <>
      <div className={styles.desktop}>
        <Content
          categories={categories}
          values={values}
          onChange={handleChange}
          onReset={onReset}
          onCloseMobile={onCloseMobile}
        />
      </div>

      {mobileOpen ? (
        <div className={styles.mobileOverlay}>
          <div className={styles.mobile}>
            <Content
              categories={categories}
              values={values}
              onChange={handleChange}
              onReset={onReset}
              onCloseMobile={onCloseMobile}
            />
          </div>
        </div>
      ) : null}
    </>
  )
}

interface ContentProps {
  categories: Category[]
  values: FilterValues
  onChange: (next: Partial<FilterValues>) => void
  onReset: () => void
  onCloseMobile?: () => void
}

const Content = ({ categories, values, onChange, onReset, onCloseMobile }: ContentProps) => {
  const [categoryOpen, setCategoryOpen] = useState(true)
  const [priceOpen, setPriceOpen] = useState(true)
  const [stockOpen, setStockOpen] = useState(true)
  const [sortOpen, setSortOpen] = useState(true)

  const hasActiveFilters =
    values.category_id ||
    typeof values.min_price === 'number' ||
    typeof values.max_price === 'number' ||
    Boolean(values.in_stock) ||
    (values.sort && values.sort !== 'new')

  const handleCategoryClick = (categoryId: string) => {
    if (values.category_id === categoryId) {
      onChange({ category_id: undefined })
    } else {
      onChange({ category_id: categoryId })
    }
  }

  const handleNumberChange = (field: 'min_price' | 'max_price', rawValue: string) => {
    const trimmed = rawValue.trim()
    if (trimmed === '') {
      onChange({ [field]: undefined })
      return
    }

    const parsed = Number(trimmed)
    if (Number.isNaN(parsed)) {
      return
    }

    onChange({ [field]: parsed })
  }

  return (
    <section className={styles.panel} aria-label="Catalog filters">
      <div className={styles.header}>
        <h2>Filters</h2>
        {onCloseMobile ? (
          <button className={styles.close} type="button" onClick={onCloseMobile}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
            Close
          </button>
        ) : null}
      </div>

      <div className={styles.accordion}>
        <div className={styles.accordionItem}>
          <button
            type="button"
            className={styles.accordionHeader}
            onClick={() => setCategoryOpen(!categoryOpen)}
            aria-expanded={categoryOpen}
          >
            <span>Categories</span>
            <svg className={`${styles.accordionIcon} ${categoryOpen ? styles.accordionIconOpen : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </button>
          <div className={`${styles.accordionContent} ${categoryOpen ? styles.accordionContentOpen : ''}`}>
            <div className={styles.categoryList}>
              <button
                type="button"
                className={`${styles.categoryItem} ${!values.category_id ? styles.categoryItemActive : ''}`}
                onClick={() => onChange({ category_id: undefined })}
              >
                All categories
              </button>
              {categories.map((category) => (
                <button
                  key={category.id}
                  type="button"
                  className={`${styles.categoryItem} ${values.category_id === category.id ? styles.categoryItemActive : ''}`}
                  onClick={() => handleCategoryClick(category.id)}
                >
                  {category.name}
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className={styles.accordion}>
        <div className={styles.accordionItem}>
          <button
            type="button"
            className={styles.accordionHeader}
            onClick={() => setPriceOpen(!priceOpen)}
            aria-expanded={priceOpen}
          >
            <span>Price</span>
            <svg className={`${styles.accordionIcon} ${priceOpen ? styles.accordionIconOpen : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </button>
          <div className={`${styles.accordionContent} ${priceOpen ? styles.accordionContentOpen : ''}`}>
            <div className={styles.row}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Min price</span>
                <input
                  className={styles.priceInput}
                  type="number"
                  min="0"
                  step="1"
                  value={values.min_price ?? ''}
                  onChange={(event) => handleNumberChange('min_price', event.target.value)}
                  placeholder="0"
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Max price</span>
                <input
                  className={styles.priceInput}
                  type="number"
                  min="0"
                  step="1"
                  value={values.max_price ?? ''}
                  onChange={(event) => handleNumberChange('max_price', event.target.value)}
                  placeholder="5000"
                />
              </label>
            </div>
          </div>
        </div>
      </div>

      <div className={styles.accordion}>
        <div className={styles.accordionItem}>
          <button
            type="button"
            className={styles.accordionHeader}
            onClick={() => setStockOpen(!stockOpen)}
            aria-expanded={stockOpen}
          >
            <span>Availability</span>
            <svg className={`${styles.accordionIcon} ${stockOpen ? styles.accordionIconOpen : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </button>
          <div className={`${styles.accordionContent} ${stockOpen ? styles.accordionContentOpen : ''}`}>
            <div className={styles.radioGroup}>
              <label className={`${styles.radioLabel} ${values.in_stock === true ? styles.radioLabelActive : ''}`}>
                <input
                  type="checkbox"
                  className={styles.radioInput}
                  checked={values.in_stock === true}
                  onChange={(event) => onChange({ in_stock: event.target.checked ? true : undefined })}
                />
                <span className={styles.radioCustom}><span className={styles.radioDot} /></span>
                <span>Only in stock</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      <div className={styles.accordion}>
        <div className={styles.accordionItem}>
          <button
            type="button"
            className={styles.accordionHeader}
            onClick={() => setSortOpen(!sortOpen)}
            aria-expanded={sortOpen}
          >
            <span>Sorting</span>
            <svg className={`${styles.accordionIcon} ${sortOpen ? styles.accordionIconOpen : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </button>
          <div className={`${styles.accordionContent} ${sortOpen ? styles.accordionContentOpen : ''}`}>
            <div className={styles.radioGroup}>
              <label className={`${styles.radioLabel} ${values.sort === 'new' || !values.sort ? styles.radioLabelActive : ''}`}>
                <input
                  type="radio"
                  name="sort"
                  className={styles.radioInput}
                  checked={values.sort === 'new' || !values.sort}
                  onChange={() => onChange({ sort: 'new' })}
                />
                <span className={styles.radioCustom}><span className={styles.radioDot} /></span>
                <span>Newest first</span>
              </label>
              <label className={`${styles.radioLabel} ${values.sort === 'price_asc' ? styles.radioLabelActive : ''}`}>
                <input
                  type="radio"
                  name="sort"
                  className={styles.radioInput}
                  checked={values.sort === 'price_asc'}
                  onChange={() => onChange({ sort: 'price_asc' })}
                />
                <span className={styles.radioCustom}><span className={styles.radioDot} /></span>
                <span>Price low to high</span>
              </label>
              <label className={`${styles.radioLabel} ${values.sort === 'price_desc' ? styles.radioLabelActive : ''}`}>
                <input
                  type="radio"
                  name="sort"
                  className={styles.radioInput}
                  checked={values.sort === 'price_desc'}
                  onChange={() => onChange({ sort: 'price_desc' })}
                />
                <span className={styles.radioCustom}><span className={styles.radioDot} /></span>
                <span>Price high to low</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      {hasActiveFilters && (
        <button type="button" className={styles.reset} onClick={onReset}>
          <svg className={styles.resetIcon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
            <path d="M3 3v5h5" />
          </svg>
          Reset filters
        </button>
      )}
    </section>
  )
}
