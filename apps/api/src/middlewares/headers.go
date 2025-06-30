package middlewares

import (
	"os"

	"github.com/gin-gonic/gin"
)

var (
	whitelist = []string{
		os.Getenv("APP_HOST"),
		os.Getenv("API_HOST"),
		"localhost:9090",
	}
)

func SecureHeaders(ctx *gin.Context) {
	/* log.Println(ctx.Request.Host)
	if !slices.Contains(whitelist, ctx.Request.Host) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid host header"})
		return
	} */
	ctx.Header("X-Frame-Options", "DENY")
	ctx.Header("Content-Security-Policy", "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';")
	ctx.Header("X-XSS-Protection", "1; mode=block")
	ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	ctx.Header("Referrer-Policy", "strict-origin")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Permissions-Policy", "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=(),magnetometer=(),gyroscope=(),fullscreen=(self),payment=()")
}
