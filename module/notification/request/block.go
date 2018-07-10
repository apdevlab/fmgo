package request

// BlockRequest model
type BlockRequest struct {
	Requestor string `json:"requestor" binding:"required,email"`
	Target    string `json:"target" binding:"required,email"`
}
