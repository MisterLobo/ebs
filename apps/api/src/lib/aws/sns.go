package aws

import (
	"context"
	"ebs/src/lib"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSSubscriber struct {
	Name  string
	inner *sns.Client
}

func NewSNSSubscriber(topic string) *SNSSubscriber {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Error loading default config: %s\n", err.Error())
		return nil
	}
	inner := sns.NewFromConfig(cfg)
	new := SNSSubscriber{
		Name:  topic,
		inner: inner,
	}
	return &new
}

func (s *SNSSubscriber) Subscribe(proto string, endpoint string) (*string, error) {
	topicArn := lib.GetTopicArn(s.Name)
	mid := os.Getenv("AWS_MEMBER_ID")
	s.inner.AddPermission(context.TODO(), &sns.AddPermissionInput{
		AWSAccountId: []string{mid},
		ActionName:   []string{"sns:*"},
		TopicArn:     aws.String(topicArn),
	})
	output, err := s.inner.Subscribe(context.TODO(), &sns.SubscribeInput{
		Protocol: aws.String(proto),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(endpoint),
	})
	if err != nil {
		log.Printf("Error subscribing to topic [%s]: %s\n", s.Name, err.Error())
		return nil, err
	}
	return output.SubscriptionArn, nil
}
