package middlewares

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))
var tokens []string

func AuthMiddleware(ctx *gin.Context) {
	bearerToken := ctx.Request.Header.Get("Authorization")
	if !strings.HasPrefix(bearerToken, "Bearer") {
		ctx.AbortWithStatus(401)
		return
	}
	reqToken := strings.Split(bearerToken, " ")[1]
	if reqToken == "" {
		ctx.AbortWithStatus(401)
	}
	claims := &types.Claims{}
	tkn, err := jwt.ParseWithClaims(reqToken, claims, func(t *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		log.Printf("token error: %s\n", err.Error())
		if err == jwt.ErrSignatureInvalid || err == jwt.ErrTokenMalformed {
			ctx.AbortWithStatus(401)
			return
		}
		ctx.AbortWithError(401, err)
		return
	}
	if !tkn.Valid {
		ctx.AbortWithStatus(401)
		return
	}

	log.Println("sub:", claims.Subject)
	db := db.GetDb()
	var user models.User
	uid, err := strconv.Atoi(claims.Subject)
	if err != nil {
		log.Println("error parsing claims:", err.Error())
		ctx.AbortWithStatus(401)
	}
	db.Model(&models.User{}).Where(&models.User{ID: uint(uid)}).Find(&user)

	if uint(uid) != user.ID || user.ID < 1 {
		ctx.AbortWithStatus(401)
		return
	}
	ctx.Set("email", user.Email)
	ctx.Set("id", user.ID)
	ctx.Set("uid", user.UID)
	ctx.Set("org", user.ActiveOrg)
	ctx.Set("role", user.Role)
}
