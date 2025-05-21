package lib

import (
	"context"
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

var scheduler gocron.Scheduler

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
