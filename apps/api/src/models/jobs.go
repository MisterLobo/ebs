package models

import (
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobTask struct {
	ID uuid.UUID `gorm:"primarykey;type:uuid;default:gen_random_uuid()" json:"id"`

	Name          string      `json:"-"`
	JobType       string      `json:"-"`
	RunsAt        time.Time   `json:"-"`
	HandlerParams []any       `gorm:"type:jsonb" json:"-"`
	PayloadID     string      `json:"-"`
	Payload       types.JSONB `gorm:"type:jsonb" json:"-"`
	Source        string      `json:"-"`
	SourceType    string      `json:"-"`
	Status        string      `gorm:"default:'pending'" json:"-"`
	Topic         string      `json:"-"`
}

func (j *JobTask) CreateAndEnqueueJobTask(jobTask JobTask) (string, error) {
	var jobID string
	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		id := jobTask.HandlerParams[0]
		clientId := jobTask.Payload["producerClientId"].(string)
		params := map[string]string{
			"name":     jobTask.Name,
			"clientId": clientId,
			"topic":    jobTask.Topic,
		}
		sid, err := lib.NewScheduledJob(jobTask.RunsAt, params, jobTask.Payload)
		if err != nil {
			log.Printf("Error creating job for %s: id=%d error=%s\n", jobTask.Source, id, err.Error())
			return err
		}
		jobID = sid.String()
		jobTask.ID = *sid
		jobTask.Payload["JobID"] = jobID
		err = tx.Create(&jobTask).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	log.Printf("Created schedule for job %s with name %s at %s\n", jobID, jobTask.Name, jobTask.RunsAt)
	return jobID, nil
}
