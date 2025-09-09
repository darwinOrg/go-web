package middleware

import (
	"github.com/darwinOrg/go-common/constants"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var AllowOrigins = []string{
	"*",
}

var AllowHeaders = []string{
	constants.TraceId,
	constants.UID,
	constants.OpId,
	constants.RunAs,
	constants.Roles,
	constants.BizTypes,
	constants.GroupId,
	constants.Platform,
	constants.UserAgent,
	constants.Lang,
	constants.Token,
	constants.ShareToken,
	constants.CompanyId,
	constants.Product,
	constants.DepartmentIds,
	constants.Source,
	constants.Since,

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
