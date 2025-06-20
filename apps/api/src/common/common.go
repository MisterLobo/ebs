package common

import (
	"ebs/src/lib"
	awslib "ebs/src/lib/aws"
	"ebs/src/utils"
	"log"
)

func SQSConsumers() {
	dlq := awslib.NewSQSConsumer(utils.WithSuffix("DLQ"), func(payload string) {
		log.Println("DLQ: message received")
	})
	dlq.Listen()
	pr := awslib.NewSQSConsumer(utils.WithSuffix("PendingReservations"), func(payload string) {
		// TODO: implement PendingReservations handler
		log.Println("PendingReservations: message received")
	})
	pr.Listen()
	eb := awslib.NewSQSConsumer(utils.WithSuffix("ExpiredBookings"), func(payload string) {
		// TODO: implement ExpiredBookings handler
		log.Println("ExpiredBookings: message received")
	})
	eb.Listen()
	pp := awslib.NewSQSConsumer(utils.WithSuffix("PaymentsProcessing"), func(payload string) {
		// TODO: implement PaymentsProcessing handler
		log.Println("PaymentsProcessing: message received")
	})
	pp.Listen()

	go EventsToOpenConsumer()
	go EventsToCloseConsumer()
	go EventsToCompleteConsumer()
	go PendingTransactionsConsumer()
	go PaymentTransactionUpdatesConsumer()
}

func SNSSubscribes() {
	eventsToOpen := awslib.NewSNSSubscriber(utils.WithSuffix("EventsToOpen"))
	eventsToOpen.Subscribe("sqs", lib.GetQueueArn(utils.WithSuffix("EventsToOpen")))
	eventsToClose := awslib.NewSNSSubscriber(utils.WithSuffix("EventsToClose"))
	eventsToClose.Subscribe("sqs", lib.GetQueueArn(utils.WithSuffix("EventsToClose")))
	eventsToComplete := awslib.NewSNSSubscriber(utils.WithSuffix("EventsToComplete"))
	eventsToComplete.Subscribe("sqs", lib.GetQueueArn(utils.WithSuffix("EventsToComplete")))
}
