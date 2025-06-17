package lib

import (
	"log"
	"os"
	"strconv"

	"github.com/wneessen/go-mail"
)

func GetSMTPClient() (*mail.Client, error) {
	host := os.Getenv("SMTP_HOST")
	port := 587
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	c, err := mail.NewClient(host, mail.WithPort(port), mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(user), mail.WithPassword(pass))
	if err != nil {
		log.Printf("Could not initialize smtp client: %s\n", err.Error())
		return nil, err
	}
	return c, nil
}

func SMTPNewDefault() (*mail.Client, error) {
	return GetSMTPClient()
}

func SMTPNewSendGrid() (*mail.Client, error) {
	host := "smtp.sendgrid.net"
	portEnv := os.Getenv("SMTP_PORT")
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		port = 587
	}
	user := os.Getenv("SENDGRID_SMTP_USER")
	pass := os.Getenv("SENDGRID_API_KEY")
	c, err := mail.NewClient(
		host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(user),
		mail.WithPassword(pass),
	)
	if err != nil {
		log.Printf("Could not initialize smtp client: %s\n", err.Error())
		return nil, err
	}
	return c, nil
}

func SMTPNewGmail() (*mail.Client, error) {
	host := "smtp.gmail.com"
	user := os.Getenv("GMAIL_USERNAME")
	pass := os.Getenv("GMAIL_PASSWORD")
	port := 587
	c, err := mail.NewClient(
		host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(user),
		mail.WithPassword(pass),
	)
	if err != nil {
		log.Printf("Could not initialize smtp client: %s\n", err.Error())
		return nil, err
	}
	return c, nil
}

func SendMail(inputParams *SendMailInput) error {
	c, err := GetSMTPClient()
	if err != nil {
		return err
	}
	msg := mail.NewMsg()
	if err := msg.FromFormat(inputParams.FromName, inputParams.From); err != nil {
		log.Printf("Failed to set From address: %s\n", err.Error())
	}
	if err := msg.To(inputParams.To...); err != nil {
		log.Printf("Failed to set To address: %s\n", err.Error())
	}
	if err := msg.ReplyTo(inputParams.ReplyTo); err != nil {

	}
	if err := msg.Cc(inputParams.Cc...); err != nil {
		log.Printf("Failed to set Cc address: %s\n", err.Error())
	}
	if err := msg.Bcc(inputParams.Bcc...); err != nil {
		log.Printf("Failed to set Bcc address: %s\n", err.Error())
	}
	msg.Subject(inputParams.Subject)
	if inputParams.Html {
		msg.SetBodyString(mail.TypeTextHTML, inputParams.Body)
	} else {
		msg.SetBodyString(mail.TypeTextPlain, inputParams.Body)
	}
	if err := c.DialAndSend(msg); err != nil {
		return err
	}
	return nil
}

type SendMailInput struct {
	From     string
	FromName string
	To       []string
	Cc       []string
	Bcc      []string
	ReplyTo  string
	Subject  string
	Body     string
	Html     bool
}
