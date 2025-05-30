package models

type Role struct {
	Name string `gorm:"primarykey" json:"name"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"-"`
}

type Permission struct {
	Name string `gorm:"primarykey" json:"name"`

	Role Role `gorm:"many2many:role_permissions;" json:"-"`
}

type RolePermission struct {
	ID         uint   `json:"id"`
	Role       string `gorm:"uniqueIndex:role_permission" json:"role"`
	Permission string `gorm:"uniqueIndex:role_permission" json:"permission"`

	InnerRole       Role       `gorm:"foreignKey:role" json:"-"`
	InnerPermission Permission `gorm:"foreignKey:permission" json:"-"`
}
