package lib

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsched "github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulerTpes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
)

type AWSSDKClient struct {
	innerConfig    *aws.Config
	innerS3        *s3.Client
	innerSQS       *sqs.Client
	innerSNS       *sns.Client
	innerScheduler *awsched.Client
}

func awsGetSdkClient() (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Error loading default config: %s\n", err.Error())
		return nil, err
	}
	iamRole := os.Getenv("AWS_IAM_ROLE_ARN")
	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
		RoleArn:         aws.String(iamRole),
		RoleSessionName: aws.String("test-session"),
	})
	if err != nil {
		log.Printf("Error configuring STS client: %s\n", err.Error())
		return nil, err
	}
	creds := output.Credentials
	cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(
		credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken),
	))
	if err != nil {
		log.Printf("Error configuration: %s\n", err.Error())
		return nil, err
	}

	return &cfg, nil
}

func AWSGetSchedulerClient() *awsched.Client {
	cfg, _ := awsGetSdkClient()
	client := awsched.NewFromConfig(*cfg)

	return client
}
func AWSGetS3Client() *s3.Client {
	cfg, err := awsGetSdkClient()
	if err != nil {
		log.Printf("Failed to iniialize S3: %s\n", err.Error())
		return nil
	}

	client := s3.NewFromConfig(*cfg)
	return client
}
func AWSGetSQSClient() *sqs.Client {
	cfg, err := awsGetSdkClient()
	if err != nil {
		log.Printf("Failed to initialize SQS client: %s\n", err.Error())
		return nil
	}
	client := sqs.NewFromConfig(*cfg)
	return client
}
func AWSGetSNSClient() *sns.Client {
	cfg, err := awsGetSdkClient()
	if err != nil {
		log.Printf("Failed to initialize SNS client: %s\n", err.Error())
		return nil
	}
	client := sns.NewFromConfig(*cfg)
	return client
}

func S3CreateObjects() {
	adminSdkObjectKey := "admin-sdk-credentials.json"
	secretsBucket := os.Getenv("S3_SECRETS_BUCKET")
	cwd, _ := os.Getwd()
	sdkFilePath := path.Join(cwd, adminSdkObjectKey)
	file, err := os.Open(sdkFilePath)
	if err != nil {
		log.Printf("Could not open file to upload: %s\n", err.Error())
		return
	}
	defer file.Close()
	client := AWSGetS3Client()
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(secretsBucket),
		Key:    aws.String(adminSdkObjectKey),
		Body:   file,
	})
	if err != nil {
		log.Printf("Could not put object to S3 bucket: %s\n", err.Error())
		return
	}
	err = s3.NewObjectExistsWaiter(client).Wait(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(secretsBucket),
		Key:    aws.String(adminSdkObjectKey),
	}, time.Minute)
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist: %s\n", adminSdkObjectKey, err.Error())
		return
	}
	log.Printf("Added object '%s' to bucket '%s'", adminSdkObjectKey, secretsBucket)
}
func S3ListObjects() {
	client := AWSGetS3Client()
	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(os.Getenv("S3_SECRETS_BUCKET")),
	})
	if err != nil {
		log.Printf("[S3] Error retrieving objects: %s\n", err.Error())
		return
	}

	log.Println("Retrieving first page")
	for _, object := range output.Contents {
		log.Printf("key=%s size=%d", aws.ToString(object.Key), *object.Size)
	}
}

func SchedulerTest() {
	client := AWSGetSchedulerClient()
	now := time.Now().UTC()
	in5m := now.Add(5 * time.Minute)
	strnow := in5m.Format("2006-01-02T15:04:05")
	log.Printf("sched: %s\n", strnow)
	sid := uuid.New().String()
	roleArn := os.Getenv("AWS_EVENTBRIDGE_ROLE_ARN")
	sched, err := client.CreateSchedule(context.TODO(), &awsched.CreateScheduleInput{
		Name:      aws.String(fmt.Sprintf("schedule_%s", sid)),
		StartDate: aws.Time(time.Now().Add(5 * time.Minute)),
		Target: &schedulerTpes.Target{
			Arn:     aws.String("arn:aws:sns:ap-southeast-1:645972258043:EventUpdateStatus"),
			RoleArn: aws.String(roleArn),
			Input:   aws.String("received 5 minutes later"),
			RetryPolicy: &schedulerTpes.RetryPolicy{
				MaximumRetryAttempts: aws.Int32(3),
			},
		},
		FlexibleTimeWindow:    &schedulerTpes.FlexibleTimeWindow{Mode: schedulerTpes.FlexibleTimeWindowModeOff},
		ScheduleExpression:    aws.String(fmt.Sprintf("at(%s)", strnow)),
		ActionAfterCompletion: schedulerTpes.ActionAfterCompletionDelete,
	})
	if err != nil {
		log.Printf("Failed to create Schedule: %s\n", err.Error())
		return
	}
	log.Printf("Created schedule at: %s\n", *sched.ScheduleArn)
}

func SQSConsumer() {
	client := AWSGetSQSClient()
	qname := "EventUpdateStatus"
	qurl, err := client.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(qname),
	})
	if err != nil {
		log.Printf("Failed to retrieve queue URL for %s: %s\n", qname, err.Error())
		return
	}
	messagesChan := make(chan *sqsTypes.Message, 5)
	go func(chn chan<- *sqsTypes.Message) {
		for {
			output, err := client.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
				QueueUrl:            qurl.QueueUrl,
				WaitTimeSeconds:     20,
				MaxNumberOfMessages: 5,
			})
			if err != nil {
				log.Printf("[SQS] Error receiving messages: %s\n", err.Error())
				return
			}
			for _, m := range output.Messages {
				log.Printf("Received message [%s] with body: %s\n", *m.MessageId, *m.Body)
				chn <- &m
			}
		}
	}(messagesChan)

	for m := range messagesChan {
		log.Printf("Message with body: %s\n", *m.Body)
		deleteMessage(client, qurl.QueueUrl, m)
	}
}

func deleteMessage(c *sqs.Client, qurl *string, msg *sqsTypes.Message) {
	_, err := c.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      qurl,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("Error deleting message from queue: %s\n", err.Error())
		return
	}
	log.Printf("Deleted message from queue: %s\n", *msg.MessageId)
}

func SNSSubscribe() {
	client := AWSGetSNSClient()
	output, err := client.Subscribe(context.Background(), &sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		TopicArn: aws.String("arn:aws:sns:ap-southeast-1:645972258043:EventUpdateStatus"),
		Endpoint: aws.String("arn:aws:sqs:ap-southeast-1:645972258043:EventUpdateStatus"),
	})
	if err != nil {
		log.Printf("Error subscribing to topic: %s\n", err.Error())
		return
	}
	log.Printf("Subscribed to topic: %s\n", *output.SubscriptionArn)
}
