package lib

import (
	"context"
	"log"
	"os"
	"path"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"google.golang.org/api/option"
)

var innerApp *firebase.App
var innerAuth *auth.Client

func GetFirebaseAuth() (*auth.Client, error) {
	cwd, _ := os.Getwd()
	log.Println("cwd:", cwd)
	secretsPath := os.Getenv("SECRETS_DIR")
	opt := option.WithCredentialsFile(path.Join(secretsPath, "admin-sdk-credentials.json"))
	if innerApp == nil {
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Fatalf("error initializing app: %v\n", err.Error())
			return nil, err
		}
		innerApp = app
	}

	if innerAuth == nil {
		auth, err := innerApp.Auth(context.Background())
		if err != nil {
			log.Fatalf("error initializing Firebase Auth: %v\n", err.Error())
			return nil, err
		}
		innerAuth = auth
	}

	return innerAuth, nil
}

func NewFirebaseApp(app *firebase.App) {
	innerApp = app
	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing Firebase Auth: %s\n", err.Error())
	}
	innerAuth = auth
}
