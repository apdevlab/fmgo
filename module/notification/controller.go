package notification

import (
	"fmgo/common/data"
	"fmgo/common/data/model"
	"fmgo/module/notification/request"
	"fmt"
	"net/http"
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

// Subscribe action to get notification
func (ctrl *Controller) Subscribe(c *gin.Context) {
	var req request.SubscribeRequest
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

	if strings.ToLower(req.Requestor) == strings.ToLower(req.Target) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Could not subscribe to self"}})
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

	var requestor, target model.User
	normalizeRequestorEmail := strings.ToLower(req.Requestor)
	if tx.First(&requestor, "email = ?", normalizeRequestorEmail).RecordNotFound() {
		requestor = model.User{Email: normalizeRequestorEmail}
		if err := tx.Create(&requestor).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeRequestorEmail, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	normalizeTargetEmail := strings.ToLower(req.Target)
	if tx.First(&target, "email = ?", normalizeTargetEmail).RecordNotFound() {
		target = model.User{Email: normalizeTargetEmail}
		if err := tx.Create(&target).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeTargetEmail, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	// If requestor and target are friends and target blocked requestor then subscription will fail
	if err := tx.Model(&requestor).Association("Friends").Find(&target).Error; err == nil {
		if err := tx.Model(&target).Association("Blocks").Find(&requestor).Error; err == nil {
			tx.Rollback()
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Requestor is being blocked by target"}})
			return
		}
	}

	if err := tx.Model(&requestor).Association("Notifications").Find(&target).Error; err != nil && err.Error() == "record not found" {
		tx.Model(&requestor).Association("Notifications").Append(&target)
	}

	if err := tx.Commit().Error; err != nil {
		glog.Errorf("Failed to commit db transaction: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to commit db transaction"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Block action to block notification and prevent further friend connection
func (ctrl *Controller) Block(c *gin.Context) {
	var req request.BlockRequest
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

	if strings.ToLower(req.Requestor) == strings.ToLower(req.Target) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "errors": []string{"Could not block self"}})
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

	var requestor, target model.User
	normalizeRequestorEmail := strings.ToLower(req.Requestor)
	if tx.First(&requestor, "email = ?", normalizeRequestorEmail).RecordNotFound() {
		requestor = model.User{Email: normalizeRequestorEmail}
		if err := tx.Create(&requestor).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeRequestorEmail, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	normalizeTargetEmail := strings.ToLower(req.Target)
	if tx.First(&target, "email = ?", normalizeTargetEmail).RecordNotFound() {
		target = model.User{Email: normalizeTargetEmail}
		if err := tx.Create(&target).Error; err != nil {
			tx.Rollback()
			glog.Errorf("Failed to create user %s: %s", normalizeTargetEmail, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to create new user"}})
			return
		}
	}

	if err := tx.Model(&requestor).Association("Blocks").Find(&target).Error; err != nil && err.Error() == "record not found" {
		tx.Model(&requestor).Association("Blocks").Append(&target)
	}

	// If requestor and target are friend, remove notification from target to requestor if any
	if err := tx.Model(&requestor).Association("Friends").Find(&target).Error; err == nil {
		tx.Model(&target).Association("Notifications").Delete(&requestor)
	}

	if err := tx.Commit().Error; err != nil {
		glog.Errorf("Failed to commit db transaction: %s", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "errors": []string{"Failed to commit db transaction"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetNotificationList action to get list of email that eligible to receive notification from given sender
func (ctrl *Controller) GetNotificationList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "recipients": []string{}})
}
