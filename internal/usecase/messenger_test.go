package usecase

import (
	"context"
	"testing"
	"time"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type messengerProductRepoMock struct {
	products map[uuid.UUID]domain.Product
}

func (m *messengerProductRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	product, ok := m.products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return product, nil
}

type messengerConversationRepoMock struct {
	conversations  map[uuid.UUID]domain.Conversation
	lastStartInput ConversationStartWriteInput
	startResult    domain.Conversation
	listResult     domain.PageResult[domain.Conversation]
	unreadCount    int
}

func (m *messengerConversationRepoMock) Start(ctx context.Context, input ConversationStartWriteInput) (domain.Conversation, error) {
	m.lastStartInput = input
	if m.startResult.ID != uuid.Nil {
		return m.startResult, nil
	}
	conversation := domain.Conversation{
		ID:                 uuid.New(),
		ProductID:          input.ProductID,
		ProductName:        "Phone",
		SellerID:           input.SellerID,
		SellerName:         "Seller",
		SellerStoreName:    "Store",
		BuyerID:            input.BuyerID,
		BuyerName:          "Buyer",
		LastMessageAt:      time.Now().UTC(),
		LastMessagePreview: input.InitialBody,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	m.conversations[conversation.ID] = conversation
	return conversation, nil
}

func (m *messengerConversationRepoMock) ListForUser(ctx context.Context, userID uuid.UUID, filter domain.ConversationFilter) (domain.PageResult[domain.Conversation], error) {
	return m.listResult, nil
}

func (m *messengerConversationRepoMock) GetByID(ctx context.Context, id, currentUserID uuid.UUID) (domain.Conversation, error) {
	conversation, ok := m.conversations[id]
	if !ok {
		return domain.Conversation{}, domain.ErrNotFound
	}
	return conversation, nil
}

func (m *messengerConversationRepoMock) GetUnreadCount(ctx context.Context, userID uuid.UUID, role domain.ConversationRole) (int, error) {
	return m.unreadCount, nil
}

type messengerMessageRepoMock struct {
	lastSendInput ConversationMessageWriteInput
	sendResult    domain.ConversationMessage
	latest        domain.ConversationMessage
}

func (m *messengerMessageRepoMock) List(ctx context.Context, conversationID uuid.UUID, filter domain.ConversationMessageFilter) (domain.PageResult[domain.ConversationMessage], error) {
	return domain.PageResult[domain.ConversationMessage]{}, nil
}

func (m *messengerMessageRepoMock) Send(ctx context.Context, input ConversationMessageWriteInput) (domain.ConversationMessage, error) {
	m.lastSendInput = input
	if m.sendResult.ID != uuid.Nil {
		return m.sendResult, nil
	}
	return domain.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		Body:           input.Body,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

func (m *messengerMessageRepoMock) GetLatest(ctx context.Context, conversationID uuid.UUID) (domain.ConversationMessage, error) {
	if m.latest.ID == uuid.Nil {
		return domain.ConversationMessage{}, domain.ErrNotFound
	}
	return m.latest, nil
}

type messengerReadStateRepoMock struct {
	lastInput  ConversationReadWriteInput
	markResult domain.ConversationReadState
}

func (m *messengerReadStateRepoMock) Mark(ctx context.Context, input ConversationReadWriteInput) (domain.ConversationReadState, error) {
	m.lastInput = input
	if m.markResult.ConversationID != uuid.Nil {
		return m.markResult, nil
	}
	return domain.ConversationReadState{
		ConversationID:    input.ConversationID,
		UserID:            input.UserID,
		LastReadMessageID: input.LastReadMessageID,
		LastReadAt:        input.LastReadAt,
	}, nil
}

func TestMessengerServiceStartConversation(t *testing.T) {
	buyerID := uuid.New()
	sellerID := uuid.New()
	productID := uuid.New()

	productRepo := &messengerProductRepoMock{
		products: map[uuid.UUID]domain.Product{
			productID: {
				ID:       productID,
				Name:     "Phone",
				SellerID: &sellerID,
			},
		},
	}
	conversationRepo := &messengerConversationRepoMock{conversations: map[uuid.UUID]domain.Conversation{}}
	messageRepo := &messengerMessageRepoMock{}
	readRepo := &messengerReadStateRepoMock{}

	service := NewMessengerService(productRepo, conversationRepo, messageRepo, readRepo)

	conversation, err := service.StartConversation(context.Background(), buyerID, productID, "  Interested in pickup today?  ")
	require.NoError(t, err)
	assert.Equal(t, productID, conversation.ProductID)
	assert.Equal(t, buyerID, conversationRepo.lastStartInput.BuyerID)
	assert.Equal(t, sellerID, conversationRepo.lastStartInput.SellerID)
	assert.Equal(t, "Interested in pickup today?", conversationRepo.lastStartInput.InitialBody)
}

func TestMessengerServiceStartConversationForbidsSelfChat(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()

	productRepo := &messengerProductRepoMock{
		products: map[uuid.UUID]domain.Product{
			productID: {
				ID:       productID,
				Name:     "Phone",
				SellerID: &userID,
			},
		},
	}
	service := NewMessengerService(
		productRepo,
		&messengerConversationRepoMock{conversations: map[uuid.UUID]domain.Conversation{}},
		&messengerMessageRepoMock{},
		&messengerReadStateRepoMock{},
	)

	_, err := service.StartConversation(context.Background(), userID, productID, "Hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMessengerServiceSendMessage(t *testing.T) {
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversationID := uuid.New()

	conversationRepo := &messengerConversationRepoMock{
		conversations: map[uuid.UUID]domain.Conversation{
			conversationID: {
				ID:       conversationID,
				SellerID: sellerID,
				BuyerID:  buyerID,
			},
		},
	}
	messageRepo := &messengerMessageRepoMock{}
	service := NewMessengerService(
		&messengerProductRepoMock{products: map[uuid.UUID]domain.Product{}},
		conversationRepo,
		messageRepo,
		&messengerReadStateRepoMock{},
	)

	message, err := service.SendMessage(context.Background(), buyerID, conversationID, "  See you at 18:00  ")
	require.NoError(t, err)
	assert.Equal(t, conversationID, message.ConversationID)
	assert.Equal(t, buyerID, messageRepo.lastSendInput.SenderID)
	assert.Equal(t, "See you at 18:00", messageRepo.lastSendInput.Body)
	assert.Equal(t, "See you at 18:00", messageRepo.lastSendInput.Preview)
}

func TestMessengerServiceForbiddenAccessToForeignConversation(t *testing.T) {
	strangerID := uuid.New()
	conversationID := uuid.New()

	conversationRepo := &messengerConversationRepoMock{
		conversations: map[uuid.UUID]domain.Conversation{
			conversationID: {
				ID:       conversationID,
				SellerID: uuid.New(),
				BuyerID:  uuid.New(),
			},
		},
	}
	service := NewMessengerService(
		&messengerProductRepoMock{products: map[uuid.UUID]domain.Product{}},
		conversationRepo,
		&messengerMessageRepoMock{},
		&messengerReadStateRepoMock{},
	)

	_, err := service.GetConversation(context.Background(), strangerID, conversationID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)

	_, err = service.SendMessage(context.Background(), strangerID, conversationID, "Hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}
