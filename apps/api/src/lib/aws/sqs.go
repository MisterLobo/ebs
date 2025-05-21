package aws

import (
	"context"
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSConsumer struct {
	Name    string
	handler *types.Handler
}

func NewSQSConsumer(queue string, handler types.Handler) *SQSConsumer {
	new := SQSConsumer{
		Name:    queue,
		handler: &handler,
	}
	return &new
}
func (s *SQSConsumer) Listen() {
	go func() {
		qname := s.Name
		client := lib.AWSGetSQSClient()
		qurl, err := client.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
			QueueName: aws.String(qname),
		})
		if err != nil {
			log.Printf("Failed to retrieve queue URL for %s: %s\n", qname, err.Error())
			return
		}
		/* attr, err := client.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(*qurl.QueueUrl),
		})
		if err != nil {
			log.Printf("Failed to retrieve attributes for %s: %s\n", qname, err.Error())
			return
		}
		log.Printf("[%s] attributes: %v\n", qname, attr.Attributes) */
		log.Printf("%s: Listening for messages...", qname)
		messagesChan := make(chan *sqstypes.Message, 5)
		go func(chn chan<- *sqstypes.Message) {
			for {
				output, err := client.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
					QueueUrl:            qurl.QueueUrl,
					WaitTimeSeconds:     20,
					MaxNumberOfMessages: 10,
				})
				if err != nil {
					log.Printf("[SQS] Error receiving messages: %s\n", err.Error())
					return
				}
				for _, m := range output.Messages {
					// log.Printf("Received message [%s] with body: %s\n", *m.MessageId, *m.Body)
					chn <- &m
				}
			}
		}(messagesChan)

		for m := range messagesChan {
			body := strings.Clone(*m.Body)
			h := *s.handler
			go h(body)
			go lib.SQSDeleteMessage(client, qurl.QueueUrl, m)
		}
	}()
}
