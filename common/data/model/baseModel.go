package model

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

// BaseModel base model definition for common entity's field
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:char(36); primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate gorm callback
func (m *BaseModel) BeforeCreate(scope *gorm.Scope) error {
	if m.ID == uuid.Nil {
		return scope.SetColumn("ID", uuid.NewV4())
	}

	return nil
}
