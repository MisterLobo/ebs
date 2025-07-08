package controllers

import (
	"context"
	"crypto/subtle"
	"ebs/src/config"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func PasskeyLoginStart(ctx *gin.Context) (opts *protocol.CredentialAssertion, status int, err error) {
	hFlowId := ctx.Request.Header.Get("X-MFA-Flow-ID")
	rd := lib.GetRedisClient()
	uid := ctx.GetString("uid")
	mfaState := rd.JSONGet(ctx, fmt.Sprintf("%s:mfa_state", uid)).Val()
	var state map[string]any
	json.Unmarshal([]byte(mfaState), &state)
	realFlowId := gjson.Get(mfaState, "flow_id").String()
	if subtle.ConstantTimeCompare([]byte(hFlowId), []byte(realFlowId)) != 1 {
		log.Printf("[FLOW-ID]: expected=%s got=%s", realFlowId, hFlowId)
		return nil, http.StatusUnauthorized, err
	}
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
	rd.JSONSet(context.Background(), fmt.Sprintf("%d:passkey:login", user.ID), "$", ses)
	return opts, http.StatusOK, nil
}
func PasskeyLoginFinish(ctx *gin.Context) (token *string, status int, err error) {
	hFlowId := ctx.Request.Header.Get("X-MFA-Flow-ID")
	hChallenge := ctx.Request.Header.Get("X-MFA-Challenge")
	rd := lib.GetRedisClient()
	uid := ctx.GetString("uid")
	mfaStateKey := fmt.Sprintf("%s:mfa_state", uid)
	state, err := rd.JSONGet(ctx, mfaStateKey).Result()
	if err != nil {
		log.Printf("Error reading state cache: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	var mfaState map[string]any
	if err := json.Unmarshal([]byte(state), &mfaState); err != nil {
		log.Printf("Could not read state: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	realFlowId := mfaState["flow_id"].(string)
	if subtle.ConstantTimeCompare([]byte(hFlowId), []byte(realFlowId)) != 1 {
		log.Printf("Flow ID mismatch: expected=%s got=%s", realFlowId, hFlowId)
		return nil, http.StatusUnauthorized, errors.New("access denied")
	}
	hNonce, err := hex.DecodeString(hChallenge)
	if err != nil {
		log.Printf("Error decoding challenge: %s\n", err.Error())
		return nil, http.StatusBadRequest, errors.New("invalid request")
	}
	encNonce := mfaState["nonce"].(string)
	secret, err := hex.DecodeString(config.API_SECRET)
	if err != nil {
		log.Printf("Error reading secret: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	decNonce, err := utils.DecryptMessage(secret, encNonce)
	if err != nil {
		log.Printf("Error decrypting message: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	nonce, err := hex.DecodeString(*decNonce)
	if err != nil {
		log.Printf("Error decoding nonce: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	if subtle.ConstantTimeCompare(hNonce, nonce) != 1 {
		log.Printf("Nonce mismatch: expected=%s got=%s\n", nonce, hNonce)
		return nil, http.StatusUnauthorized, errors.New("access denied")
	}
	mfaState["state"] = "complete"
	_, err = rd.JSONSet(ctx, mfaStateKey, "$", mfaState).Result()
	if err != nil {
		log.Printf("Error updating MFA state: %s\n", err.Error())
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	var query struct {
		Email string `form:"email" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&query); err != nil {
		log.Printf("Error validating request: %s\n", err.Error())
		return nil, http.StatusBadRequest, errors.New("invalid request")
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
		return nil, http.StatusBadRequest, errors.New("invalid request")
	}
	val, err := rd.JSONGet(context.Background(), fmt.Sprintf("%d:passkey:login", user.ID)).Result()
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	if val == "" {
		return nil, http.StatusInternalServerError, errors.New("something went wrong")
	}
	var ses webauthn.SessionData
	json.Unmarshal([]byte(val), &ses)
	wa, _ := lib.GetWebAuthn()
	_, err = wa.FinishLogin(user, ses, ctx.Request)
	if err != nil {
		log.Printf("Passkey login failed: %s\n", err.Error())
		return nil, http.StatusUnauthorized, errors.New("access denied")
	}
	jwt, err := utils.GenerateJWT(user.Email, user.ID, user.ActiveOrg)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid request")
	}
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
