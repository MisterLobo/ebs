package common

import (
	"context"
	"ebs/src/lib"
	awslib "ebs/src/lib/aws"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func SQSConsumers() {
	dlq := awslib.NewSQSConsumer("DLQ", func(payload string) {
		log.Println("DLQ: message received")
	})
	dlq.Listen()
	z := awslib.NewSQSConsumer("Z", func(payload string) {
		log.Println("Z: message received")
	})
	z.Listen()
	pr := awslib.NewSQSConsumer("PendingReservations", func(payload string) {
		log.Println("PendingReservations: message received")
	})
	pr.Listen()
	eb := awslib.NewSQSConsumer("ExpiredBookings", func(payload string) {
		log.Println("ExpiredBookings: message received")
	})
	eb.Listen()
	pp := awslib.NewSQSConsumer("PaymentsProcessing", func(payload string) {
		log.Println("PaymentsProcessing: message received")
	})
	pp.Listen()

	go EventsToOpenConsumer()
	go EventsToCloseConsumer()
	go EventsToCompleteConsumer()
	go PendingTransactionsConsumer()

	c := lib.AWSGetSQSClient()
	qurl, err := c.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
		QueueName:              aws.String("Z"),
		QueueOwnerAWSAccountId: aws.String(os.Getenv("AWS_MEMBER_ID")),
	})
	if err != nil {
		log.Printf("Could not get queue URL: %s\n", err.Error())
	} else {
		c.SendMessage(context.Background(), &sqs.SendMessageInput{
			MessageBody: aws.String("this is a test"),
			QueueUrl:    qurl.QueueUrl,
		})
	}
}

func SNSSubscribes() {
	/* eb := awslib.NewSNSSubscriber("ExpiredBookings")
	eb.Subscribe("sqs", lib.GetQueueArn("ExpiredBookings")) */
}
