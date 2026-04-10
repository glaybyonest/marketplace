package dto

type SellerAIProductCardDraft struct {
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	Brand       string            `json:"brand"`
	Unit        string            `json:"unit"`
	Specs       map[string]string `json:"specs"`
}

type SellerAIProductCardRequest struct {
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
	ExistingDraft  *SellerAIProductCardDraft `json:"existing_draft"`
}
