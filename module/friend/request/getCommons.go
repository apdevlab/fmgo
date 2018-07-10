package request

// GetCommonsRequest model
type GetCommonsRequest struct {
	Friends []string `json:"friends" binding:"required,len=2"`
}
