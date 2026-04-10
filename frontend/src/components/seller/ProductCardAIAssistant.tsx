import { useEffect, useState } from 'react'

import {
  sellerAiService,
  type ProductCardAIDraft,
  type ProductCardAIDraftResult,
  type ProductCardAIMode,
  type ProductCardAITone,
} from '@/services/sellerAiService'
import { toApiError } from '@/utils/error'

import styles from '@/components/seller/ProductCardAIAssistant.module.scss'

interface ProductCardFormSnapshot {
  name: string
  slug: string
  description: string
  brand: string
  unit: string
  specsText: string
  categoryId: string
}

interface ProductCardAIAssistantProps {
  isOpen: boolean
  defaultMode: ProductCardAIMode
  categoryName?: string
  formState: ProductCardFormSnapshot
  busy?: boolean
  onApplyDraft: (draft: ProductCardAIDraft) => void
  onClose: () => void
}

const buildList = (value: string) =>
  value
    .split(/\n|,/)
    .map((item) => item.trim())
    .filter(Boolean)

const parseSpecsText = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) {
    return { specs: {} as Record<string, string>, error: '' }
  }

  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { specs: {}, error: 'Характеристики не распознаны: ожидается JSON-объект.' }
    }

    const specs: Record<string, string> = {}
    for (const [key, rawValue] of Object.entries(parsed)) {
      const normalizedKey = key.trim()
      if (!normalizedKey) {
        continue
      }
      if (
        typeof rawValue !== 'string' &&
        typeof rawValue !== 'number' &&
        typeof rawValue !== 'boolean'
      ) {
        continue
      }
      specs[normalizedKey] = String(rawValue).trim()
    }

    return { specs, error: '' }
  } catch {
    return {
      specs: {},
      error:
        'Характеристики не распознаны: исправьте JSON, если хотите использовать их в AI-черновике.',
    }
  }
}

const hasMeaningfulDraft = (draft: ProductCardAIDraft) =>
  Boolean(
    draft.name.trim() ||
    draft.slug.trim() ||
    draft.description.trim() ||
    draft.brand.trim() ||
    draft.unit.trim() ||
    Object.keys(draft.specs).length > 0,
  )

export const ProductCardAIAssistant = ({
  isOpen,
  defaultMode,
  categoryName,
  formState,
  busy = false,
  onApplyDraft,
  onClose,
}: ProductCardAIAssistantProps) => {
  const [mode, setMode] = useState<ProductCardAIMode>(defaultMode)
  const [tone, setTone] = useState<ProductCardAITone>('neutral')
  const [featuresText, setFeaturesText] = useState('')
  const [keywordsText, setKeywordsText] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [availabilityMessage, setAvailabilityMessage] = useState('')
  const [specNotice, setSpecNotice] = useState('')
  const [result, setResult] = useState<ProductCardAIDraftResult | null>(null)

  useEffect(() => {
    if (!isOpen) {
      return
    }

    setMode(defaultMode)
    setError('')
    setAvailabilityMessage('')
  }, [defaultMode, isOpen])

  if (!isOpen) {
    return null
  }

  const handleRun = async (nextMode: ProductCardAIMode = mode) => {
    const parsedSpecs = parseSpecsText(formState.specsText)
    setSpecNotice(parsedSpecs.error)
    setLoading(true)
    setError('')

    const existingDraft: ProductCardAIDraft = {
      name: formState.name,
      slug: formState.slug,
      description: formState.description,
      brand: formState.brand,
      unit: formState.unit,
      specs: parsedSpecs.specs,
    }

    try {
      const response = await sellerAiService.generateProductCardDraft({
        mode: nextMode,
        categoryId: formState.categoryId || undefined,
        categoryName: categoryName?.trim() || undefined,
        sourceName: formState.name || undefined,
        rawDescription: formState.description || undefined,
        features: buildList(featuresText),
        brand: formState.brand || undefined,
        unit: formState.unit || undefined,
        specs: parsedSpecs.specs,
        keywords: buildList(keywordsText),
        tone,
        existingDraft: hasMeaningfulDraft(existingDraft) ? existingDraft : undefined,
      })

      setResult(response)
      setAvailabilityMessage('')
      setMode(nextMode)
    } catch (runError) {
      const apiError = toApiError(runError)
      if (apiError.code === 'feature_disabled' || apiError.code === 'provider_unavailable') {
        setAvailabilityMessage(
          apiError.code === 'feature_disabled'
            ? 'AI-ассистент сейчас отключён на backend. Форма товара продолжает работать в обычном режиме.'
            : 'AI-ассистент временно недоступен. Попробуйте позже или заполните форму вручную.',
        )
      }
      setError(apiError.message)
      setResult(null)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className={styles.assistant} aria-label="AI-черновик карточки">
      <div className={styles.header}>
        <div>
          <span className="badge-pill">AI-ассистент</span>
          <h3>Черновик карточки внутри текущей формы</h3>
          <p>
            Ассистент собирает структурированный draft и не сохраняет товар автоматически. Перед
            сохранением продавец должен проверить факты вручную.
          </p>
        </div>
        <div className={styles.actions}>
          <button
            type="button"
            className="action-ghost"
            onClick={onClose}
            disabled={loading || busy}
          >
            Скрыть панель
          </button>
        </div>
      </div>

      <div className={styles.controls}>
        <div className={styles.modeRow}>
          <button
            type="button"
            className={`${styles.modeButton} ${mode === 'generate' ? styles.modeButtonActive : ''}`}
            onClick={() => setMode('generate')}
            disabled={loading}
          >
            AI-черновик карточки
          </button>
          <button
            type="button"
            className={`${styles.modeButton} ${mode === 'improve' ? styles.modeButtonActive : ''}`}
            onClick={() => setMode('improve')}
            disabled={loading}
          >
            Улучшить текст
          </button>
        </div>

        <div className={styles.controlGrid}>
          <label className={styles.field}>
            Тон текста
            <select
              value={tone}
              onChange={(event) => setTone(event.target.value as ProductCardAITone)}
              disabled={loading}
            >
              <option value="neutral">Нейтральный</option>
              <option value="sales">Коммерческий</option>
              <option value="premium">Премиальный</option>
            </select>
          </label>

          <label className={`${styles.field} ${styles.fullWidth}`}>
            Подсказки и факты о товаре
            <textarea
              value={featuresText}
              onChange={(event) => setFeaturesText(event.target.value)}
              placeholder="Каждый факт с новой строки: двойные стенки, объём 450 мл, подходит для кофе"
            />
          </label>

          <label className={`${styles.field} ${styles.fullWidth}`}>
            Ключевые слова
            <textarea
              value={keywordsText}
              onChange={(event) => setKeywordsText(event.target.value)}
              placeholder="Например: термокружка, кружка для кофе, кружка в дорогу"
            />
          </label>
        </div>

        <div className={styles.actions}>
          <button
            type="button"
            className="action-primary"
            onClick={() => void handleRun(mode)}
            disabled={loading || busy}
          >
            {loading
              ? 'Генерируем...'
              : mode === 'generate'
                ? 'Сгенерировать черновик'
                : 'Улучшить текст'}
          </button>
          <button
            type="button"
            className="action-secondary"
            onClick={() => void handleRun('improve')}
            disabled={loading || busy}
          >
            Улучшить текст
          </button>
        </div>
      </div>

      {specNotice ? <div className={styles.notice}>{specNotice}</div> : null}
      {availabilityMessage ? (
        <div className={styles.availability}>{availabilityMessage}</div>
      ) : null}
      {error && !availabilityMessage ? <div className={styles.error}>{error}</div> : null}

      {result ? (
        <section className={styles.preview}>
          <div className={styles.previewHeader}>
            <div>
              <h4>Предпросмотр AI-черновика</h4>
              <p>
                Это вспомогательный draft, а не финальная истина. Проверьте данные перед сохранением
                товара.
              </p>
            </div>
            <div className={styles.actions}>
              <button
                type="button"
                className="action-primary"
                onClick={() => onApplyDraft(result.draft)}
                disabled={busy}
              >
                Применить в форму
              </button>
            </div>
          </div>

          <div className={styles.meta}>
            Провайдер: {result.provider || 'openai'}
            {result.model ? ` • ${result.model}` : ''}
          </div>

          <div className={styles.previewGrid}>
            <div className={styles.previewField}>
              <span>Название</span>
              <strong>{result.draft.name || 'Не заполнено'}</strong>
            </div>
            <div className={styles.previewField}>
              <span>Slug</span>
              <code>{result.draft.slug || 'Не заполнено'}</code>
            </div>
            <div className={`${styles.previewField} ${styles.fullWidth}`}>
              <span>Описание</span>
              <p className={styles.description}>{result.draft.description || 'Не заполнено'}</p>
            </div>
            <div className={styles.previewField}>
              <span>Бренд</span>
              <strong>{result.draft.brand || 'Не заполнено'}</strong>
            </div>
            <div className={styles.previewField}>
              <span>Единица продажи</span>
              <strong>{result.draft.unit || 'Не заполнено'}</strong>
            </div>
          </div>

          <div className={styles.previewField}>
            <span>Характеристики</span>
            {Object.keys(result.draft.specs).length > 0 ? (
              <div className={styles.specs}>
                {Object.entries(result.draft.specs).map(([key, value]) => (
                  <div key={key} className={styles.specRow}>
                    <span className={styles.specKey}>{key}</span>
                    <strong>{value}</strong>
                  </div>
                ))}
              </div>
            ) : (
              <p>Характеристики не заполнены.</p>
            )}
          </div>

          {result.warnings.length > 0 ? (
            <div className={styles.previewField}>
              <span>Предупреждения</span>
              <div className={styles.chips}>
                {result.warnings.map((warning) => (
                  <span key={warning} className={styles.chipWarn}>
                    {warning}
                  </span>
                ))}
              </div>
            </div>
          ) : null}

          {result.missingFields.length > 0 ? (
            <div className={styles.previewField}>
              <span>Что ещё нужно проверить или заполнить вручную</span>
              <div className={styles.chips}>
                {result.missingFields.map((field) => (
                  <span key={field} className={styles.chipMissing}>
                    {field}
                  </span>
                ))}
              </div>
            </div>
          ) : null}
        </section>
      ) : null}
    </section>
  )
}
