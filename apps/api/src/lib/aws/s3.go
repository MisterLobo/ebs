package aws

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3Client() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Could not load default config: %s\n", err.Error())
		return nil
	}
	svc := s3.NewFromConfig(cfg)
	return svc
}

func S3UploadAsset(name string, f string) error {
	assetsBucket := os.Getenv("S3_ASSETS_BUCKET")
	file, err := os.Open(f)
	if err != nil {
		log.Printf("Could not open file to upload: %s\n", err.Error())
		return err
	}
	defer file.Close()
	client := GetS3Client()
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(assetsBucket),
		Key:         aws.String(name),
		Body:        file,
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		log.Printf("Could not put object to S3 bucket: %s\n", err.Error())
		return err
	}
	err = s3.NewObjectExistsWaiter(client).Wait(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(assetsBucket),
		Key:    aws.String(name),
	}, time.Minute)
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist: %s\n", name, err.Error())
		return err
	}
	log.Printf("Added object '%s' to bucket '%s'", name, assetsBucket)
	return nil
}
