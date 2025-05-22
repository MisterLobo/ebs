package aws

import (
	"context"
	"log"

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

func SESSendMessage(from *string, destination *types.Destination, message *types.Message) {
	c := GetSESClient()
	input := &ses.SendEmailInput{
		Destination: destination,
		Source:      from,
		Message:     message,
	}
	out, err := c.SendEmail(context.TODO(), input)
	if err != nil {
		log.Printf("Error sending email: %s\n", err.Error())
		return
	}
	log.Printf("Sent email with id: %s\n", *out.MessageId)
}
