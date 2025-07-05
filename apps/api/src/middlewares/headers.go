package middlewares

import (
	"github.com/gin-gonic/gin"
)

func SecureHeaders(ctx *gin.Context) {
	ctx.Header("X-Frame-Options", "DENY")
	ctx.Header("Content-Security-Policy", "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';")
	ctx.Header("X-XSS-Protection", "1; mode=block")
	ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	ctx.Header("Referrer-Policy", "strict-origin")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Permissions-Policy", "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=(),magnetometer=(),gyroscope=(),fullscreen=(self),payment=()")
}
