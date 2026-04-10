package openaiintegration

import (
	"context"
	"net/http"
	"strings"
	"time"

	"marketplace-backend/internal/usecase"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const providerNameOpenAI = "openai"

type Client struct {
	client          openai.Client
	model           string
	timeout         time.Duration
	maxOutputTokens int
}

func NewClient(apiKey, model string, timeout time.Duration, maxOutputTokens int) *Client {
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	if apiKey == "" || model == "" || timeout <= 0 || maxOutputTokens <= 0 {
		return nil
	}

	return &Client{
		client: openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithHTTPClient(&http.Client{Timeout: timeout}),
		),
		model:           model,
		timeout:         timeout,
		maxOutputTokens: maxOutputTokens,
	}
}

func (c *Client) GenerateStructuredJSON(ctx context.Context, request usecase.StructuredJSONRequest) (usecase.StructuredJSONResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.Responses.New(callCtx, responses.ResponseNewParams{
		Input:             responses.ResponseNewParamsInputUnion{OfString: openai.String(request.Input)},
		Instructions:      openai.String(request.Instructions),
		MaxOutputTokens:   openai.Int(int64(c.maxOutputTokens)),
		Model:             c.model,
		Store:             openai.Bool(false),
		Temperature:       openai.Float(0.2),
		ParallelToolCalls: openai.Bool(false),
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:        request.SchemaName,
					Schema:      request.Schema,
					Strict:      openai.Bool(true),
					Description: openai.String("Structured seller product card draft response"),
				},
			},
		},
	})
	if err != nil {
		return usecase.StructuredJSONResult{}, err
	}

	return usecase.StructuredJSONResult{
		Output:   response.OutputText(),
		Provider: providerNameOpenAI,
		Model:    c.model,
	}, nil
}
