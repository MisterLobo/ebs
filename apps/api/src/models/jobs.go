package models

import (
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
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
}

func (self *JobTask) CreateAndEnqueueJobTask(jobTask JobTask) (string, error) {
	var jobID string
	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		scheduler, err := lib.GetScheduler()
		if err != nil {
			return err
		}
		eventId := jobTask.HandlerParams[0]
		job, err := scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(jobTask.RunsAt)),
			gocron.NewTask(func(id uint) {
				log.Println("Running scheduled task")
				err := lib.KafkaProduceMessage(self.Payload["producerClientId"].(string), self.Payload["topic"].(string), jobTask.Payload)
				if err != nil {
					log.Printf("Error on producting message: %s\n", err.Error())
					return
				}
			}, eventId),
		)
		if err != nil {
			log.Printf("Error creating job for Event: id=%d error=%s\n", eventId, err.Error())
			return err
		}
		jobID = job.ID().String()
		jobTask.ID = job.ID()
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
	return jobID, nil
}
