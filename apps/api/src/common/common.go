package common

import (
	"ebs/src/lib"
	awslib "ebs/src/lib/aws"
	"log"
)

func SQSConsumers() {
	dlq := awslib.NewSQSConsumer("DLQ", func(payload string) {
		log.Println("DLQ: message received")
	})
	dlq.Listen()
	pr := awslib.NewSQSConsumer("PendingReservations", func(payload string) {
		// TODO: implement PendingReservations handler
		log.Println("PendingReservations: message received")
	})
	pr.Listen()
	eb := awslib.NewSQSConsumer("ExpiredBookings", func(payload string) {
		// TODO: implement ExpiredBookings handler
		log.Println("ExpiredBookings: message received")
	})
	eb.Listen()
	pp := awslib.NewSQSConsumer("PaymentsProcessing", func(payload string) {
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
	eventsToOpen := awslib.NewSNSSubscriber("EventsToOpen")
	eventsToOpen.Subscribe("sqs", lib.GetQueueArn("EventsToOpen"))
	eventsToClose := awslib.NewSNSSubscriber("EventsToClose")
	eventsToClose.Subscribe("sqs", lib.GetQueueArn("EventsToClose"))
	eventsToComplete := awslib.NewSNSSubscriber("EventsToComplete")
	eventsToComplete.Subscribe("sqs", lib.GetQueueArn("EventsToComplete"))
}
