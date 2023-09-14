package wrapper

import (
	ve "github.com/darwinOrg/go-validator-ext"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		ve.AddCustomRules(v)
		ve.CustomValidator = v
	}
}
