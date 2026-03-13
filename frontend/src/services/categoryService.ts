import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { Category } from '@/types/domain'
import { normalizeCategory } from '@/utils/normalize'

interface CategoryPayload {
  name: string
  slug?: string
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
    void _payload
    throw new Error('Category creation is not supported by current backend API')
  },

  async updateCategory(_id: string, _payload: Partial<CategoryPayload>): Promise<Category> {
    void _id
    void _payload
    throw new Error('Category update is not supported by current backend API')
  },

  async deleteCategory(_id: string): Promise<void> {
    void _id
    throw new Error('Category deletion is not supported by current backend API')
  },
}

