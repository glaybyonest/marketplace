package usecase

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

const (
	messengerMaxMessageLength = 2000
	messengerPreviewLength    = 160
)

type MessengerProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
}

type MessengerConversationRepository interface {
	Start(ctx context.Context, input ConversationStartWriteInput) (domain.Conversation, error)
	ListForUser(ctx context.Context, userID uuid.UUID, filter domain.ConversationFilter) (domain.PageResult[domain.Conversation], error)
	GetByID(ctx context.Context, id, currentUserID uuid.UUID) (domain.Conversation, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID, role domain.ConversationRole) (int, error)
}

type MessengerMessageRepository interface {
	List(ctx context.Context, conversationID uuid.UUID, filter domain.ConversationMessageFilter) (domain.PageResult[domain.ConversationMessage], error)
	Send(ctx context.Context, input ConversationMessageWriteInput) (domain.ConversationMessage, error)
	GetLatest(ctx context.Context, conversationID uuid.UUID) (domain.ConversationMessage, error)
}

type MessengerReadStateRepository interface {
	Mark(ctx context.Context, input ConversationReadWriteInput) (domain.ConversationReadState, error)
}

type ConversationStartWriteInput struct {
	ProductID   uuid.UUID
	SellerID    uuid.UUID
	BuyerID     uuid.UUID
	OrderID     *uuid.UUID
	InitialBody string
}

type ConversationMessageWriteInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	Preview        string
}

type ConversationReadWriteInput struct {
	ConversationID    uuid.UUID
	UserID            uuid.UUID
	LastReadMessageID *uuid.UUID
	LastReadAt        *time.Time
}

type MessengerService struct {
	products      MessengerProductRepository
	conversations MessengerConversationRepository
	messages      MessengerMessageRepository
	reads         MessengerReadStateRepository
}

func NewMessengerService(
	products MessengerProductRepository,
	conversations MessengerConversationRepository,
	messages MessengerMessageRepository,
	reads MessengerReadStateRepository,
) *MessengerService {
	return &MessengerService{
		products:      products,
		conversations: conversations,
		messages:      messages,
		reads:         reads,
	}
}

func (s *MessengerService) StartConversation(ctx context.Context, userID, productID uuid.UUID, initialMessage string) (domain.Conversation, error) {
	if userID == uuid.Nil {
		return domain.Conversation{}, domain.ErrUnauthorized
	}
	if productID == uuid.Nil {
		return domain.Conversation{}, domain.ErrInvalidInput
	}

	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if product.SellerID == nil || *product.SellerID == uuid.Nil {
		return domain.Conversation{}, domain.ErrInvalidInput
	}
	if *product.SellerID == userID {
		return domain.Conversation{}, domain.ErrForbidden
	}

	body, err := normalizeMessengerMessage(initialMessage, true)
	if err != nil {
		return domain.Conversation{}, err
	}

	return s.conversations.Start(ctx, ConversationStartWriteInput{
		ProductID:   product.ID,
		SellerID:    *product.SellerID,
		BuyerID:     userID,
		InitialBody: body,
	})
}

func (s *MessengerService) ListConversations(ctx context.Context, userID uuid.UUID, filter domain.ConversationFilter) (domain.PageResult[domain.Conversation], error) {
	if userID == uuid.Nil {
		return domain.PageResult[domain.Conversation]{}, domain.ErrUnauthorized
	}

	normalized, err := normalizeConversationFilter(filter)
	if err != nil {
		return domain.PageResult[domain.Conversation]{}, err
	}

	return s.conversations.ListForUser(ctx, userID, normalized)
}

func (s *MessengerService) GetConversation(ctx context.Context, userID, conversationID uuid.UUID) (domain.Conversation, error) {
	if userID == uuid.Nil {
		return domain.Conversation{}, domain.ErrUnauthorized
	}
	if conversationID == uuid.Nil {
		return domain.Conversation{}, domain.ErrInvalidInput
	}

	conversation, err := s.conversations.GetByID(ctx, conversationID, userID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if !conversation.HasParticipant(userID) {
		return domain.Conversation{}, domain.ErrForbidden
	}
	return conversation, nil
}

func (s *MessengerService) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, filter domain.ConversationMessageFilter) (domain.PageResult[domain.ConversationMessage], error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return domain.PageResult[domain.ConversationMessage]{}, err
	}

	normalized := normalizeConversationMessageFilter(filter)
	return s.messages.List(ctx, conversationID, normalized)
}

func (s *MessengerService) SendMessage(ctx context.Context, userID, conversationID uuid.UUID, body string) (domain.ConversationMessage, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return domain.ConversationMessage{}, err
	}
	if !conversation.HasParticipant(userID) {
		return domain.ConversationMessage{}, domain.ErrForbidden
	}

	normalizedBody, err := normalizeMessengerMessage(body, false)
	if err != nil {
		return domain.ConversationMessage{}, err
	}

	return s.messages.Send(ctx, ConversationMessageWriteInput{
		ConversationID: conversationID,
		SenderID:       userID,
		Body:           normalizedBody,
		Preview:        truncatePreview(normalizedBody),
	})
}

func (s *MessengerService) MarkAsRead(ctx context.Context, userID, conversationID uuid.UUID) (domain.ConversationReadState, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return domain.ConversationReadState{}, err
	}

	latestMessage, err := s.messages.GetLatest(ctx, conversationID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.ConversationReadState{}, err
	}

	markInput := ConversationReadWriteInput{
		ConversationID: conversationID,
		UserID:         userID,
	}

	if errors.Is(err, domain.ErrNotFound) {
		now := time.Now().UTC()
		markInput.LastReadAt = &now
		return s.reads.Mark(ctx, markInput)
	}

	markInput.LastReadMessageID = &latestMessage.ID
	markInput.LastReadAt = &latestMessage.CreatedAt
	return s.reads.Mark(ctx, markInput)
}

func (s *MessengerService) GetUnreadCount(ctx context.Context, userID uuid.UUID, role domain.ConversationRole) (int, error) {
	if userID == uuid.Nil {
		return 0, domain.ErrUnauthorized
	}

	normalizedRole, err := normalizeConversationRole(role)
	if err != nil {
		return 0, err
	}

	return s.conversations.GetUnreadCount(ctx, userID, normalizedRole)
}

func normalizeConversationFilter(filter domain.ConversationFilter) (domain.ConversationFilter, error) {
	role, err := normalizeConversationRole(filter.Role)
	if err != nil {
		return domain.ConversationFilter{}, err
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return domain.ConversationFilter{
		Role:  role,
		Page:  page,
		Limit: limit,
	}, nil
}

func normalizeConversationMessageFilter(filter domain.ConversationMessageFilter) domain.ConversationMessageFilter {
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	return domain.ConversationMessageFilter{
		Page:  page,
		Limit: limit,
	}
}

func normalizeConversationRole(role domain.ConversationRole) (domain.ConversationRole, error) {
	switch domain.ConversationRole(strings.TrimSpace(string(role))) {
	case "", domain.ConversationRoleAll:
		return domain.ConversationRoleAll, nil
	case domain.ConversationRoleBuyer:
		return domain.ConversationRoleBuyer, nil
	case domain.ConversationRoleSeller:
		return domain.ConversationRoleSeller, nil
	default:
		return "", domain.ErrInvalidInput
	}
}

func normalizeMessengerMessage(body string, allowEmpty bool) (string, error) {
	normalized := strings.TrimSpace(body)
	if normalized == "" {
		if allowEmpty {
			return "", nil
		}
		return "", domain.ErrInvalidInput
	}
	if utf8.RuneCountInString(normalized) > messengerMaxMessageLength {
		return "", domain.ErrInvalidInput
	}
	return normalized, nil
}

func truncatePreview(body string) string {
	if utf8.RuneCountInString(body) <= messengerPreviewLength {
		return body
	}
	runes := []rune(body)
	return string(runes[:messengerPreviewLength])
}
