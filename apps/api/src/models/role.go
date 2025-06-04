package models

type Role struct {
	Name string `gorm:"primarykey" json:"name"`

	Permissions []*Permission `gorm:"many2many:role_permissions;" json:"-"`
}

type Permission struct {
	Name string `gorm:"primarykey" json:"name"`

	Role []*Role `gorm:"many2many:role_permissions;" json:"-"`
}

type RolePermission struct {
	RoleName       string `gorm:"primaryKey" json:"role"`
	PermissionName string `gorm:"primaryKey" json:"permission"`
}
