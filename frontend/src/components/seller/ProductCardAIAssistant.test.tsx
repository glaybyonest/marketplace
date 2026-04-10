import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ProductCardAIAssistant } from '@/components/seller/ProductCardAIAssistant'
import { sellerAiService } from '@/services/sellerAiService'

vi.mock('@/services/sellerAiService', async () => {
  const actual = await vi.importActual<typeof import('@/services/sellerAiService')>(
    '@/services/sellerAiService',
  )

  return {
    ...actual,
    sellerAiService: {
      generateProductCardDraft: vi.fn(),
    },
  }
})

const mockedSellerAiService = vi.mocked(sellerAiService)

describe('ProductCardAIAssistant', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mockedSellerAiService.generateProductCardDraft.mockResolvedValue({
      draft: {
        name: 'Термокружка Steel Cup',
        slug: 'termokruzhka-steel-cup',
        description: 'Аккуратный черновик карточки для маркетплейса.',
        brand: 'Steel Cup',
        unit: 'шт.',
        specs: {
          Объём: '450 мл',
        },
      },
      warnings: ['Проверьте характеристики вручную'],
      missingFields: ['сертификаты'],
      provider: 'openai',
      model: 'gpt-4.1-mini',
    })
  })

  it('requests a draft and applies it', async () => {
    const user = userEvent.setup()
    const onApplyDraft = vi.fn()

    render(
      <ProductCardAIAssistant
        isOpen
        defaultMode="generate"
        categoryName="Термопосуда"
        formState={{
          name: 'Термокружка',
          slug: '',
          description: 'Базовое описание',
          brand: '',
          unit: '',
          specsText: '{"Объём":"450 мл"}',
          categoryId: '7f93fbd5-7caf-490e-b4c7-bb2f37331c18',
        }}
        onApplyDraft={onApplyDraft}
        onClose={() => {}}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Сгенерировать черновик' }))

    expect(mockedSellerAiService.generateProductCardDraft).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'generate',
        categoryId: '7f93fbd5-7caf-490e-b4c7-bb2f37331c18',
        categoryName: 'Термопосуда',
      }),
    )

    expect(await screen.findByText('Предпросмотр AI-черновика')).toBeInTheDocument()
    expect(screen.getByText('Термокружка Steel Cup')).toBeInTheDocument()
    expect(screen.getByText('Проверьте характеристики вручную')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Применить в форму' }))

    expect(onApplyDraft).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Термокружка Steel Cup',
        slug: 'termokruzhka-steel-cup',
      }),
    )
  })
})
