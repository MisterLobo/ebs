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

func GetFirebaseAuth() (*auth.Client, error) {
	cwd, _ := os.Getwd()
	log.Println("cwd:", cwd)
	opt := option.WithCredentialsFile(path.Join(cwd, "admin-sdk-credentials.json"))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err.Error())
		return nil, err
	}

	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing auth: %v\n", err.Error())
		return nil, err
	}

	return auth, nil
}
