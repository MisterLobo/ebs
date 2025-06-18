package mailer

import (
	"ebs/src/lib"
	"ebs/src/types"
	"fmt"
	"os"
)

func NewMailerMessage(input *lib.SendMailInput) error {
	emailQueue := os.Getenv("EMAIL_QUEUE")
	if err := lib.KafkaProduceMessage("emails", emailQueue, &types.JSONB{
		"from":      input.From,
		"from-name": input.FromName,
		"to":        input.To,
		"cc":        input.Cc,
		"bcc":       input.Bcc,
		"reply-to":  input.ReplyTo,
		"body":      input.Body,
		"html":      input.Html,
		"subject":   input.Subject,
	}); err != nil {
		return fmt.Errorf("error sending message to queue: %s", err.Error())
	}
	return nil
}
