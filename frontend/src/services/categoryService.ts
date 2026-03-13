import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { Category } from '@/types/domain'
import { normalizeCategory } from '@/utils/normalize'

interface CategoryPayload {
  name: string
  slug?: string
  parentId?: string
}

const flattenCategories = (nodes: unknown[]): Category[] => {
  const result: Category[] = []

  const visit = (node: unknown) => {
    const category = normalizeCategory(node)
    result.push(category)

    const children = (node as { children?: unknown[] })?.children
    if (Array.isArray(children)) {
      children.forEach(visit)
    }
  }

  nodes.forEach(visit)
  return result
}

export const categoryService = {
  async getCategories(): Promise<Category[]> {
    const response = await apiClient.get('/v1/categories')
    const source = pickData<unknown>(response.data)

    if (Array.isArray(source)) {
      return flattenCategories(source)
    }

    return []
  },

  async createCategory(_payload: CategoryPayload): Promise<Category> {
    const response = await apiClient.post('/v1/admin/categories', {
      name: _payload.name,
      slug: _payload.slug,
      parent_id: _payload.parentId || undefined,
    })
    return normalizeCategory(pickData(response.data))
  },

  async updateCategory(_id: string, _payload: Partial<CategoryPayload>): Promise<Category> {
    const response = await apiClient.patch(`/v1/admin/categories/${_id}`, {
      name: _payload.name,
      slug: _payload.slug,
      parent_id: _payload.parentId || undefined,
    })
    return normalizeCategory(pickData(response.data))
  },

  async deleteCategory(_id: string): Promise<void> {
    await apiClient.delete(`/v1/admin/categories/${_id}`)
  },
}
