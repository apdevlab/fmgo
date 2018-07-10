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
	if tx.First(&user1, "email = ?", normalizeEmail1).RecordNotFound() {
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
		// If one of them or both blocked each other, then friend connection will fail
		if tx.Model(&user1).Association("Blocks").Find(&user2).Error == nil || tx.Model(&user2).Association("Blocks").Find(&user1).Error == nil {
			tx.Rollback()
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Friend connection are being blocked"}})
			return
		}

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

// GetCommons action to get commond friend list
func (ctrl *Controller) GetCommons(c *gin.Context) {
	// deserialize and validate POST data
	var req request.GetCommonsRequest
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Could not get common friend list from same email address"}})
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

	var user1, user2 model.User
	normalizeEmail1 := strings.ToLower(req.Friends[0])
	if db.Preload("Friends").First(&user1, "email = ?", normalizeEmail1).RecordNotFound() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"success": false, "errors": []string{fmt.Sprintf("User with email %s does not exist", normalizeEmail1)}})
		return
	}

	normalizeEmail2 := strings.ToLower(req.Friends[1])
	if db.Preload("Friends").First(&user2, "email = ?", normalizeEmail2).RecordNotFound() {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"success": false, "errors": []string{fmt.Sprintf("User with email %s does not exist", normalizeEmail2)}})
		return
	}

	friends1 := make([]string, 0)
	for _, friend := range user1.Friends {
		friends1 = append(friends1, friend.Email)
	}

	friends2 := make([]string, 0)
	for _, friend := range user2.Friends {
		friends2 = append(friends2, friend.Email)
	}

	intersect := intersection(friends1, friends2)
	resp := response.FriendListResponse{
		Success: true,
		Friends: intersect,
		Count:   len(intersect),
	}
	c.JSON(http.StatusOK, resp)
}

func intersection(a []string, b []string) []string {
	result := make([]string, 0)

	// interacting on the smallest list first can potentailly be faster...but not by much, worse case is the same
	low, high := a, b
	if len(a) > len(b) {
		low = b
		high = a
	}

	done := false
	for i, l := range low {
		for j, h := range high {
			// get future index values
			f1 := i + 1
			f2 := j + 1
			if l == h {
				result = append(result, h)
				if f1 < len(low) && f2 < len(high) {
					// if the future values aren't the same then that's the end of the intersection
					if low[f1] != high[f2] {
						done = true
					}
				}
				// we don't want to interate on the entire list everytime, so remove the parts we already looped on will make it faster each pass
				high = high[:j+copy(high[j:], high[j+1:])]
				break
			}
		}
		// nothing in the future so we are done
		if done {
			break
		}
	}

	return result
}
