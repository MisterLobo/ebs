package models

import (
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/types"
	"encoding/json"
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

func (self *JobTask) CreateAndEnqueueJobTask(jobTask JobTask) (string, error) {
	var jobID string
	db := db.GetDb()
	now := time.Now().UTC()
	in5m := now.Add(5 * time.Minute)
	strnow := in5m.Format("2006-01-02T15:04:05")
	err := db.Transaction(func(tx *gorm.DB) error {
		eventId := jobTask.HandlerParams[0]
		pBytes, err := json.Marshal(jobTask.Payload)
		if err != nil {
			log.Printf("Failed to marshal payload: %s\n", err.Error())
			return err
		}
		sRunsAt := jobTask.RunsAt.Format("2006-01-02T15:04:05")
		sPayload := string(pBytes)
		sid, err := lib.CreateSchedule(jobTask.Name, jobTask.RunsAt, sRunsAt, jobTask.Topic, sPayload)
		if err != nil {
			log.Printf("Error creating job for Event: id=%d error=%s\n", eventId, err.Error())
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
	log.Printf("Created schedule for job %s with name %s at %s\n", jobID, jobTask.Name, strnow)
	return jobID, nil
}
