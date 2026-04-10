package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type MessengerService interface {
	StartConversation(ctx context.Context, userID, productID uuid.UUID, initialMessage string) (domain.Conversation, error)
	ListConversations(ctx context.Context, userID uuid.UUID, filter domain.ConversationFilter) (domain.PageResult[domain.Conversation], error)
	GetConversation(ctx context.Context, userID, conversationID uuid.UUID) (domain.Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID uuid.UUID, filter domain.ConversationMessageFilter) (domain.PageResult[domain.ConversationMessage], error)
	SendMessage(ctx context.Context, userID, conversationID uuid.UUID, body string) (domain.ConversationMessage, error)
	MarkAsRead(ctx context.Context, userID, conversationID uuid.UUID) (domain.ConversationReadState, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID, role domain.ConversationRole) (int, error)
}

type MessengerHandler struct {
	service  MessengerService
	validate *validator.Validate
}

func NewMessengerHandler(service MessengerService) *MessengerHandler {
	return &MessengerHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *MessengerHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	result, err := h.service.ListConversations(r.Context(), userID, domain.ConversationFilter{
		Role:  domain.ConversationRole(strings.TrimSpace(r.URL.Query().Get("role"))),
		Page:  parseIntWithDefault(r.URL.Query().Get("page"), 1),
		Limit: parseIntWithDefault(r.URL.Query().Get("limit"), 20),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ConversationListResponse{
		Items: mapConversationResponses(result.Items),
		Page:  result.Page,
		Limit: result.Limit,
		Total: result.Total,
	})
}

func (h *MessengerHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.CreateConversationRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(req.ProductID))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	conversation, err := h.service.StartConversation(r.Context(), userID, productID, req.Message)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, toConversationResponse(conversation))
}

func (h *MessengerHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	conversationID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	conversation, err := h.service.GetConversation(r.Context(), userID, conversationID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, toConversationResponse(conversation))
}

func (h *MessengerHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	conversationID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	result, err := h.service.ListMessages(r.Context(), userID, conversationID, domain.ConversationMessageFilter{
		Page:  parseIntWithDefault(r.URL.Query().Get("page"), 1),
		Limit: parseIntWithDefault(r.URL.Query().Get("limit"), 50),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.MessageListResponse{
		Items: mapMessageResponses(result.Items),
		Page:  result.Page,
		Limit: result.Limit,
		Total: result.Total,
	})
}

func (h *MessengerHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	conversationID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var req dto.SendMessageRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	message, err := h.service.SendMessage(r.Context(), userID, conversationID, req.Body)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, toMessageResponse(message))
}

func (h *MessengerHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	conversationID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	readState, err := h.service.MarkAsRead(r.Context(), userID, conversationID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, toConversationReadStateResponse(readState))
}

func (h *MessengerHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	total, err := h.service.GetUnreadCount(r.Context(), userID, domain.ConversationRole(strings.TrimSpace(r.URL.Query().Get("role"))))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.UnreadCountResponse{Total: total})
}

func toConversationResponse(item domain.Conversation) dto.ConversationResponse {
	responseItem := dto.ConversationResponse{
		ID:                 item.ID.String(),
		ProductID:          item.ProductID.String(),
		ProductName:        item.ProductName,
		ProductImageURL:    item.ProductImageURL,
		SellerID:           item.SellerID.String(),
		SellerName:         item.SellerName,
		SellerStoreName:    item.SellerStoreName,
		BuyerID:            item.BuyerID.String(),
		BuyerName:          item.BuyerName,
		LastMessageAt:      item.LastMessageAt.UTC().Format(time.RFC3339),
		LastMessagePreview: item.LastMessagePreview,
		UnreadCount:        item.UnreadCount,
		CurrentUserRole:    string(item.CurrentUserRole),
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          item.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if item.OrderID != nil {
		responseItem.OrderID = item.OrderID.String()
	}

	return responseItem
}

func mapConversationResponses(items []domain.Conversation) []dto.ConversationResponse {
	result := make([]dto.ConversationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toConversationResponse(item))
	}
	return result
}

func toMessageResponse(item domain.ConversationMessage) dto.MessageResponse {
	responseItem := dto.MessageResponse{
		ID:             item.ID.String(),
		ConversationID: item.ConversationID.String(),
		SenderID:       item.SenderID.String(),
		Body:           item.Body,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.EditedAt != nil {
		editedAt := item.EditedAt.UTC().Format(time.RFC3339)
		responseItem.EditedAt = &editedAt
	}
	return responseItem
}

func mapMessageResponses(items []domain.ConversationMessage) []dto.MessageResponse {
	result := make([]dto.MessageResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toMessageResponse(item))
	}
	return result
}

func toConversationReadStateResponse(item domain.ConversationReadState) dto.ConversationReadStateResponse {
	responseItem := dto.ConversationReadStateResponse{
		ConversationID: item.ConversationID.String(),
		UserID:         item.UserID.String(),
	}
	if item.LastReadMessageID != nil {
		value := item.LastReadMessageID.String()
		responseItem.LastReadMessageID = &value
	}
	if item.LastReadAt != nil {
		value := item.LastReadAt.UTC().Format(time.RFC3339)
		responseItem.LastReadAt = &value
	}
	return responseItem
}
