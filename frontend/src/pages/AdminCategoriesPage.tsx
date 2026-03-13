import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import { AdminNav } from '@/components/admin/AdminNav'
import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { categoryService } from '@/services/categoryService'
import type { Category } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'

import styles from '@/pages/AdminPage.module.scss'

interface CategoryFormState {
  name: string
  slug: string
  parentId: string
}

const initialFormState: CategoryFormState = {
  name: '',
  slug: '',
  parentId: '',
}

export const AdminCategoriesPage = () => {
  const [items, setItems] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [formState, setFormState] = useState<CategoryFormState>(initialFormState)

  const categoryMap = useMemo(
    () => new Map(items.map((category) => [category.id, category])),
    [items],
  )

  const parentOptions = useMemo(
    () => items.filter((category) => category.id !== editingId),
    [editingId, items],
  )

  const loadCategories = async () => {
    setLoading(true)
    setError(null)

    try {
      const nextItems = await categoryService.getCategories()
      setItems(nextItems)
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Failed to load categories'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadCategories()
  }, [])

  const resetForm = () => {
    setEditingId(null)
    setFormState(initialFormState)
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      const payload = {
        name: formState.name,
        slug: formState.slug || undefined,
        parentId: formState.parentId || undefined,
      }

      if (editingId) {
        await categoryService.updateCategory(editingId, payload)
      } else {
        await categoryService.createCategory(payload)
      }

      await loadCategories()
      resetForm()
    } catch (submitError) {
      setError(getErrorMessage(submitError, 'Failed to save category'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = (category: Category) => {
    setEditingId(category.id)
    setFormState({
      name: category.name,
      slug: category.slug ?? '',
      parentId: category.parentId ?? '',
    })
  }

  const handleDelete = async (category: Category) => {
    if (!window.confirm(`Delete category "${category.name}"?`)) {
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await categoryService.deleteCategory(category.id)
      await loadCategories()
      if (editingId === category.id) {
        resetForm()
      }
    } catch (deleteError) {
      setError(getErrorMessage(deleteError, 'Failed to delete category'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div>
          <h1>Categories</h1>
          <p>Maintain category tree and slugs used by the catalog and admin filters.</p>
        </div>
        <AdminNav />
      </section>

      {loading ? <AppLoader label="Loading categories..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.contentGrid}>
        <article className={styles.panel}>
          <h2>{editingId ? 'Edit category' : 'New category'}</h2>
          <p>Leave slug empty to generate it from the name. Parent is optional.</p>

          <form className={styles.form} onSubmit={handleSubmit}>
            <label>
              Name
              <input
                value={formState.name}
                onChange={(event) => setFormState((current) => ({ ...current, name: event.target.value }))}
                placeholder="Concrete mixes"
                required
              />
            </label>

            <label>
              Slug
              <input
                value={formState.slug}
                onChange={(event) => setFormState((current) => ({ ...current, slug: event.target.value }))}
                placeholder="concrete-mixes"
              />
            </label>

            <label>
              Parent category
              <select
                value={formState.parentId}
                onChange={(event) => setFormState((current) => ({ ...current, parentId: event.target.value }))}
              >
                <option value="">Root category</option>
                {parentOptions.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
            </label>

            <div className={styles.formActions}>
              <button type="submit" className={styles.primaryButton} disabled={submitting}>
                {submitting ? 'Saving...' : editingId ? 'Update category' : 'Create category'}
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
              <h2>Existing categories</h2>
              <p>{items.length} categories loaded.</p>
            </div>
          </div>

          {items.length === 0 && !loading ? (
            <div className={styles.empty}>
              <h2>No categories yet</h2>
              <p>Create the first category from the form.</p>
            </div>
          ) : (
            <div className={styles.list}>
              {items.map((category) => {
                const parentName = category.parentId ? categoryMap.get(category.parentId)?.name ?? 'Unknown parent' : 'Root'
                return (
                  <article key={category.id} className={styles.listCard}>
                    <div className={styles.listHeader}>
                      <div>
                        <h3>{category.name}</h3>
                        <p className={styles.listMeta}>Slug: {category.slug ?? '-'} · Parent: {parentName}</p>
                      </div>
                      <div className={styles.rowActions}>
                        <button type="button" className={styles.ghostButton} onClick={() => handleEdit(category)}>
                          Edit
                        </button>
                        <button
                          type="button"
                          className={styles.dangerButton}
                          onClick={() => handleDelete(category)}
                          disabled={submitting}
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </article>
                )
              })}
            </div>
          )}
        </article>
      </section>
    </div>
  )
}
