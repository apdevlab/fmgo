package friend

import (
	"fmgo/common/data"
	"fmgo/common/data/model"
	"fmgo/module/friend/request"
	"fmgo/module/friend/response"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang/glog"
	"gopkg.in/go-playground/validator.v8"
)

// Controller struct
type Controller struct {
	dbFactory *data.DBFactory
}

// NewController initialize new Friend Controller instance
func NewController(dbFactory *data.DBFactory) *Controller {
	return &Controller{dbFactory: dbFactory}
}

// Connect action to create friend connection between two user
func (ctrl *Controller) Connect(c *gin.Context) {
	// deserialize and validate POST data
	var req request.ConnectRequest
	var errors []string
	if err := c.ShouldBindWith(&req, binding.JSON); err != nil {
		ve, ok := err.(validator.ValidationErrors)
		if ok {
			for _, v := range ve {
				msg := fmt.Sprintf("%s is %s", v.Field, v.Tag)
				if v.Tag == "len" {
					msg = fmt.Sprintf("%s %s should be %s", v.Field, v.Tag, v.Param)
				}
				errors = append(errors, msg)
			}
		} else {
			errors = append(errors, err.Error())
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": errors})
		return
	}

	if strings.ToLower(strings.TrimSpace(req.Friends[0])) == strings.ToLower(strings.TrimSpace(req.Friends[1])) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Could not connect same email"}})
		return
	}

	// validate email format
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	for _, email := range req.Friends {
		if !re.MatchString(email) {
			errors = append(errors, fmt.Sprintf("%s is an invalid email format", email))
		}
	}
	if len(errors) > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": errors})
		return
	}

	db, err := ctrl.dbFactory.DBConnection()
	if err != nil {
		fmt.Println("err")
		glog.Errorf("Failed to open db connection: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to open db connection"}})
		return
	}
	defer db.Close()

	tx := db.Begin()
	if tx.Error != nil {
		glog.Errorf("Failed to create new db transaction: %s", tx.Error)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to start new db transaction"}})
		return
	}

	// create user if email does not exist yet
	var user1, user2 model.User
	normalizeEmail1 := strings.ToLower(req.Friends[0])
	if tx.Preload("Friends").First(&user1, "email = ?", normalizeEmail1).RecordNotFound() {
		user1 = model.User{Email: normalizeEmail1}
		if err := tx.Create(&user1).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeEmail1, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	normalizeEmail2 := strings.ToLower(req.Friends[1])
	if tx.First(&user2, "email = ?", normalizeEmail2).RecordNotFound() {
		user2 = model.User{Email: normalizeEmail2}
		if err := tx.Create(&user2).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeEmail2, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	if err := tx.Model(&user1).Association("Friends").Find(&user2).Error; err != nil && err.Error() == "record not found" {
		tx.Model(&user1).Association("Friends").Append(&user2)
		tx.Model(&user2).Association("Friends").Append(&user1)
	}

	if err := tx.Commit().Error; err != nil {
		glog.Errorf("Failed to commit db transaction: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to commit db transaction"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetFriends action to get friend list for given email address
func (ctrl *Controller) GetFriends(c *gin.Context) {
	// deserialize and validate POST data
	var req request.GetFriendRequests
	var errors []string
	if err := c.ShouldBindWith(&req, binding.JSON); err != nil {
		ve, ok := err.(validator.ValidationErrors)
		if ok {
			for _, v := range ve {
				msg := fmt.Sprintf("%s is %s", v.Field, v.Tag)
				if v.Tag == "email" {
					msg = fmt.Sprintf("%s is invalid", v.Field)
				}
				errors = append(errors, msg)
			}
		} else {
			errors = append(errors, err.Error())
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": errors})
		return
	}

	db, err := ctrl.dbFactory.DBConnection()
	if err != nil {
		fmt.Println("err")
		glog.Errorf("Failed to open db connection: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to open db connection"}})
		return
	}
	defer db.Close()

	var user model.User
	if db.Preload("Friends").First(&user, "email = ?", strings.ToLower(req.Email)).RecordNotFound() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"success": false, "errors": []string{fmt.Sprintf("User with email %s does not exist", req.Email)}})
		return
	}

	friends := make([]string, 0)
	for _, friend := range user.Friends {
		friends = append(friends, friend.Email)
	}

	resp := response.FriendListResponse{
		Success: true,
		Friends: friends,
		Count:   len(friends),
	}
	c.JSON(http.StatusOK, resp)
}
