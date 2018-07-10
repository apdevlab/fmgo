package request

// SubscribeRequest model
type SubscribeRequest struct {
	Requestor string `json:"requestor" binding:"required,email"`
	Target    string `json:"target" binding:"required,email"`
}
