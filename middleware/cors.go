package middleware

import (
	"net/http"

	"github.com/darwinOrg/go-common/constants"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var AllowMethods = []string{
	http.MethodGet,
	http.MethodPost,
}

var AllowOrigins = []string{
	"*",
}

var AllowHeaders = []string{
	constants.TraceId,
	constants.SpanId,
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
	constants.RemoteIp,
	constants.CompanyId,
	constants.Product,
	constants.Products,
	constants.DepartmentIds,
	constants.LoginPlatform,
	constants.TargetPlatform,
	constants.Ticket,
	constants.Source,
	constants.Since,
	constants.OutUserId,

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
	corsConfig.AllowMethods = AllowMethods
	corsConfig.AllowOrigins = AllowOrigins
	corsConfig.AllowHeaders = AllowHeaders

	return cors.New(corsConfig)
}
