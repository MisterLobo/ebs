package lib

import (
	"context"
	"log"
	"os"
	"path"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var innerApp *firebase.App
var innerAuth *auth.Client
var innerMessaging *messaging.Client

func getOpts() *option.ClientOption {
	secretsPath := os.Getenv("SECRETS_DIR")
	opt := option.WithCredentialsFile(path.Join(secretsPath, "admin-sdk-credentials.json"))
	return &opt
}
func GetFirebaseAuth() (*auth.Client, error) {
	if innerAuth != nil {
		return innerAuth, nil
	}
	opt := getOpts()
	if innerApp == nil {
		app, err := firebase.NewApp(context.Background(), nil, *opt)
		if err != nil {
			log.Fatalf("error initializing app: %v\n", err.Error())
		}
		innerApp = app
	}

	auth, err := innerApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing Firebase Auth: %v\n", err.Error())
	}
	innerAuth = auth

	return auth, nil
}

func GetFirebaseMessaging() (*messaging.Client, error) {
	if innerMessaging != nil {
		return innerMessaging, nil
	}
	opt := getOpts()
	if innerApp == nil {
		app, err := firebase.NewApp(context.Background(), nil, *opt)
		if err != nil {
			log.Fatalf("error intializing app: %v\n", err.Error())
		}
		innerApp = app
	}

	msg, err := innerApp.Messaging(context.Background())
	if err != nil {
		log.Fatalf("error initializing FCM: %v\n", err.Error())
	}
	innerMessaging = msg
	return msg, nil
}

func NewFirebaseApp(app *firebase.App) {
	innerApp = app
	auth, err := innerApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing Firebase Auth: %s\n", err.Error())
	}
	innerAuth = auth
}
