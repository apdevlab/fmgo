package request

// GetFriendRequests model
type GetFriendRequests struct {
	Email string `json:"email" binding:"required,email"`
}
