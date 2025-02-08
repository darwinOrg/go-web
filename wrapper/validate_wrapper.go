package wrapper

import (
	ve "github.com/darwinOrg/go-validator-ext"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

var candidateValidatorTags = []string{"remark", "json", "form", "label"}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			for _, t := range candidateValidatorTags {
				tagValue := fld.Tag.Get(t)
				if tagValue != "" && tagValue != "-" {
					tagName := strings.SplitN(tagValue, ",", 2)[0]
					if tagName != "" && tagName != "-" {
						return tagName
					}
				}
			}

			return fld.Name
		})

		ve.AddCustomRules(v)
		ve.CustomValidator = v
	}
}
