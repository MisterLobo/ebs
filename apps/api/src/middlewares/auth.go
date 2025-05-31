package middlewares

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"log"
	"net/http"
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
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	reqToken := strings.Split(bearerToken, " ")[1]
	if reqToken == "" {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	claims := &types.Claims{}
	tkn, err := jwt.ParseWithClaims(reqToken, claims, func(t *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		log.Printf("token error: %s\n", err.Error())
		if err == jwt.ErrSignatureInvalid || err == jwt.ErrTokenMalformed {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	if !tkn.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.Println("sub:", claims.Subject)
	db := db.GetDb()
	var user models.User
	uid, err := strconv.Atoi(claims.Subject)
	if err != nil {
		log.Println("error parsing claims:", err.Error())
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	err = db.
		Model(&models.User{}).
		Where(&models.User{ID: uint(uid)}).
		Find(&user).
		Error
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	ctx.Set("email", user.Email)
	ctx.Set("id", user.ID)
	ctx.Set("uid", user.UID)
	ctx.Set("org", user.ActiveOrg)
	ctx.Set("role", user.Role)
	ctx.Set("perms", claims)
}
