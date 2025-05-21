package aws

import (
	"context"
	"ebs/src/lib"
	"ebs/src/types"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSSubscriber struct {
	Name    string
	handler *types.Handler
	inner   *sns.Client
}

func NewSNSSubscriber(topic string) *SNSSubscriber {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Error loading default config: %s\n", err.Error())
		return nil
	}
	inner := sns.NewFromConfig(cfg)
	/* inner.SetSubscriptionAttributes(context.TODO(), &sns.SetSubscriptionAttributesInput{
		AttributeName: aws.String(""),
		AttributeValue: aws.String(""),
	}) */
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
	/* log.Printf("[%s] Metadata: %v\n", s.Name, output.ResultMetadata)
	result, err := s.inner.ListSubscriptionsByTopic(context.TODO(), &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topicArn),
	})
	if err != nil {
		log.Printf("[%s] Error retrieving subscriptions: %s\n", s.Name, err.Error())
	} else {
		for _, sub := range result.Subscriptions {
			attrs, err := s.inner.GetSubscriptionAttributes(context.TODO(), &sns.GetSubscriptionAttributesInput{
				SubscriptionArn: sub.SubscriptionArn,
			})
			if err != nil {
				log.Printf("[%s] Error retrieving attributes: %s\n", s.Name, err.Error())
			} else {
				log.Printf("[%s] Attributes: %s\n", s.Name, attrs)
			}
		}
	} */
	return output.SubscriptionArn, nil
}
