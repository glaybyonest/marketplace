package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationRole string

const (
	ConversationRoleBuyer  ConversationRole = "buyer"
	ConversationRoleSeller ConversationRole = "seller"
	ConversationRoleAll    ConversationRole = "all"
)

type Conversation struct {
	ID                 uuid.UUID        `json:"id"`
	ProductID          uuid.UUID        `json:"product_id"`
	ProductName        string           `json:"product_name"`
	ProductImageURL    string           `json:"product_image_url,omitempty"`
	SellerID           uuid.UUID        `json:"seller_id"`
	SellerName         string           `json:"seller_name"`
	SellerStoreName    string           `json:"seller_store_name,omitempty"`
	BuyerID            uuid.UUID        `json:"buyer_id"`
	BuyerName          string           `json:"buyer_name"`
	OrderID            *uuid.UUID       `json:"order_id,omitempty"`
	LastMessageAt      time.Time        `json:"last_message_at"`
	LastMessagePreview string           `json:"last_message_preview,omitempty"`
	UnreadCount        int              `json:"unread_count"`
	CurrentUserRole    ConversationRole `json:"current_user_role,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

func (c Conversation) HasParticipant(userID uuid.UUID) bool {
	return c.BuyerID == userID || c.SellerID == userID
}

func (c Conversation) CounterpartyName(userID uuid.UUID) string {
	if c.BuyerID == userID {
		if c.SellerStoreName != "" {
			return c.SellerStoreName
		}
		return c.SellerName
	}
	return c.BuyerName
}

type ConversationMessage struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"created_at"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
}

type ConversationReadState struct {
	ConversationID    uuid.UUID  `json:"conversation_id"`
	UserID            uuid.UUID  `json:"user_id"`
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
	LastReadAt        *time.Time `json:"last_read_at,omitempty"`
}

type ConversationFilter struct {
	Role  ConversationRole
	Page  int
	Limit int
}

type ConversationMessageFilter struct {
	Page  int
	Limit int
}
