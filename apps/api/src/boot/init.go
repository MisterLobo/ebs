package boot

import (
	"context"
	"ebs/src/common"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"errors"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
)

func InitDb() *gorm.DB {
	db := db.GetDb()

	err := db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.Event{},
		&models.Ticket{},
		&models.Booking{},
		&models.Reservation{},
		&models.Admission{},
		&models.EventSubscription{},
		&models.JobTask{},
		&models.Setting{},
	)
	if err != nil {
		log.Fatalf("error migration: %s", err.Error())
	}

	return db
}

func InitBroker() {
	go RecoverQueuedJobs()
	go UpdateExpiredJobs()
	lib.KafkaConsumer("footest")
	lib.KafkaProducer("asdf")
	go lib.KafkaCreateTopics("events-open", "events-close")
	go common.EventsOpenConsumer()
	go lib.S3ListObjects()
	// go lib.S3CreateObjects()
	go lib.SNSSubscribe()
	go lib.SQSConsumer()
	go lib.SchedulerTest()
}

func InitScheduler() {
	sched, err := lib.GetScheduler()
	if err != nil {
		log.Println("An error has occurred. Check logs for info")
		return
	}
	j, err := sched.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Now().Add(5*time.Minute))),
		gocron.NewTask(func(a string, b int) {
			log.Println("Running background task:", a, b)
		}, "hello", 5),
	)
	if err != nil {
		log.Printf("Error running job: %s\n", err.Error())
		return
	}
	jobsWaitingInQueue := len(sched.Jobs())
	log.Println("Jobs in queue:", jobsWaitingInQueue)
	log.Printf("Job ID: %s %s\n", j.Name(), j.ID().String())
	/* j, err := sched.NewJob(
		gocron.DurationJob(10*time.Second),
		gocron.NewTask(func(a string, b int) {
			log.Printf("%s, %d\n", a, b)
			lib.KafkaProduceMessage("topic1", map[string]any{
				a: b,
			})
		}, "hello", 1),
	)
	if err != nil {
		log.Printf("Error running job: %s", err.Error())
		return
	}
	log.Printf("Job ID: %s\n", j.ID().String()) */
	sched.Start()
}

func StopScheduler() {
	sched, err := lib.GetScheduler()
	if err != nil {
		log.Println("Error retrieving Scheduler. Check logs for info")
		return
	}
	err = sched.Shutdown()
	if err != nil {
		log.Println("An error has occurred while shutting stopping Scheduler. Check logs for info")
		return
	}
}

func RecoverQueuedJobs() error {
	sched, err := lib.GetScheduler()
	if err != nil {
		return err
	}
	db := db.GetDb()
	ss := db.Session(&gorm.Session{PrepareStmt: true})
	var jobTasks []models.JobTask
	today := time.Now()
	in1m := today.Add(1 * time.Minute)
	in3months := today.Add((24 * 30 * 3) * time.Hour)
	err = ss.
		Model(&models.JobTask{}).Select("id", "payload", "runs_at").
		Where(&models.JobTask{Status: "pending", JobType: "OneTimeJobStartDateTime"}).
		Where("runs_at BETWEEN ? AND ?", in1m, in3months).
		Order("runs_at asc").
		Limit(100).
		Find(&jobTasks).
		Error
	if err != nil {
		log.Printf("Error retrieving jobs: %s\n", err.Error())
		return err
	}
	log.Printf("Found %d pending jobs", len(jobTasks))
	for _, jobTask := range jobTasks {
		log.Printf("Queueing: %s\n", jobTask.ID.String())
		jobDef := gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(jobTask.RunsAt))
		jt := gocron.NewTask(func() {
			log.Println("Running scheduled task")
			err := lib.KafkaProduceMessage(jobTask.Payload["producerClientId"].(string), jobTask.Payload["topic"].(string), jobTask.Payload)
			if err != nil {
				log.Printf("Error on producting message: %s\n", err.Error())
				return
			}
		}, jobTask.HandlerParams...)
		job, err := sched.NewJob(
			jobDef,
			jt,
		)
		if err != nil {
			log.Printf("Failed to schedule job [%s]. Skipping: %s\n", jobTask.ID.String(), err.Error())
			continue
		}
		log.Printf("Added job to scheduler: name=%s id=%s job=%s\n", jobTask.Name, jobTask.ID.String(), job.ID().String())
	}

	return nil
}

func UpdateExpiredJobs() {
	db := db.GetDb()
	err := db.
		Transaction(func(tx *gorm.DB) error {
			err := tx.Model(&models.JobTask{}).
				Where("status", "pending").
				Where("runs_at < ?", time.Now()).
				Update("status", "expired").Error
			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		log.Printf("Error while processing expired jobs: %s\n", err.Error())
	}
}

func DownloadSDKFileFromS3() {
	cwd, _ := os.Getwd()
	log.Printf("[S3] cwd:%s\n", cwd)
	filename := "admin-sdk-credentials.json"
	sdkFilePath := path.Join("/secrets", filename)
	_, err := os.Stat(sdkFilePath)
	if errors.Is(err, os.ErrNotExist) {
		log.Println("File not found. Downloading...")
		client := lib.AWSGetS3Client()
		adminSdkObjectKey := filename
		secretsBucket := os.Getenv("S3_SECRETS_BUCKET")
		object, err := client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(secretsBucket),
			Key:    aws.String(adminSdkObjectKey),
		})
		if err != nil {
			log.Printf("[S3] Error retrieving object: %s\n", err.Error())
			return
		}
		defer object.Body.Close()
		file, err := os.Create(sdkFilePath)
		if err != nil {
			log.Printf("Could not create file %s: %s\n", filename, err.Error())
			return
		}
		defer file.Close()
		body, err := io.ReadAll(object.Body)
		if err != nil {
			log.Printf("Couldn't read object body from %s: %s\n", filename, err.Error())
			return
		}
		_, err = file.Write(body)
		if err != nil {
			log.Printf("Error writing to file: %s\n", err.Error())
			return
		}
		log.Println("File has been written")
	}
	log.Println("File exists!")
}
