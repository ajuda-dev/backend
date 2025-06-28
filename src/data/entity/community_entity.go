package entity

import "gorm.io/gorm"

type CommunityEntity struct {
	gorm.Model
	Id          uint  `gorm:"primaryKey;"`
	Name        string `gorm:"not null;unique"`
	Description string `gorm:"not null"`
	OwnerId     uint   `gorm:"not null"`
	
	Owner       UserEntity `gorm:"foreignKey:OwnerId;references:Id"`
}


func (c *CommunityEntity) TableName() string {
	return "communities"
}
