package boot

import (
	"context"
	"ebs/src/common"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
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
		&models.Transaction{},
		&models.EventSubscription{},
		&models.JobTask{},
		&models.Setting{},
		&models.Team{},
		&models.TeamMember{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Rating{},
	)
	if err != nil {
		log.Fatalf("error migration: %s", err.Error())
	}
	if err = db.Exec(`
	CREATE OR REPLACE FUNCTION set_tenant(tenant_id text) RETURNS void AS $$
	BEGIN
		PERFORM set_config('app.current_tenant', tenant_id, false);
	END;
	$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Printf("Error creating FUNCTION set_tenant: %s\n", err.Error())
	}

	return db
}

func InitBroker() {
	appEnv := os.Getenv("APP_ENV")
	go common.UpdateMissingSlugs()
	go RecoverQueuedJobs()
	go UpdateExpiredJobs()
	go StatusUpdateExpiredBookings()
	if appEnv == "test" || appEnv == "prod" {
		go InitTopics()
		go InitQueues()

		go lib.S3ListObjects()
		go common.SQSConsumers()
		go common.SNSSubscribes()
	} else {
		var h1 types.Handler = common.KafkaEventsToOpenConsumer
		go lib.KafkaConsumer("events", "EventsToOpen", &h1)
		var h2 types.Handler = common.KafkaEventsToCloseConsumer
		go lib.KafkaConsumer("events", "EventsToClose", &h2)
		var h3 types.Handler = common.KafkaEventsToCompleteConsumer
		go lib.KafkaConsumer("events", "EventsToComplete", &h3)
		go lib.KafkaCreateTopics("events-open", "events-close", "EventsToOpen", "EventsToClose", "EventsToComplete")
	}
	go lib.TestRedis()

}

func InitQueues() {
	lib.SQSCreateQueue("PendingReservations")
	lib.SQSCreateQueue("PendingTransactions")
	lib.SQSCreateQueue("PaymentsProcessing")
	lib.SQSCreateQueue("ExpiredBookings")
	lib.SQSCreateQueue("PaymentTransactionUpdates")
}
func InitTopics() {
	// lib.SNSCreateTopic("ExpiredBookings")
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
			err := lib.KafkaProduceMessage(jobTask.Payload["producerClientId"].(string), jobTask.Payload["topic"].(string), &jobTask.Payload)
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

func StatusUpdateExpiredBookings() {
	db := db.GetDb()
	err := db.Transaction(func(tx *gorm.DB) error {
		var eventIDs []uint
		if err := tx.
			Model(&models.Event{}).
			Where("status IN (?)", []string{string(types.EVENT_TICKETS_NOTIFY)}).
			Where(tx.
				Where("date_time < ?", time.Now()).
				Or("opens_at < ?", time.Now()).
				Or("deadline < ?", time.Now()),
			).
			Select("id").
			Pluck("id", &eventIDs).
			Error; err != nil {
			return err
		}
		log.Printf("[CONSUMER]: Found %d Events overdue\n", len(eventIDs))
		if err := tx.Model(&models.Event{}).
			Where("id IN (?)", eventIDs).
			Update("status", types.EVENT_EXPIRED).
			Error; err != nil {
			return err
		}
		var bids []uint
		if err := tx.
			Model(&models.Booking{}).
			Where("event_id IN (?)", eventIDs).
			Select("id").
			Pluck("id", &bids).
			Error; err != nil {
			return err
		}
		if err := tx.
			Model(&models.Booking{}).
			Where("id IN (?)", bids).
			Update("status", types.BOOKING_EXPIRED).
			Error; err != nil {
			return err
		}
		if err := tx.
			Model(&models.Reservation{}).
			Where("booking_id IN (?)", bids).
			Update("status", "expired").
			Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Error while processing expired bookings: %s\n", err.Error())
	}
}

func DownloadSDKFileFromS3() {
	cwd, _ := os.Getwd()
	log.Printf("[S3] cwd:%s\n", cwd)
	filename := "admin-sdk-credentials.json"
	secretsPath := os.Getenv("SECRETS_DIR")
	sdkFilePath := path.Join(secretsPath, filename)
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
