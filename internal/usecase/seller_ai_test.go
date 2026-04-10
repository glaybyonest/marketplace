package usecase

import (
	"context"
	"errors"
	"testing"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type productCardDraftGeneratorStub struct {
	lastRequest StructuredJSONRequest
	result      StructuredJSONResult
	err         error
}

func (s *productCardDraftGeneratorStub) GenerateStructuredJSON(ctx context.Context, request StructuredJSONRequest) (StructuredJSONResult, error) {
	s.lastRequest = request
	if s.err != nil {
		return StructuredJSONResult{}, s.err
	}
	return s.result, nil
}

func TestSellerAIServiceGenerateProductCardDraft(t *testing.T) {
	t.Run("valid generate request", func(t *testing.T) {
		generator := &productCardDraftGeneratorStub{
			result: StructuredJSONResult{
				Output: `{
					"draft": {
						"name": "  Термокружка Steel Cup  ",
						"slug": "termokruzhka-steel-cup",
						"description": "Надежная термокружка для повседневного использования.",
						"brand": "",
						"unit": "",
						"specs": {
							"Объём": "450 мл",
							"Материал": "нержавеющая сталь"
						}
					},
					"warnings": ["Проверьте объём и материалы перед публикацией", "Проверьте объём и материалы перед публикацией"],
					"missing_fields": ["бренд"]
				}`,
				Provider: "openai",
				Model:    "gpt-4.1-mini",
			},
		}
		service := NewSellerAIService(true, generator)

		result, err := service.GenerateProductCardDraft(context.Background(), uuid.New(), GenerateProductCardInput{
			Mode:         productCardModeGenerate,
			CategoryName: "Термопосуда",
			SourceName:   "Термокружка Steel Cup",
			Features:     []string{"Двойные стенки", "Подходит для кофе"},
			Brand:        "Steel Cup",
			Unit:         "шт.",
			Specs: map[string]string{
				"Объём": "450 мл",
			},
			Tone: productCardToneNeutral,
		})

		require.NoError(t, err)
		assert.Equal(t, "Термокружка Steel Cup", result.Draft.Name)
		assert.Equal(t, "termokruzhka-steel-cup", result.Draft.Slug)
		assert.Equal(t, "Steel Cup", result.Draft.Brand)
		assert.Equal(t, "шт.", result.Draft.Unit)
		assert.Equal(t, map[string]string{"Объём": "450 мл"}, result.Draft.Specs)
		assert.Equal(t, []string{"Проверьте объём и материалы перед публикацией"}, result.Warnings)
		assert.Equal(t, "openai", result.Provider)
		assert.Equal(t, "gpt-4.1-mini", result.Model)
		assert.Contains(t, generator.lastRequest.Instructions, "Не выдумывай факты")
		assert.Equal(t, "seller_product_card_draft", generator.lastRequest.SchemaName)
	})

	t.Run("valid improve request", func(t *testing.T) {
		generator := &productCardDraftGeneratorStub{
			result: StructuredJSONResult{
				Output: `{
					"draft": {
						"name": "",
						"slug": "",
						"description": "Краткое и аккуратное описание без лишних обещаний.",
						"brand": "",
						"unit": "",
						"specs": {}
					},
					"warnings": [],
					"missing_fields": []
				}`,
				Provider: "openai",
				Model:    "gpt-4.1-mini",
			},
		}
		service := NewSellerAIService(true, generator)

		result, err := service.GenerateProductCardDraft(context.Background(), uuid.New(), GenerateProductCardInput{
			Mode:           productCardModeImprove,
			CategoryID:     uuid.NewString(),
			RawDescription: "Старое описание",
			ExistingDraft: GeneratedProductCardDraft{
				Name:        "Смарт-лампа Luma",
				Slug:        "smart-lampa-luma",
				Description: "Старое описание",
				Brand:       "Luma",
				Unit:        "шт.",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "Смарт-лампа Luma", result.Draft.Name)
		assert.Equal(t, "smart-lampa-luma", result.Draft.Slug)
		assert.Equal(t, "Luma", result.Draft.Brand)
		assert.Equal(t, "шт.", result.Draft.Unit)
		assert.Equal(t, "Краткое и аккуратное описание без лишних обещаний", result.Draft.Description)
	})

	t.Run("feature disabled", func(t *testing.T) {
		service := NewSellerAIService(false, nil)

		_, err := service.GenerateProductCardDraft(context.Background(), uuid.New(), GenerateProductCardInput{
			Mode:         productCardModeGenerate,
			CategoryName: "Категория",
			SourceName:   "Товар",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrFeatureDisabled)
	})

	t.Run("invalid payload", func(t *testing.T) {
		service := NewSellerAIService(true, &productCardDraftGeneratorStub{})

		_, err := service.GenerateProductCardDraft(context.Background(), uuid.New(), GenerateProductCardInput{
			Mode:         "bad-mode",
			CategoryName: "Категория",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUnprocessable)
	})

	t.Run("provider error", func(t *testing.T) {
		service := NewSellerAIService(true, &productCardDraftGeneratorStub{
			err: errors.New("timeout"),
		})

		_, err := service.GenerateProductCardDraft(context.Background(), uuid.New(), GenerateProductCardInput{
			Mode:         productCardModeGenerate,
			CategoryName: "Категория",
			SourceName:   "Товар",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProviderFailed)
	})

	t.Run("sanitize output", func(t *testing.T) {
		result := sanitizeGeneratedDraft(GenerateProductCardInput{
			SourceName: "Смарт лампа",
			Brand:      "Luma",
			Unit:       "шт.",
			Specs: map[string]string{
				"Цвет": "белый",
			},
		}, ProductCardDraftResult{
			Draft: GeneratedProductCardDraft{
				Name:        "  Смарт лампа  ",
				Slug:        "смарт лампа",
				Description: "100% лучший выбор. Подходит для дома.",
				Specs: map[string]string{
					"Цвет":   " белый ",
					"Размер": "40 см",
				},
			},
			Warnings:      []string{"  Проверить текст  ", "Проверить текст"},
			MissingFields: []string{"", "бренд"},
		})

		assert.Equal(t, "smart-lampa", result.Draft.Slug)
		assert.Equal(t, "Подходит для дома", result.Draft.Description)
		assert.Equal(t, map[string]string{"Цвет": "белый"}, result.Draft.Specs)
		assert.Equal(t, []string{"Проверить текст"}, result.Warnings)
		assert.NotContains(t, result.MissingFields, "бренд")
	})
}
