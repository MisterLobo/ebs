package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsched "github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulerTypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
	/* iamRole := os.Getenv("AWS_IAM_ROLE_ARN")
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
	} */

	return &cfg, nil
}

func awsGetSecretsManagerClient() *secretsmanager.Client {
	cfg, _ := awsGetSdkClient()
	sm := secretsmanager.NewFromConfig(*cfg)
	return sm
}

func AWSGetSecret(name string) string {
	c := awsGetSecretsManagerClient()
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(name),
		VersionStage: aws.String("AWSCURRENT"),
	}
	result, err := c.GetSecretValue(context.TODO(), input)
	if err != nil {
		log.Printf("Error retrieving secret %s: %s\n", name, err.Error())
		return ""
	}
	secretString := *result.SecretString
	return secretString
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
	roleArn := os.Getenv("SCHEDULER_ROLE_ARN")
	sched, err := client.CreateSchedule(context.TODO(), &awsched.CreateScheduleInput{
		Name:      aws.String(fmt.Sprintf("schedule_%s", sid)),
		StartDate: aws.Time(time.Now().Add(5 * time.Minute)),
		Target: &schedulerTypes.Target{
			Arn:     aws.String("arn:aws:sns:ap-southeast-1:645972258043:PendingReservations"),
			RoleArn: aws.String(roleArn),
			Input:   aws.String("received 5 minutes later"),
			RetryPolicy: &schedulerTypes.RetryPolicy{
				MaximumRetryAttempts: aws.Int32(3),
			},
		},
		FlexibleTimeWindow:    &schedulerTypes.FlexibleTimeWindow{Mode: schedulerTypes.FlexibleTimeWindowModeOff},
		ScheduleExpression:    aws.String(fmt.Sprintf("at(%s)", strnow)),
		ActionAfterCompletion: schedulerTypes.ActionAfterCompletionDelete,
	})
	if err != nil {
		log.Printf("Failed to create Schedule: %s\n", err.Error())
		return
	}
	log.Printf("Created schedule at: %s\n", *sched.ScheduleArn)
}

func SQSConsumer(qname string) {
	client := AWSGetSQSClient()
	qurl, err := client.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(qname),
	})
	if err != nil {
		log.Printf("Failed to retrieve queue URL for %s: %s\n", qname, err.Error())
		return
	}
	log.Printf("%s: Listening for messages...", qname)
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
				// log.Printf("Received message [%s] with body: %s\n", *m.MessageId, *m.Body)
				chn <- &m
			}
		}
	}(messagesChan)

	for m := range messagesChan {
		SQSDeleteMessage(client, qurl.QueueUrl, m)
	}
}

func SQSDeleteMessage(c *sqs.Client, qurl *string, msg *sqsTypes.Message) {
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

func SNSCreateTopic(name string) (string, error) {
	owner := GetIAMUserArn(os.Getenv("AWS_IAM_USER"))
	log.Printf("owner: %s\n", owner)
	c := AWSGetSNSClient()
	out, err := c.CreateTopic(context.TODO(), &sns.CreateTopicInput{
		Name: aws.String(name),
		Attributes: map[string]string{
			"FifoTopic": "false",
		},
	})
	if err != nil {
		log.Printf("Error creating topic %s: %s\n", name, err.Error())
		return "", err
	}
	log.Printf("[%s] Topic has been created\n", name)
	return *out.TopicArn, nil
}

func SNSDeleteTopic(topic string) {
	arn := GetTopicArn(topic)
	c := AWSGetSNSClient()
	_, err := c.DeleteTopic(context.TODO(), &sns.DeleteTopicInput{
		TopicArn: aws.String(arn),
	})
	if err != nil {
		log.Printf("Error deleting topic [%s]: %s\n", topic, err.Error())
		return
	}
}

func SQSDeleteQueue(q string) {
	c := AWSGetSQSClient()
	_, err := c.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(q),
	})
	if err != nil {
		log.Printf("Error retrieving queue URL [%s]: %s\n", q, err.Error())
		return
	}
	_, err = c.DeleteQueue(context.TODO(), &sqs.DeleteQueueInput{
		QueueUrl: aws.String(""),
	})
	if err != nil {
		log.Printf("Error deleting queue [%s]: %s\n", q, err.Error())
		return
	}
}

func SQSCreateQueue(name string) (string, error) {
	memberId := os.Getenv("AWS_MEMBER_ID")
	region := os.Getenv("AWS_REGION")
	jpolicy := map[string]any{
		"Version": "2012-10-17",
		"Id":      "__default_policy_ID",
		"Statement": []map[string]any{
			{
				"Sid":    "__owner_statement",
				"Effect": "Allow",
				"Principal": map[string]string{
					"AWS": GetIAMUserArn(os.Getenv("AWS_IAM_USER")),
				},
				"Action":   "SQS:*",
				"Resource": GetQueueArn(name),
			},
			{
				"Sid":    fmt.Sprintf("topic-subscription-arn:aws:sns:%s:%s:%s", region, memberId, name),
				"Effect": "Allow",
				"Principal": map[string]string{
					"Service": "sns.amazonaws.com",
				},
				"Action":   "SQS:SendMessage",
				"Resource": GetQueueArn(name),
				"Condition": map[string]any{
					"ArnLike": map[string]string{
						"aws:SourceArn": GetTopicArn(name),
					},
				},
			},
		},
	}

	bpolicy, _ := json.Marshal(jpolicy)
	policy := string(bpolicy)
	c := AWSGetSQSClient()
	out, err := c.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: aws.String(name),
		Attributes: map[string]string{
			"DelaySeconds":                  "0",
			"ReceiveMessageWaitTimeSeconds": "20",
			"Policy":                        policy,
		},
	})
	if err != nil {
		log.Printf("Error creating queue %s: %s\n", name, err.Error())
		return "", err
	}
	log.Printf("[%s] Queue has been created\n", name)
	return *out.QueueUrl, nil
}

func SNSSubscribe(topic string, q string, proto string) {
	topicArn := GetTopicArn(topic)
	qArn := q
	if proto == "sqs" {
		qArn = GetQueueArn(q)
	}
	client := AWSGetSNSClient()
	output, err := client.Subscribe(context.Background(), &sns.SubscribeInput{
		Protocol: aws.String(proto),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(qArn),
	})
	if err != nil {
		log.Printf("Error subscribing to topic: %s\n", err.Error())
		return
	}
	log.Printf("Subscribed to topic: %s\n", *output.SubscriptionArn)
}

func GetTopicArn(topic string) string {
	memberId := os.Getenv("AWS_MEMBER_ID")
	region := os.Getenv("AWS_REGION")
	topicArn := fmt.Sprintf("arn:aws:sns:%s:%s:%s", region, memberId, topic)
	return topicArn
}

func GetQueueArn(q string) string {
	memberId := os.Getenv("AWS_MEMBER_ID")
	region := os.Getenv("AWS_REGION")
	qArn := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", region, memberId, q)
	return qArn
}

func GetIAMUserArn(user string) string {
	memberId := os.Getenv("AWS_MEMBER_ID")
	arn := fmt.Sprintf("arn:aws:iam::%s:%s/%s", memberId, "user", user)
	return arn
}
