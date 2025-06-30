package controllers

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

func PasskeyLoginStart(ctx *gin.Context) (opts *protocol.CredentialAssertion, status int, err error) {
	var body struct {
		Email string `json:"email" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return nil, http.StatusBadRequest, err
	}
	var user models.User
	db := db.GetDb()
	if err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.User{}).
			Where("email = ?", body.Email).
			First(&user).
			Error; err != nil {
			return err
		}
		if err := utils.GetCredentials(&user); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, http.StatusBadRequest, err
	}
	if len(user.Credentials) == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "No credentials registered"})
		return
	}
	wa, _ := lib.GetWebAuthn()
	opts, ses, err := wa.BeginLogin(user)
	if err != nil {
		log.Printf("Could not initialize login with passkey: %s\n", err.Error())
		ctx.Status(http.StatusInternalServerError)
		return
	}
	rd := lib.GetRedisClient()
	rd.JSONSet(context.Background(), fmt.Sprintf("%d:passkey:login", user.ID), "$", ses)
	return opts, http.StatusOK, nil
}
func PasskeyLoginFinish(ctx *gin.Context) (token *string, status int, err error) {
	var query struct {
		Email string `form:"email" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&query); err != nil {
		log.Printf("Error validating request: %s\n", err.Error())
		return nil, http.StatusBadRequest, err
	}
	var user models.User
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Model(&models.User{}).
			Where("email = ?", query.Email).
			First(&user).
			Error; err != nil {
			return err
		}
		if err := utils.GetCredentials(&user); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("Error retrieving user [%s]: %s\n", query.Email, err.Error())
		ctx.Status(http.StatusBadRequest)
		return nil, http.StatusBadRequest, err
	}
	rd := lib.GetRedisClient()
	val, err := rd.JSONGet(context.Background(), fmt.Sprintf("%d:passkey:login", user.ID)).Result()
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	if val == "" {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	var ses webauthn.SessionData
	json.Unmarshal([]byte(val), &ses)
	wa, _ := lib.GetWebAuthn()
	_, err = wa.FinishLogin(user, ses, ctx.Request)
	if err != nil {
		log.Printf("Passkey login failed: %s\n", err.Error())
		ctx.Status(http.StatusUnauthorized)
		return nil, http.StatusUnauthorized, err
	}
	jwt, err := utils.GenerateJWT(user.Email, user.ID, user.ActiveOrg)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	log.Printf("[jwt]: %s\n", jwt)

	uid := ctx.GetString("uid")
	go func() {
		rd := lib.GetRedisClient()
		_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:user", user.ID), "$", &user).Result()
		if err != nil {
			log.Printf("[redis] Error updating user cache: %s\n", err.Error())
		}
		token := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
		fcm, _ := lib.GetFirebaseMessaging()
		fcm.SubscribeToTopic(ctx.Copy(), []string{token}, "Notifications")
	}()
	return &jwt, http.StatusOK, nil
}
