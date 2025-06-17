package middlewares

import (
	"context"
	"ebs/src/lib"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func VerifyIdToken(ctx *gin.Context) {
	idToken := ctx.GetHeader("Authorization")
	if idToken == "" {
		err := errors.New("missing authorization header")
		log.Printf("Check failed: %s\n", err.Error())
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	fauth, err := lib.GetFirebaseAuth()
	if err != nil {
		log.Printf("Error retrieving Firebase Auth instance: %s\n", err.Error())
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	token, err := fauth.VerifyIDToken(ctx, idToken)
	if err != nil {
		msg := "Failed to verify ID token"
		err := fmt.Errorf("failed to verify ID token: %s", err.Error())
		log.Printf("Failed to verify ID token: %v\n", err)
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
		return
	}
	rd := lib.GetRedisClient()

	rd.Set(context.Background(), fmt.Sprintf("%s:token", token.UID), idToken, 24*time.Hour)
	rd.JSONSet(context.Background(), token.UID, "$", token)
	// rd.ExpireAt(context.Background(), token.UID, time.Unix(token.Expires, 0))
	ctx.Set("uid", token.UID)
}
