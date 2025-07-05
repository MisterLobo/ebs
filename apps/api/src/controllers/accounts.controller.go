package controllers

import (
	"context"
	"ebs/src/config"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/lib/mailer"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func AccountsPasskeyRegisterStart(ctx *gin.Context) (opts *protocol.CredentialCreation, status int, err error) {
	userId := ctx.GetUint("id")
	var user models.User
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&models.User{ID: userId}).First(&user).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, http.StatusBadRequest, err
	}
	wa, err := lib.GetWebAuthn()
	if err != nil {
		log.Printf("Failed to init webauthn: %s\n", err.Error())
		return nil, http.StatusInternalServerError, err
	}
	opts, ses, err := wa.BeginRegistration(
		user,
		webauthn.WithAuthenticatorSelection(wa.Config.AuthenticatorSelection),
	)
	if err != nil {
		log.Printf("Failed to begin registration: %s\n", err.Error())
		return nil, http.StatusInternalServerError, err
	}
	rd := lib.GetRedisClient()
	_, err = rd.JSONSet(context.Background(), fmt.Sprintf("%d:passkey:reg", userId), "$", ses).Result()
	if err != nil {
		log.Printf("Could not save session: %s\n", err.Error())
		return nil, http.StatusInternalServerError, err
	}
	return opts, http.StatusOK, nil
}

func AccountsPasskeyRegisterFinish(ctx *gin.Context) (status int, err error) {
	userId := ctx.GetUint("id")
	var user models.User
	db := db.GetDb()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&models.User{ID: userId}).First(&user).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return http.StatusBadRequest, err
	}
	rd := lib.GetRedisClient()
	val := rd.JSONGet(context.Background(), fmt.Sprintf("%d:passkey:reg", userId)).Val()
	var ses webauthn.SessionData
	json.Unmarshal([]byte(val), &ses)
	wa, _ := lib.GetWebAuthn()

	cred, err := wa.FinishRegistration(user, ses, ctx.Request)
	if err != nil {
		log.Printf("Could not finish passkey registration: %s\n", err.Error())
		return http.StatusInternalServerError, err
	}
	user.AddCredential(*cred)
	if err := utils.SaveCredentials(&user); err != nil {
		log.Printf("Failed to store credentials for user [%d]: %s\n", userId, err.Error())
		ctx.Status(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func AccountsVerify(ctx *gin.Context) (status int, err error) {
	var body struct {
		Type  string `json:"type" binding:"required"`
		Email string `json:"email" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return http.StatusBadRequest, err
	}
	email := body.Email
	tok := &models.Token{
		RequesterType: body.Type,
		Type:          "verification",
		TokenName:     "account_verification",
	}
	db := db.GetDb()
	switch body.Type {
	case "org":
		var org models.Organization
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where(&models.Organization{ContactEmail: email}).First(&org).Error; err != nil {
				return err
			}
			tok.RequestedBy = org.ID
			return nil
		}); err != nil {
			return http.StatusBadRequest, err
		}
	case "user":
		var user models.User
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where(&models.User{Email: email}).First(&user).Error; err != nil {
				return err
			}
			tok.RequestedBy = user.ID
			return nil
		}); err != nil {
			return http.StatusBadRequest, err
		}
	}
	reqId := uuid.NewString()
	payload := &types.JSONB{
		"id":  reqId,
		"sub": email,
		"ttl": 600,
		"dt":  time.Now().String(),
	}
	bPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Could not serialize JSON payload: %s\n", err.Error())
		return http.StatusInternalServerError, err
	}
	tx := db.Begin()
	token, err := utils.EncryptMessage([]byte(config.API_SECRET), string(bPayload))
	if err != nil {
		tx.Rollback()
		log.Printf("Payload encryption failed: %s\n", err.Error())
		return http.StatusInternalServerError, err
	}
	tok.TokenValue = types.JSONB{
		"token": token,
	}
	if err := tx.Create(tok).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, err
	}
	tx.Commit()
	senderFrom := os.Getenv("SMTP_FROM")
	verifyLink := fmt.Sprintf("%s/accounts/verify?token=%s", os.Getenv("APP_HOST"), token)
	input := &lib.SendMailInput{
		From:     senderFrom,
		FromName: "noreply",
		Subject:  "Verify Email",
		To:       []string{email},
		Body: fmt.Sprintf(`
					<p>You have requested an email verification. Please click the following link to proceed.</p>
					<a href="%s">verify email</a>
					<p>If link does not work, try copying the url below and pasting in your browser</p>
					<p>%s</p>
					`, verifyLink, verifyLink),
		Html: true,
	}
	if err := mailer.NewMailerMessage(input); err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

func AccountsConnectCalendar(ctx *gin.Context) error {
	return nil
}
