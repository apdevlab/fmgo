package notification

import (
	"fmgo/common/data"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller struct
type Controller struct {
	dbFactory *data.DBFactory
}

// NewController initialize new Friend Controller instance
func NewController(dbFactory *data.DBFactory) *Controller {
	return &Controller{dbFactory: dbFactory}
}

// Subscribe action to get notification
func (ctrl *Controller) Subscribe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}
