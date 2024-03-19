package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var AllowOrigins = []string{
	"*",
}

var AllowHeaders = []string{
	"profile",
	"trace-id",
	"uid",
	"op-id",
	"run-as",
	"roles",
	"biz-types",
	"group-id",
	"platform",
	"user_agent",
	"lang",
	"goid",
	"pageNo",
	"pageSize",
	"token",
	"s-token",
	"remote-ip",
	"company-id",
	"product",
	"department-ids",

	"device-id",
	"hardware",
	"os",
	"os_version",
	"location",
	"ip",
	"network_type",
	"timestamp",
	"resolution",
	"app_key",
	"app_version",
	"app_vsn",
	"X-Forwarded-For",
	"X-Forwarded-Proto",
	"Authorization",
}

func Cors() gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = AllowOrigins
	corsConfig.AllowHeaders = AllowHeaders

	return cors.New(corsConfig)
}
