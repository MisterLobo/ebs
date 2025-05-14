package lib

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
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
