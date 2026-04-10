import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'

export type ProductCardAIMode = 'generate' | 'improve'
export type ProductCardAITone = 'neutral' | 'sales' | 'premium'

export interface ProductCardAIDraft {
  name: string
  slug: string
  description: string
  brand: string
  unit: string
  specs: Record<string, string>
}

export interface GenerateProductCardDraftPayload {
  mode: ProductCardAIMode
  categoryId?: string
  categoryName?: string
  sourceName?: string
  rawDescription?: string
  features?: string[]
  brand?: string
  unit?: string
  specs?: Record<string, string>
  keywords?: string[]
  tone?: ProductCardAITone
  existingDraft?: ProductCardAIDraft
}

export interface ProductCardAIDraftResult {
  draft: ProductCardAIDraft
  warnings: string[]
  missingFields: string[]
  provider: string
  model: string
}

const normalizeDraft = (raw: Record<string, unknown>): ProductCardAIDraft => {
  const sourceSpecs = (raw.specs as Record<string, unknown>) ?? {}
  const specs: Record<string, string> = {}

  for (const [key, value] of Object.entries(sourceSpecs)) {
    if (typeof value !== 'string') {
      continue
    }

    const normalizedKey = key.trim()
    const normalizedValue = value.trim()
    if (!normalizedKey || !normalizedValue) {
      continue
    }

    specs[normalizedKey] = normalizedValue
  }

  return {
    name: typeof raw.name === 'string' ? raw.name : '',
    slug: typeof raw.slug === 'string' ? raw.slug : '',
    description: typeof raw.description === 'string' ? raw.description : '',
    brand: typeof raw.brand === 'string' ? raw.brand : '',
    unit: typeof raw.unit === 'string' ? raw.unit : '',
    specs,
  }
}

const normalizeStringArray = (raw: unknown) =>
  Array.isArray(raw) ? raw.filter((value): value is string => typeof value === 'string') : []

export const sellerAiService = {
  async generateProductCardDraft(
    payload: GenerateProductCardDraftPayload,
  ): Promise<ProductCardAIDraftResult> {
    const response = await apiClient.post('/v1/seller/ai/product-card', {
      mode: payload.mode,
      category_id: payload.categoryId,
      category_name: payload.categoryName,
      source_name: payload.sourceName,
      raw_description: payload.rawDescription,
      features: payload.features ?? [],
      brand: payload.brand,
      unit: payload.unit,
      specs: payload.specs ?? {},
      keywords: payload.keywords ?? [],
      tone: payload.tone ?? 'neutral',
      existing_draft: payload.existingDraft
        ? {
            name: payload.existingDraft.name,
            slug: payload.existingDraft.slug,
            description: payload.existingDraft.description,
            brand: payload.existingDraft.brand,
            unit: payload.existingDraft.unit,
            specs: payload.existingDraft.specs,
          }
        : undefined,
    })

    const data = pickData<Record<string, unknown>>(response.data)

    return {
      draft: normalizeDraft((data.draft as Record<string, unknown>) ?? {}),
      warnings: normalizeStringArray(data.warnings),
      missingFields: normalizeStringArray(data.missing_fields),
      provider: typeof data.provider === 'string' ? data.provider : '',
      model: typeof data.model === 'string' ? data.model : '',
    }
  },
}
