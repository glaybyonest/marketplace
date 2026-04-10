package dto

type CreateConversationRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Message   string `json:"message" validate:"omitempty,max=2000"`
}

type SendMessageRequest struct {
	Body string `json:"body" validate:"required,max=2000"`
}

type ConversationResponse struct {
	ID                 string `json:"id"`
	ProductID          string `json:"product_id"`
	ProductName        string `json:"product_name"`
	ProductImageURL    string `json:"product_image_url,omitempty"`
	SellerID           string `json:"seller_id"`
	SellerName         string `json:"seller_name"`
	SellerStoreName    string `json:"seller_store_name,omitempty"`
	BuyerID            string `json:"buyer_id"`
	BuyerName          string `json:"buyer_name"`
	OrderID            string `json:"order_id,omitempty"`
	LastMessageAt      string `json:"last_message_at"`
	LastMessagePreview string `json:"last_message_preview,omitempty"`
	UnreadCount        int    `json:"unread_count"`
	CurrentUserRole    string `json:"current_user_role,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type ConversationListResponse struct {
	Items []ConversationResponse `json:"items"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
	Total int                    `json:"total"`
}

type MessageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Body           string  `json:"body"`
	CreatedAt      string  `json:"created_at"`
	EditedAt       *string `json:"edited_at,omitempty"`
}

type MessageListResponse struct {
	Items []MessageResponse `json:"items"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Total int               `json:"total"`
}

type ConversationReadStateResponse struct {
	ConversationID    string  `json:"conversation_id"`
	UserID            string  `json:"user_id"`
	LastReadMessageID *string `json:"last_read_message_id,omitempty"`
	LastReadAt        *string `json:"last_read_at,omitempty"`
}

type UnreadCountResponse struct {
	Total int `json:"total"`
}
