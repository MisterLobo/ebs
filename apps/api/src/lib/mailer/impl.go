package mailer

import (
	"ebs/src/lib"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"fmt"
	"os"
)

func NewMailerMessage(input *lib.SendMailInput) error {
	emailQueue := os.Getenv("EMAIL_QUEUE")
	apiEnv := os.Getenv("API_ENV")
	emailBody := &types.JSONB{
		"from":      input.From,
		"from-name": input.FromName,
		"to":        input.To,
		"cc":        input.Cc,
		"bcc":       input.Bcc,
		"reply-to":  input.ReplyTo,
		"body":      input.Body,
		"html":      input.Html,
		"subject":   input.Subject,
	}
	if apiEnv == "local" {
		if err := lib.KafkaProduceMessage("emails", utils.WithSuffix(emailQueue), emailBody); err != nil {
			return fmt.Errorf("error sending message to queue: %s", err.Error())
		}
	}
	body, err := json.Marshal(&emailBody)
	if err != nil {
		return err
	}
	if err := lib.SQSProduceMessage(utils.WithSuffix(emailQueue), string(body)); err != nil {
		return fmt.Errorf("error sending message to queue: %s", err.Error())
	}
	return nil
}
