package request

// ConnectRequest model
type ConnectRequest struct {
	Friends []string `json:"friends" binding:"required,len=2"`
}
