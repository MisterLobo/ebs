package lib

import (
	"os"

	"github.com/pusher/pusher-http-go/v5"
)

var pusherClient *pusher.Client

func GetPusherClient() *pusher.Client {
	if pusherClient != nil {
		return pusherClient
	}
	pusherClient = &pusher.Client{
		AppID:   os.Getenv("PUSHER_APP_ID"),
		Key:     os.Getenv("PUSHER_KEY"),
		Secret:  os.Getenv("PUSHER_SECRET"),
		Cluster: os.Getenv("PUSHER_CLUSTER"),
	}
	return pusherClient
}
