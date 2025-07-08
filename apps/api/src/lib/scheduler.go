package lib

import (
	"context"
	"ebs/src/config"
	"ebs/src/types"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsched "github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulerTypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

type Key string

const (
	varsKey Key = "vars"
)

var scheduler gocron.Scheduler

func NewScheduler(s gocron.Scheduler) {
	scheduler = s
}

func GetScheduler() (gocron.Scheduler, error) {
	if scheduler != nil {
		return scheduler, nil
	}
	sched, err := gocron.NewScheduler()
	if err != nil {
		log.Printf("Error initializing Scheduler: %s\n", err.Error())
		return nil, err
	}
	scheduler = sched
	numJobs := len(sched.Jobs())
	log.Printf("Jobs in queue: %d\n", numJobs)
	return sched, nil
}

func CreateCronJob(handler any, duration time.Duration, args ...any) (*string, error) {
	sched, err := GetScheduler()
	if err != nil {
		log.Println("An error has occurred. Check logs for info")
		return nil, err
	}
	j, err := sched.NewJob(
		gocron.DurationJob(duration),
		gocron.NewTask(handler, args),
	)
	if err != nil {
		return nil, err
	}
	id := j.ID().String()
	return &id, nil
}

func CreateOneTimeCronJob(def gocron.JobDefinition, task gocron.Task) (*string, error) {
	sched, err := GetScheduler()
	if err != nil {
		log.Println("An error has occurred. Check logs for info")
		return nil, err
	}
	j, err := sched.NewJob(
		def,
		task,
	)
	if err != nil {
		return nil, err
	}
	id := j.ID().String()
	log.Printf("Job: %s %s\n", id, j.Name())
	return &id, nil
}

func CreateSchedule(name string, startDate time.Time, scheduleExpression string, topic string, input string) (*uuid.UUID, error) {
	// eventStatusUpdateArn := "arn:aws:sns:ap-southeast-1:645972258043:EventUpdateStatus"
	client := AWSGetSchedulerClient()
	now := time.Now().UTC()
	in5m := now.Add(5 * time.Minute)
	strnow := in5m.Format("2006-01-02T15:04:05")
	log.Printf("sched: %s\n", strnow)
	sid := uuid.New()
	roleArn := os.Getenv("SCHEDULER_ROLE_ARN")
	topicArn := GetTopicArn(topic)
	sched, err := client.CreateSchedule(context.TODO(), &awsched.CreateScheduleInput{
		Name:      aws.String(fmt.Sprintf("schedule_%s", name)),
		StartDate: aws.Time(startDate),
		Target: &schedulerTypes.Target{
			Arn:     aws.String(topicArn),
			RoleArn: aws.String(roleArn),
			Input:   aws.String(input),
			RetryPolicy: &schedulerTypes.RetryPolicy{
				MaximumRetryAttempts: aws.Int32(3),
			},
		},
		FlexibleTimeWindow:    &schedulerTypes.FlexibleTimeWindow{Mode: schedulerTypes.FlexibleTimeWindowModeOff},
		ScheduleExpression:    aws.String(fmt.Sprintf("at(%s)", scheduleExpression)),
		ActionAfterCompletion: schedulerTypes.ActionAfterCompletionDelete,
	})
	if err != nil {
		log.Printf("Failed to create Schedule: %s\n", err.Error())
		return nil, err
	}
	log.Printf("Created schedule at: %s\n", *sched.ScheduleArn)
	return &sid, nil
}

func CreateCronSchedule(clientId, topic string, startDate time.Time, scheduleExpression string, payload types.JSONB) {
	s, err := GetScheduler()
	if err != nil {
		log.Printf("Error initializing Scheduler client: %s\n", err.Error())
		return
	}
	j, err := s.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(startDate)),
		gocron.NewTask(func(id uint) {
			KafkaProduceMessage(clientId, topic, &payload)
		}, 1),
	)
	if err != nil {
		log.Printf("Error creating job: %s\n", err.Error())
		return
	}
	log.Printf("New Job: %s\n", j.ID().String())
}

type Scheduler interface {
	Name() string
	CreateScheduleWithStartDate(ctx context.Context, s time.Time, p types.JSONB) (*uuid.UUID, error)
}

type EventBridgeScheduler struct {
	inner *awsched.Client
}

func (e *EventBridgeScheduler) Name() string {
	return "EventBridge"
}

func (e *EventBridgeScheduler) CreateScheduleWithStartDate(ctx context.Context, s time.Time, p types.JSONB) (*uuid.UUID, error) {
	vars := ctx.Value(varsKey).(map[string]string)
	name := vars["name"]
	topic := vars["topic"]
	in := *e.inner
	bPayload, _ := json.Marshal(p)
	input := string(bPayload)
	sid := uuid.New()
	roleArn := os.Getenv("SCHEDULER_ROLE_ARN")
	topicArn := GetTopicArn(topic)
	sRunsAt := s.Format("2006-01-02T15:04:05")
	sched, err := in.CreateSchedule(context.TODO(), &awsched.CreateScheduleInput{
		Name:      aws.String(fmt.Sprintf("schedule_%s", name)),
		StartDate: aws.Time(s),
		Target: &schedulerTypes.Target{
			Arn:     aws.String(topicArn),
			RoleArn: aws.String(roleArn),
			Input:   aws.String(input),
			RetryPolicy: &schedulerTypes.RetryPolicy{
				MaximumRetryAttempts: aws.Int32(3),
			},
		},
		FlexibleTimeWindow:    &schedulerTypes.FlexibleTimeWindow{Mode: schedulerTypes.FlexibleTimeWindowModeOff},
		ScheduleExpression:    aws.String(fmt.Sprintf("at(%s)", sRunsAt)),
		ActionAfterCompletion: schedulerTypes.ActionAfterCompletionDelete,
	})
	if err != nil {
		log.Printf("Failed to create Schedule: %s\n", err.Error())
		return nil, err
	}
	log.Printf("Created schedule at: %s\n", *sched.ScheduleArn)
	return &sid, nil
}

type LocalScheduler struct {
	inner *gocron.Scheduler
}

func (l *LocalScheduler) Name() string {
	return "Local"
}
func (l *LocalScheduler) CreateScheduleWithStartDate(ctx context.Context, s time.Time, p types.JSONB) (*uuid.UUID, error) {
	in := *l.inner
	j, err := in.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(s)),
		gocron.NewTask(func(ctx context.Context, p types.JSONB) {
			log.Printf("[%s] Running scheduled task...\n", l.Name())
			KafkaTaskHandlerFunc(ctx, &p)
		}, ctx, p),
	)
	if err != nil {
		log.Printf("Error creating job: %s\n", err.Error())
		return nil, err
	}
	sRunsAt := s.Format(config.TIME_PARSE_FORMAT)
	log.Printf("[%s] New Job scheduled on: %s %s\n", l.Name(), j.ID().String(), sRunsAt)
	jid := j.ID()
	return &jid, nil
}

func NewAwsScheduler() *EventBridgeScheduler {
	inner := AWSGetSchedulerClient()
	s := EventBridgeScheduler{inner: inner}
	return &s
}

func NewLocalScheduler() *LocalScheduler {
	inner, _ := GetScheduler()
	s := LocalScheduler{inner: &inner}
	return &s
}

// CreateScheduler returns either an instance of LocalScheduler or EventBridgeScheduler based on the app environment value
func CreateScheduler() Scheduler {
	env := config.API_ENV
	if env == string(types.Production) || env == string(types.Test) {
		ebs := NewAwsScheduler()
		return ebs
	}
	local := NewLocalScheduler()
	return local
}

// Wrapper for creating scheduled job based on the app environment. local will use the LocalScheduler otherwise will use AWS EventBridge Scheduler
func NewScheduledJob(startDate time.Time, vars map[string]string, p types.JSONB) (*uuid.UUID, error) {
	sch := CreateScheduler()
	ctx := context.Background()
	ctx = context.WithValue(ctx, varsKey, vars)
	log.Printf("Created scheduler with name: %s\n", sch.Name())

	sid, err := sch.CreateScheduleWithStartDate(ctx, startDate, p)
	if err != nil {
		return nil, err
	}
	return sid, nil
}

func KafkaTaskHandlerFunc(ctx context.Context, p *types.JSONB) {
	vars := ctx.Value(varsKey).(map[string]string)
	clientId := vars["clientId"]
	topic := vars["topic"]
	go KafkaProduceMessage(clientId, topic, p)
}
