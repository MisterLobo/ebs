package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func S3DownloadAsset(name string) error {
	assetsBucket := os.Getenv("S3_ASSETS_BUCKET")
	client := GetS3Client()
	result, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(assetsBucket),
		Key:    aws.String(name),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			// return nil if key does not exist
			return nil
		}
		return err
	}
	defer result.Body.Close()
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Could not read working directory: %s\n", err.Error())
		return err
	}
	tempdir := os.Getenv("TEMP_DIR")
	filepath := path.Join(wd, tempdir, fmt.Sprintf("%s.jpeg", name))
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return err
	}
	_, err = file.Write(body)
	return err
}

func S3UploadAsset(name string, f string) (*string, error) {
	assetsBucket := os.Getenv("S3_ASSETS_BUCKET")
	file, err := os.Open(f)
	if err != nil {
		log.Printf("Could not open file to upload: %s\n", err.Error())
		return nil, err
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
		return nil, err
	}
	err = s3.NewObjectExistsWaiter(client).Wait(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(assetsBucket),
		Key:    aws.String(name),
	}, time.Minute)
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist: %s\n", name, err.Error())
		return nil, err
	}
	log.Printf("Added object '%s' to bucket '%s'", name, assetsBucket)
	pre := s3.NewPresignClient(client)
	r, err := pre.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(assetsBucket),
		Key:    aws.String(name),
	}, func(po *s3.PresignOptions) {
		po.Expires = time.Duration(3600 * time.Second)
	})
	if err != nil {
		log.Printf("Could not generate presigned URL for object [%s]: %s\n", name, err.Error())
		return nil, err
	}
	return &r.URL, nil
}
