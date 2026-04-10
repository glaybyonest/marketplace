package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

const (
	productCardModeGenerate = "generate"
	productCardModeImprove  = "improve"

	productCardToneNeutral = "neutral"
	productCardToneSales   = "sales"
	productCardTonePremium = "premium"

	productCardProviderOpenAI = "openai"

	maxProductCardCategoryNameLength = 160
	maxProductCardSourceNameLength   = 180
	maxProductCardDescriptionLength  = 4000
	maxProductCardBrandLength        = 120
	maxProductCardUnitLength         = 64
	maxProductCardSpecs              = 30
	maxProductCardSpecKeyLength      = 80
	maxProductCardSpecValueLength    = 300
	maxProductCardFeatures           = 20
	maxProductCardFeatureLength      = 180
	maxProductCardKeywords           = 20
	maxProductCardKeywordLength      = 80
	maxProductCardWarnings           = 12
	maxProductCardWarningLength      = 220
	maxProductCardMissingFields      = 12
	maxProductCardMissingFieldLength = 120

	maxGeneratedDraftNameLength        = 180
	maxGeneratedDraftSlugLength        = 180
	maxGeneratedDraftDescriptionLength = 2000
)

type GenerateProductCardInput struct {
	Mode           string                    `json:"mode"`
	CategoryID     string                    `json:"category_id"`
	CategoryName   string                    `json:"category_name"`
	SourceName     string                    `json:"source_name"`
	RawDescription string                    `json:"raw_description"`
	Features       []string                  `json:"features"`
	Brand          string                    `json:"brand"`
	Unit           string                    `json:"unit"`
	Specs          map[string]string         `json:"specs"`
	Keywords       []string                  `json:"keywords"`
	Tone           string                    `json:"tone"`
	ExistingDraft  GeneratedProductCardDraft `json:"existing_draft"`
}

type GeneratedProductCardDraft struct {
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	Brand       string            `json:"brand"`
	Unit        string            `json:"unit"`
	Specs       map[string]string `json:"specs"`
}

type ProductCardDraftResult struct {
	Draft         GeneratedProductCardDraft `json:"draft"`
	Warnings      []string                  `json:"warnings"`
	MissingFields []string                  `json:"missing_fields"`
	Provider      string                    `json:"provider"`
	Model         string                    `json:"model"`
}

type StructuredJSONRequest struct {
	Instructions string
	Input        string
	SchemaName   string
	Schema       map[string]any
}

type StructuredJSONResult struct {
	Output   string
	Provider string
	Model    string
}

type ProductCardDraftGenerator interface {
	GenerateStructuredJSON(ctx context.Context, request StructuredJSONRequest) (StructuredJSONResult, error)
}

type SellerAIService struct {
	enabled   bool
	generator ProductCardDraftGenerator
}

func NewSellerAIService(enabled bool, generator ProductCardDraftGenerator) *SellerAIService {
	return &SellerAIService{
		enabled:   enabled,
		generator: generator,
	}
}

func (s *SellerAIService) GenerateProductCardDraft(ctx context.Context, sellerID uuid.UUID, input GenerateProductCardInput) (ProductCardDraftResult, error) {
	if sellerID == uuid.Nil {
		return ProductCardDraftResult{}, domain.ErrUnauthorized
	}
	if s == nil || !s.enabled || s.generator == nil {
		return ProductCardDraftResult{}, domain.ErrFeatureDisabled
	}

	normalized, err := normalizeGenerateProductCardInput(input)
	if err != nil {
		return ProductCardDraftResult{}, err
	}

	payload, err := buildProductCardGenerationPayload(normalized)
	if err != nil {
		return ProductCardDraftResult{}, domain.ErrProviderFailed
	}

	generated, err := s.generator.GenerateStructuredJSON(ctx, StructuredJSONRequest{
		Instructions: buildProductCardSystemPrompt(),
		Input:        string(payload),
		SchemaName:   "seller_product_card_draft",
		Schema:       buildProductCardStructuredSchema(),
	})
	if err != nil {
		return ProductCardDraftResult{}, domain.ErrProviderFailed
	}

	var raw struct {
		Draft         GeneratedProductCardDraft `json:"draft"`
		Warnings      []string                  `json:"warnings"`
		MissingFields []string                  `json:"missing_fields"`
	}
	if err := json.Unmarshal([]byte(generated.Output), &raw); err != nil {
		return ProductCardDraftResult{}, domain.ErrProviderFailed
	}

	return sanitizeGeneratedDraft(normalized, ProductCardDraftResult{
		Draft:         raw.Draft,
		Warnings:      raw.Warnings,
		MissingFields: raw.MissingFields,
		Provider:      generated.Provider,
		Model:         generated.Model,
	}), nil
}

func normalizeGenerateProductCardInput(input GenerateProductCardInput) (GenerateProductCardInput, error) {
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	switch input.Mode {
	case productCardModeGenerate, productCardModeImprove:
	default:
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	input.CategoryID = strings.TrimSpace(input.CategoryID)
	if input.CategoryID != "" {
		if _, err := uuid.Parse(input.CategoryID); err != nil {
			return GenerateProductCardInput{}, domain.ErrUnprocessable
		}
	}

	input.CategoryName = trimToLimit(input.CategoryName, maxProductCardCategoryNameLength)
	if input.CategoryID == "" && input.CategoryName == "" {
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	input.SourceName = trimToLimit(input.SourceName, maxProductCardSourceNameLength)
	input.RawDescription = trimToLimit(input.RawDescription, maxProductCardDescriptionLength)
	input.Brand = trimToLimit(input.Brand, maxProductCardBrandLength)
	input.Unit = trimToLimit(input.Unit, maxProductCardUnitLength)
	input.Features = normalizeStringListWithLimit(input.Features, maxProductCardFeatures, maxProductCardFeatureLength)
	input.Keywords = normalizeStringListWithLimit(input.Keywords, maxProductCardKeywords, maxProductCardKeywordLength)

	var err error
	input.Specs, err = normalizeStringMap(input.Specs, maxProductCardSpecs, maxProductCardSpecKeyLength, maxProductCardSpecValueLength)
	if err != nil {
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	input.ExistingDraft, err = normalizeGeneratedDraftPayload(input.ExistingDraft)
	if err != nil {
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	input.Tone = strings.ToLower(strings.TrimSpace(input.Tone))
	if input.Tone == "" {
		input.Tone = productCardToneNeutral
	}
	switch input.Tone {
	case productCardToneNeutral, productCardToneSales, productCardTonePremium:
	default:
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	if !hasMeaningfulProductCardInput(input) {
		return GenerateProductCardInput{}, domain.ErrUnprocessable
	}

	return input, nil
}

func normalizeGeneratedDraftPayload(draft GeneratedProductCardDraft) (GeneratedProductCardDraft, error) {
	draft.Name = trimToLimit(draft.Name, maxGeneratedDraftNameLength)
	draft.Slug = trimToLimit(draft.Slug, maxGeneratedDraftSlugLength)
	draft.Description = trimToLimit(draft.Description, maxGeneratedDraftDescriptionLength)
	draft.Brand = trimToLimit(draft.Brand, maxProductCardBrandLength)
	draft.Unit = trimToLimit(draft.Unit, maxProductCardUnitLength)

	var err error
	draft.Specs, err = normalizeStringMap(draft.Specs, maxProductCardSpecs, maxProductCardSpecKeyLength, maxProductCardSpecValueLength)
	if err != nil {
		return GeneratedProductCardDraft{}, err
	}

	return draft, nil
}

func hasMeaningfulProductCardInput(input GenerateProductCardInput) bool {
	return input.SourceName != "" ||
		input.RawDescription != "" ||
		len(input.Features) > 0 ||
		len(input.Specs) > 0 ||
		hasMeaningfulDraft(input.ExistingDraft)
}

func hasMeaningfulDraft(draft GeneratedProductCardDraft) bool {
	return draft.Name != "" ||
		draft.Slug != "" ||
		draft.Description != "" ||
		draft.Brand != "" ||
		draft.Unit != "" ||
		len(draft.Specs) > 0
}

func buildProductCardGenerationPayload(input GenerateProductCardInput) ([]byte, error) {
	type draftPayload struct {
		Name        string            `json:"name"`
		Slug        string            `json:"slug"`
		Description string            `json:"description"`
		Brand       string            `json:"brand"`
		Unit        string            `json:"unit"`
		Specs       map[string]string `json:"specs"`
	}

	return json.Marshal(map[string]any{
		"mode":            input.Mode,
		"category_id":     input.CategoryID,
		"category_name":   input.CategoryName,
		"source_name":     input.SourceName,
		"raw_description": input.RawDescription,
		"features":        input.Features,
		"brand":           input.Brand,
		"unit":            input.Unit,
		"specs":           input.Specs,
		"keywords":        input.Keywords,
		"tone":            input.Tone,
		"existing_draft": draftPayload{
			Name:        input.ExistingDraft.Name,
			Slug:        input.ExistingDraft.Slug,
			Description: input.ExistingDraft.Description,
			Brand:       input.ExistingDraft.Brand,
			Unit:        input.ExistingDraft.Unit,
			Specs:       input.ExistingDraft.Specs,
		},
	})
}

func buildProductCardSystemPrompt() string {
	return strings.Join([]string{
		"Ты AI-ассистент маркетплейса для продавцов.",
		"Отвечай только на русском языке, кроме поля slug: slug должен быть в латинице, в нижнем регистре, с дефисами.",
		"Сформируй только структурированный JSON по заданной схеме для черновика карточки товара.",
		"Не выдумывай факты. Если факт неизвестен или не дан явно, не заполняй его и укажи это в missing_fields или warnings.",
		"Черновик должен подходить для карточки товара маркетплейса: аккуратно, коммерчески понятно, без крикливого спама и без ложных обещаний.",
		"Не используй CAPS LOCK, агрессивные маркетинговые штампы, медицинские/лечебные обещания, утверждения вроде «100% лучший», «гарантированно лечит», «лучший на рынке».",
		"Никогда не придумывай цену, остаток, sku, сертификаты, гарантию, размеры, материалы, состав, страну происхождения и технические характеристики, если они не были явно переданы во входе.",
		"Поле specs заполняй только по явно переданным характеристикам. Если характеристик недостаточно, оставь specs пустым объектом и укажи missing_fields.",
		"Если mode=improve, бережно улучшай существующий черновик, сохраняя уже известные факты и не добавляя новых неподтвержденных данных.",
		"Описание должно помогать продавцу заполнить карточку, но оставаться черновиком, который нужно проверить вручную.",
	}, "\n")
}

func buildProductCardStructuredSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"draft", "warnings", "missing_fields"},
		"properties": map[string]any{
			"draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "slug", "description", "brand", "unit", "specs"},
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
					"slug": map[string]any{
						"type": "string",
					},
					"description": map[string]any{
						"type": "string",
					},
					"brand": map[string]any{
						"type": "string",
					},
					"unit": map[string]any{
						"type": "string",
					},
					"specs": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "string",
						},
					},
				},
			},
			"warnings": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			"missing_fields": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}

func sanitizeGeneratedDraft(input GenerateProductCardInput, result ProductCardDraftResult) ProductCardDraftResult {
	draft := result.Draft

	draft.Name = trimToLimit(draft.Name, maxGeneratedDraftNameLength)
	if draft.Name == "" {
		draft.Name = firstNonEmpty(input.ExistingDraft.Name, input.SourceName)
	}

	draft.Slug = trimToLimit(draft.Slug, maxGeneratedDraftSlugLength)
	draft.Slug = normalizeAISlug(draft.Slug)
	if draft.Slug == "" {
		draft.Slug = normalizeAISlug(firstNonEmpty(input.ExistingDraft.Slug, draft.Name, input.SourceName, input.ExistingDraft.Name))
	}

	draft.Description = trimToLimit(compactWhitespace(draft.Description), maxGeneratedDraftDescriptionLength)
	draft.Description = removeProhibitedClaims(draft.Description)
	if draft.Description == "" {
		draft.Description = firstNonEmpty(input.ExistingDraft.Description, input.RawDescription)
		draft.Description = trimToLimit(compactWhitespace(draft.Description), maxGeneratedDraftDescriptionLength)
	}

	draft.Brand = trimToLimit(draft.Brand, maxProductCardBrandLength)
	if draft.Brand == "" {
		draft.Brand = firstNonEmpty(input.Brand, input.ExistingDraft.Brand)
	}

	draft.Unit = trimToLimit(draft.Unit, maxProductCardUnitLength)
	if draft.Unit == "" {
		draft.Unit = firstNonEmpty(input.Unit, input.ExistingDraft.Unit)
	}

	draft.Specs = sanitizeGeneratedSpecs(draft.Specs, input)
	if len(draft.Specs) == 0 {
		if len(input.Specs) > 0 {
			draft.Specs = cloneStringMap(input.Specs)
		} else if len(input.ExistingDraft.Specs) > 0 {
			draft.Specs = cloneStringMap(input.ExistingDraft.Specs)
		}
	}

	result.Draft = draft
	result.Warnings = normalizeStringListWithLimit(result.Warnings, maxProductCardWarnings, maxProductCardWarningLength)
	result.MissingFields = normalizeStringListWithLimit(result.MissingFields, maxProductCardMissingFields, maxProductCardMissingFieldLength)
	result.MissingFields = pruneResolvedMissingFields(result.MissingFields, draft)
	result.Provider = firstNonEmpty(strings.TrimSpace(result.Provider), productCardProviderOpenAI)
	result.Model = strings.TrimSpace(result.Model)

	result = appendMissingField(result, draft.Brand == "", "бренд")
	result = appendMissingField(result, draft.Unit == "", "единица продажи")
	result = appendMissingField(result, draft.Description == "", "описание товара")
	result = appendMissingField(result, draft.Name == "", "название товара")
	result = appendMissingField(result, len(draft.Specs) == 0, "подтверждённые характеристики")

	return result
}

func sanitizeGeneratedSpecs(specs map[string]string, input GenerateProductCardInput) map[string]string {
	normalized, err := normalizeStringMap(specs, maxProductCardSpecs, maxProductCardSpecKeyLength, maxProductCardSpecValueLength)
	if err != nil || len(normalized) == 0 {
		return map[string]string{}
	}

	allowed := make(map[string]string, len(input.Specs)+len(input.ExistingDraft.Specs))
	for key := range input.Specs {
		allowed[normalizeKey(key)] = key
	}
	for key := range input.ExistingDraft.Specs {
		if _, exists := allowed[normalizeKey(key)]; exists {
			continue
		}
		allowed[normalizeKey(key)] = key
	}
	if len(allowed) == 0 {
		return map[string]string{}
	}

	result := make(map[string]string, len(normalized))
	for key, value := range normalized {
		allowedKey, ok := allowed[normalizeKey(key)]
		if !ok {
			continue
		}
		result[allowedKey] = value
	}
	return result
}

func normalizeStringListWithLimit(items []string, maxItems, maxItemLength int) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = trimToLimit(item, maxItemLength)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
		if len(result) >= maxItems {
			break
		}
	}
	return result
}

func normalizeStringMap(input map[string]string, maxItems, maxKeyLength, maxValueLength int) (map[string]string, error) {
	if input == nil {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(input))
	seen := make(map[string]struct{}, len(input))
	for key, value := range input {
		key = trimToLimit(key, maxKeyLength)
		value = trimToLimit(value, maxValueLength)
		if key == "" || value == "" {
			continue
		}
		normalizedKey := normalizeKey(key)
		if normalizedKey == "" {
			continue
		}
		if _, exists := seen[normalizedKey]; exists {
			continue
		}
		seen[normalizedKey] = struct{}{}
		result[key] = value
		if len(result) >= maxItems {
			break
		}
	}
	return result, nil
}

func removeProhibitedClaims(value string) string {
	if value == "" {
		return ""
	}

	segments := splitTextSegments(value)
	clean := make([]string, 0, len(segments))
	for _, segment := range segments {
		lower := strings.ToLower(segment)
		if strings.Contains(lower, "100% лучший") ||
			strings.Contains(lower, "гарантированно") ||
			strings.Contains(lower, "лечит") ||
			strings.Contains(lower, "самый лучший") ||
			strings.Contains(lower, "лучший на рынке") {
			continue
		}
		clean = append(clean, segment)
	}
	return compactWhitespace(strings.Join(clean, ". "))
}

func splitTextSegments(value string) []string {
	value = strings.NewReplacer("!", ".", "?", ".", "\n", ".").Replace(value)
	parts := strings.Split(value, ".")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func appendMissingField(result ProductCardDraftResult, condition bool, value string) ProductCardDraftResult {
	if !condition {
		return result
	}
	result.MissingFields = normalizeStringListWithLimit(append(result.MissingFields, value), maxProductCardMissingFields, maxProductCardMissingFieldLength)
	return result
}

func normalizeAISlug(value string) string {
	if containsCyrillic(value) {
		return trimToLimit(normalizeSlug(transliterateCyrillic(value)), maxGeneratedDraftSlugLength)
	}

	slug := normalizeSlug(value)
	if slug != "" {
		return trimToLimit(slug, maxGeneratedDraftSlugLength)
	}
	return trimToLimit(normalizeSlug(transliterateCyrillic(value)), maxGeneratedDraftSlugLength)
}

func transliterateCyrillic(value string) string {
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e", "ё", "e",
		"ж", "zh", "з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m",
		"н", "n", "о", "o", "п", "p", "р", "r", "с", "s", "т", "t", "у", "u",
		"ф", "f", "х", "h", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "sch", "ъ", "",
		"ы", "y", "ь", "", "э", "e", "ю", "yu", "я", "ya",
	)
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func trimToLimit(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}

	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit]))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeKey(value string) string {
	return strings.ToLower(compactWhitespace(value))
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func pruneResolvedMissingFields(items []string, draft GeneratedProductCardDraft) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		normalized := normalizeKey(item)
		switch {
		case strings.Contains(normalized, "бренд") && draft.Brand != "":
			continue
		case (strings.Contains(normalized, "единица") || strings.Contains(normalized, "unit")) && draft.Unit != "":
			continue
		case strings.Contains(normalized, "описан") && draft.Description != "":
			continue
		case strings.Contains(normalized, "назван") && draft.Name != "":
			continue
		case strings.Contains(normalized, "характерист") && len(draft.Specs) > 0:
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func containsCyrillic(value string) bool {
	for _, r := range strings.ToLower(value) {
		if r >= 'а' && r <= 'я' || r == 'ё' {
			return true
		}
	}
	return false
}
