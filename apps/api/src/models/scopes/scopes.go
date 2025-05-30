package scopes

import "gorm.io/gorm"

func WithID(id uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func WithIDs(ids ...uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id IN (?)", ids)
	}
}

func WithPendingStatus(db *gorm.DB) *gorm.DB {
	return db.Where("status = ?", "pending")
}
