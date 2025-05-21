package aws

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func GetSESClient() *ses.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Could not load default config: %s\n", err.Error())
		return nil
	}
	svc := ses.NewFromConfig(cfg)
	return svc
}

func SESSendMessage() {
	c := GetSESClient()
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{
				os.Getenv("SNS_EMAIL"),
			},
		},
	}
	c.SendEmail(context.TODO(), input)
}
