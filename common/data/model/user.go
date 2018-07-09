package model

// User data model
type User struct {
	BaseModel
	Email   string  `gorm:"type:varchar(100);unique_index;not null"`
	Friends []*User `gorm:"many2many:friends;association_jointable_foreignkey:friend_id"`
}
