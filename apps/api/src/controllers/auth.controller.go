package controllers

import (
	"context"
	"crypto/rand"
	"ebs/src/config"
	"ebs/src/db"
	"ebs/src/lib"
	"ebs/src/models"
	"ebs/src/types"
	"ebs/src/utils"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"gorm.io/gorm"
)

func AuthLogin(ctx *gin.Context) (token *string, status int, err error) {
	var body types.RegisterUserRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return nil, http.StatusBadRequest, err
	}
	auth, err := lib.GetFirebaseAuth()
	if err != nil {
		log.Printf("Error initializing FirebaseAuth client: %s\n", err.Error())
		return nil, http.StatusBadRequest, err
	}
	user, err := auth.GetUserByEmail(context.Background(), body.Email)
	if err != nil {
		log.Printf("error from Firebase: %s\n", err.Error())
		return nil, http.StatusNotFound, err
	}

	db := db.GetDb()
	var muser models.User
	if err = db.
		Model(&models.User{}).
		Select("id", "name", "email").
		Where(&models.User{Email: user.Email}).
		First(&muser).
		Error; err != nil {
		log.Printf("error: %s\n", err.Error())
		return nil, http.StatusNotFound, err
	}

	uid := ctx.GetString("uid")
	rd := lib.GetRedisClient()
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := db.
			Model(&models.User{}).
			Where("id", muser.ID).
			Update("last_active", time.Now()).
			Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Error logging in user [%d]: %s\n", muser.ID, err.Error())
		ctx.Status(http.StatusBadRequest)
		return nil, http.StatusBadRequest, err
	}
	err = utils.GetCredentials(&muser)
	if err != nil {
		log.Printf("Could not retrieve credentials for user [%d]: %s\n", muser.ID, err.Error())
		return nil, http.StatusBadRequest, err
	}
	if ctx.Request.Header.Get("origin") != "app:mobile" && len(muser.StoredCredentials) > 0 {
		flowId := uuid.NewString()
		bNonce := make([]byte, 32)
		rand.Read(bNonce)
		secret, _ := hex.DecodeString(config.API_SECRET)
		nonce := hex.EncodeToString(bNonce)
		enc, err := utils.EncryptMessage(secret, nonce)
		if err != nil {
			log.Printf("Error encrypting message: %s\n", err.Error())
			return nil, http.StatusInternalServerError, err
		}
		rd.JSONSet(ctx, fmt.Sprintf("%s:mfa_state", user.UID), "$", &map[string]any{
			"nonce":     enc,
			"state":     "pending",
			"flow_id":   flowId,
			"user_id":   int(muser.ID),
			"timestamp": time.Now().UnixMilli(),
		})
		exp := 5 * time.Minute
		rd.Expire(ctx, "", exp)
		rd.Set(ctx, fmt.Sprintf("%d:mfa_state", muser.ID), fmt.Sprintf("%s:mfa_state", user.UID), exp)
		ctx.Header("X-Authenticate-MFA", "true")
		ctx.Header("X-MFA-Flow-ID", flowId)
		ctx.Header("X-MFA-Challenge", nonce)
		log.Println("Credentials found: initializing secondary auth")
		return nil, http.StatusUnauthorized, nil
	}

	jwt, _ := utils.GenerateJWT(user.Email, muser.ID, muser.ActiveOrg)

	_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:user", muser.ID), "$", &muser).Result()
	if err != nil {
		log.Printf("[redis] Error updating user cache: %s\n", err.Error())
	}
	_, err = rd.JSONSet(ctx, fmt.Sprintf("%d:meta", muser.ID), "$", map[string]string{"photoURL": user.PhotoURL}).Result()
	if err != nil {
		log.Printf("[redis] Error updating user cache: %s\n", err.Error())
	}
	val := rd.JSONGet(context.Background(), fmt.Sprintf("%s:fcm", uid), "$.token").Val()
	fcm, _ := lib.GetFirebaseMessaging()
	fcm.SubscribeToTopic(ctx, []string{val}, "Notifications")

	return &jwt, http.StatusOK, nil
}

func AuthRegister(ctx *gin.Context) (uid *string, status int, err error) {
	var body types.RegisterUserRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return nil, http.StatusBadRequest, err
	}
	auth, err := lib.GetFirebaseAuth()
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	user, err := auth.GetUserByEmail(context.Background(), body.Email)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	db := db.GetDb()
	err = db.Transaction(func(tx *gorm.DB) error {
		var muser models.User
		if err := tx.
			Model(&models.User{}).
			Select("tenant_id").
			Where("email = ?", body.Email).
			First(&muser).
			Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("could not complete transaction")
			}
		}
		if muser.TenantID != nil {
			err := errors.New("user is already registered in the system. Please proceed to Log In")
			log.Printf("error: %s\n", err.Error())
			return err
		}

		newUser := models.User{
			Email: user.Email,
			UID:   user.UID,
			Role:  types.ROLE_OWNER,
			Name:  user.DisplayName,
		}
		if err := tx.Create(&newUser).Error; err != nil {
			log.Printf("Error creating user: %s\n", err.Error())
			return fmt.Errorf("error creating user: %s", user.Email)
		}

		newOrg := models.Organization{
			Name:         fmt.Sprintf("%s's organization", user.DisplayName),
			OwnerID:      newUser.ID,
			Type:         types.ORG_PERSONAL,
			ContactEmail: user.Email,
			TenantID:     newUser.TenantID,
			Status:       "active",
		}
		if err := tx.Create(&newOrg).Error; err != nil {
			return err
		}

		newTeam := models.Team{
			OrganizationID: newOrg.ID,
			OwnerID:        newUser.ID,
			Name:           "Default",
			Status:         "active",
		}
		if err := tx.Create(&newTeam).Error; err != nil {
			return err
		}
		sc := lib.GetStripeClient()
		acc, err := sc.V1Accounts.Create(context.Background(), &stripe.AccountCreateParams{
			BusinessProfile: &stripe.AccountCreateBusinessProfileParams{
				Name:         stripe.String(newOrg.Name),
				SupportEmail: stripe.String(newOrg.ContactEmail),
			},
			BusinessType: stripe.String("individual"),
			Company: &stripe.AccountCreateCompanyParams{
				Name: stripe.String(newOrg.Name),
			},
			Type:     stripe.String("express"),
			Email:    stripe.String(newOrg.ContactEmail),
			Metadata: map[string]string{"organizationId": fmt.Sprintf("%d", newOrg.ID)},
			Capabilities: &stripe.AccountCreateCapabilitiesParams{
				CardPayments: &stripe.AccountCreateCapabilitiesCardPaymentsParams{
					Requested: stripe.Bool(true),
				},
				Transfers: &stripe.AccountCreateCapabilitiesTransfersParams{
					Requested: stripe.Bool(true),
				},
			},
		})
		if err != nil {
			log.Printf("Error creating account for organization: %s\n", err.Error())
			return errors.New("error creating account for organization")
		}
		if err := tx.
			Model(&models.Organization{}).
			Where("id = ?", newOrg.ID).
			Updates(&models.Organization{
				StripeAccountID: &acc.ID,
			}).Error; err != nil {
			log.Printf("Error creating Connect account: %s\n", err.Error())
		}

		err = tx.
			Model(&models.User{}).
			Where(&models.User{ID: newUser.ID}).
			Update("active_org", newOrg.ID).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, http.StatusBadRequest, err
	}
	return &user.UID, http.StatusOK, nil
}
