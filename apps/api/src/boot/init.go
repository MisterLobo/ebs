package boot

import (
	"context"
	"ebs/src/common"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
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
		&models.Notification{},
		&models.Credential{},
		&models.Token{},
		&models.Account{},
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

	CREATE OR REPLACE FUNCTION daily_transactions(org_id INT)
	RETURNS TABLE (
		txn_date TEXT,
		completed_txn BIGINT,
		pending_txn BIGINT,
		total_txn BIGINT,
		total_revenue NUMERIC
	) AS $$
	BEGIN
		RETURN QUERY
		SELECT
			TO_CHAR(d::date, 'YYYY-MM-DD') AS txn_date,
			COUNT(DISTINCT CASE WHEN t.status = 'paid' THEN 1 END) AS completed_txn,
			COUNT(DISTINCT CASE WHEN t.status = 'pending' THEN 1 END) AS pending_txn,
			COUNT(DISTINCT t.id) AS total_txn,
			COALESCE(SUM(DISTINCT t.amount_paid),0) AS total_revenue
		FROM generate_series(
			CURRENT_DATE - INTERVAL '30 days',
			CURRENT_DATE,
			INTERVAL '1 day'
		) as d
		LEFT JOIN events e ON e.organizer_id=org_id
		LEFT JOIN bookings b ON b.event_id=e.id
		LEFT JOIN transactions t ON t.id=b.transaction_id AND t.created_at::date = d::date
		GROUP BY d.d
		ORDER BY d.d;
	END;
	$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Printf("Error creating FUNCTION set_tenant: %s\n", err.Error())
	}

	return db
}

func InitBroker() {
	apiEnv := os.Getenv("API_ENV")
	go common.UpdateMissingSlugs()
	go RecoverQueuedJobs()
	go UpdateExpiredJobs()
	go StatusUpdateExpiredBookings()
	if apiEnv == "test" || apiEnv == "production" {
		go func() {
			InitTopics()
			InitQueues()

			lib.S3ListObjects()
			common.SQSConsumers()
			common.SNSSubscribes()
		}()
	} else {
		emailQueue := os.Getenv("EMAIL_QUEUE")
		lib.KafkaCreateTopics(
			utils.WithSuffix("Retry"),
			utils.WithSuffix("EventsToOpen"),
			utils.WithSuffix("EventsToClose"),
			utils.WithSuffix("EventsToComplete"),
			utils.WithSuffix("PendingTransactions"),
			utils.WithSuffix("PaymentTransactionUpdates"),
			utils.WithSuffix(emailQueue),
		)
		var retryConsumer types.Handler = common.KafkaRetryConsumer
		go lib.KafkaConsumer("retry", utils.WithSuffix("Retry"), &retryConsumer)

		var eventsToOpenConsumer types.Handler = common.KafkaEventsToOpenConsumer
		go lib.KafkaConsumer("events", utils.WithSuffix("EventsToOpen"), &eventsToOpenConsumer)

		var eventsToCloseConsumer types.Handler = common.KafkaEventsToCloseConsumer
		go lib.KafkaConsumer("events", utils.WithSuffix("EventsToClose"), &eventsToCloseConsumer)

		var eventsToCompleteConsumer types.Handler = common.KafkaEventsToCompleteConsumer
		go lib.KafkaConsumer("events", utils.WithSuffix("EventsToComplete"), &eventsToCompleteConsumer)

		var emailsToSendConsumer types.Handler = common.KafkaEmailsToSendConsumer
		go lib.KafkaConsumer("emails", utils.WithSuffix(emailQueue), &emailsToSendConsumer)

		var pendingTxnConsumer types.Handler = common.KafkaPendingTransactionsConsumer
		go lib.KafkaConsumer("transactions", utils.WithSuffix("PendingTransactions"), &pendingTxnConsumer)

		var kafkaPaymentTransactionUpdatesConsumer types.Handler = common.KafkaPaymentTransactionUpdatesConsumer
		go lib.KafkaConsumer("payments", utils.WithSuffix("PaymentTransactionUpdates"), &kafkaPaymentTransactionUpdatesConsumer)
	}
}

func InitQueues() {
	emailQueue := os.Getenv("EMAIL_QUEUE")
	lib.SQSCreateQueue(utils.WithSuffix(emailQueue))
	lib.SQSCreateQueue(utils.WithSuffix("EventsToOpen"))
	lib.SQSCreateQueue(utils.WithSuffix("EventsToClose"))
	lib.SQSCreateQueue(utils.WithSuffix("EventsToComplete"))
	lib.SQSCreateQueue(utils.WithSuffix("PendingReservations"))
	lib.SQSCreateQueue(utils.WithSuffix("PendingTransactions"))
	lib.SQSCreateQueue(utils.WithSuffix("PaymentsProcessing"))
	lib.SQSCreateQueue(utils.WithSuffix("ExpiredBookings"))
	lib.SQSCreateQueue(utils.WithSuffix("PaymentTransactionUpdates"))
	lib.SQSCreateQueue(utils.WithSuffix("DLQ"))
}
func InitTopics() {
	lib.SNSCreateTopic(utils.WithSuffix("EventsToOpen"))
	lib.SNSCreateTopic(utils.WithSuffix("EventsToClose"))
	lib.SNSCreateTopic(utils.WithSuffix("EventsToComplete"))
	lib.SNSCreateTopic(utils.WithSuffix("PendingTransactions"))
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
		log.Println("An error has occurred while shutting down Scheduler. Check logs for info")
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

func DownloadSDKFileFromS3() error {
	filename := "admin-sdk-credentials.json"
	secretsPath := os.Getenv("SECRETS_DIR")
	secretsBucket := os.Getenv("S3_SECRETS_BUCKET")
	sdkFilePath := path.Join(secretsPath, filename)
	return DownloadFileFromS3(filename, sdkFilePath, secretsBucket, true)
}
func DownloadServiceKeyFromS3() error {
	filename := "client_secret.json"
	secretsPath := os.Getenv("SECRETS_DIR")
	secretsBucket := os.Getenv("S3_SECRETS_BUCKET")
	sdkFilePath := path.Join(secretsPath, filename)
	return DownloadFileFromS3(filename, sdkFilePath, secretsBucket, true)
}
func DownloadFileFromS3(filename, localpath, bucket string, overwriteIfExists bool) error {
	_, err := os.Stat(localpath)
	if errors.Is(err, os.ErrNotExist) || (err == nil && overwriteIfExists) {
		log.Printf("Downloading file key=%s bucket=%s\n", filename, bucket)
		client := lib.AWSGetS3Client()
		adminSdkObjectKey := filename
		object, err := client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(adminSdkObjectKey),
		})
		if err != nil {
			log.Printf("[S3] Error retrieving object: %s\n", err.Error())
			return err
		}
		defer object.Body.Close()
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Could not create file %s: %s\n", filename, err.Error())
			return err
		}
		defer file.Close()
		body, err := io.ReadAll(object.Body)
		if err != nil {
			log.Printf("Couldn't read object body from %s: %s\n", filename, err.Error())
			return err
		}
		_, err = file.Write(body)
		if err != nil {
			log.Printf("Error writing to file: %s\n", err.Error())
			return err
		}
		log.Println("File has been written")
		return nil
	}
	log.Println("File exists!")
	return nil
}
