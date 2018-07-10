package request

// GetNotificationRequest model
type GetNotificationRequest struct {
	Sender string `json:"sender" binding:"required,email"`
	Text   string `json:"text"`
}
