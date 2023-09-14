package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var AllowOrigins = []string{
	"*",
}

var AllowHeaders = []string{
	"device-id",
	"hardware",
	"os",
	"os_version",
	"location",
	"ip",
	"network_type",
	"timestamp",
	"user_agent",
	"resolution",
	"platform",
	"app_key",
	"app_version",
	"app_vsn",
	"trace_id",
	"token",
	"startrek_token",
	"X-Forwarded-For",
	"X-Forwarded-Proto",
}

func Cors() gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = AllowOrigins
	corsConfig.AllowHeaders = AllowHeaders

	return cors.New(corsConfig)
}
