package common

import (
	"ebs/src/db"
	"ebs/src/models"
	"fmt"
	"log"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

func UpdateMissingSlugs() {
	db := db.GetDb()
	rows, err := db.
		Model(&models.Organization{}).
		Where("slug IS NULL").
		Rows()
	if err != nil {
		log.Printf("Error querying Organizations: %s\n", err.Error())
		return
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for rows.Next() {
			log.Println("Scanning rows...")
			var org models.Organization
			if err := tx.ScanRows(rows, &org); err != nil {
				return err
			}
			log.Println("Update org...")
			newSlug := slug.Make(org.Name)
			if err := tx.
				Model(&models.Organization{}).
				Where("id = ?", org.ID).
				Updates(&models.Organization{Slug: fmt.Sprintf("%s-%d", newSlug, org.ID)}).
				Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("Error on update operation: %s\n", err.Error())
	}
}
