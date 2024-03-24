package wrapper

import (
	"encoding/json"
	"fmt"
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/go-openapi/spec"
	"net/http"
	"os"
	"reflect"
	"strings"
)

const (
	contentTypeJson = "application/json"
	bodyKey         = "body"
	responseOk      = 200
)

var (
	requestApis []*requestApi
)

type requestApi struct {
	method         string
	basePath       string
	relativePath   string
	apiDir         string
	remark         string
	requestObject  any
	responseObject any
}

type GenerateSwaggerRequest struct {
	Namespace   string
	Title       string
	Description string
	ServiceName string
	OutDir      string
	Version     string
}

func appendRequestApi[T any, V any](rh *RequestHolder[T, V], method string) {
	if !dgsys.IsQa() && !dgsys.IsProd() {
		requestApis = append(requestApis, &requestApi{
			method:         method,
			basePath:       rh.BasePath(),
			relativePath:   rh.RelativePath,
			apiDir:         rh.ApiDir,
			remark:         rh.Remark,
			requestObject:  new(T),
			responseObject: new(V),
		})
	}
}

func GenerateSwaggerFile(req *GenerateSwaggerRequest) {
	if len(requestApis) == 0 {
		return
	}
	if req.Version == "" {
		req.Version = "v1.0.0"
	}

	swaggerProps := spec.SwaggerProps{
		Swagger:             "2.0",
		Definitions:         spec.Definitions{},
		SecurityDefinitions: spec.SecurityDefinitions{},
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title:       fmt.Sprintf("%s%s", req.Namespace, req.Title),
				Description: fmt.Sprintf("%s%s", req.Namespace, req.Description),
				Version:     req.Version,
			},
		},
		Paths: buildApiPaths(),
	}

	filename := fmt.Sprintf("%s/%s.swagger.json", req.OutDir, req.ServiceName)
	saveSwagger(swaggerProps, filename)
}

func buildApiPaths() *spec.Paths {
	paths := map[string]spec.PathItem{}

	for _, api := range requestApis {
		url := fmt.Sprintf("%s/%s", api.basePath, api.relativePath)
		url = strings.ReplaceAll(url, "//", "/")
		var parameters []spec.Parameter
		if api.method == http.MethodGet {
			parameters = buildGetParameters(api)
		} else {
			parameters = buildPostParameters(api)
		}

		paths[url] = spec.PathItem{
			PathItemProps: spec.PathItemProps{
				Post: &spec.Operation{
					OperationProps: spec.OperationProps{
						Summary:    api.remark,
						Consumes:   []string{contentTypeJson},
						Produces:   []string{contentTypeJson},
						Parameters: parameters,
						Responses:  buildResponses(api),
					},
				},
			},
		}
	}

	return &spec.Paths{
		Paths: paths,
	}
}

func buildGetParameters(api *requestApi) []spec.Parameter {
	tpe := reflect.TypeOf(api.requestObject)
	if tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
	}
	cnt := tpe.NumField()
	var parameters []spec.Parameter

	for i := 0; i < cnt; i++ {
		field := tpe.Field(i)
		p := *spec.QueryParam(extractNameFromField(field))
		parameters = append(parameters, p)

		switch field.Type.Kind() {
		case reflect.String:
			p.Type = "string"
		case reflect.Bool:
			p.Type = "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			p.Type = "integer"
		case reflect.Float32, reflect.Float64:
			p.Type = "number"
		case reflect.Slice, reflect.Array:
			p.Type = "array"
		default:
			fmt.Printf("Unsupported field type: %s\n", field.Type.Kind())
		}

		p.Required = extractRequiredFlagFromField(field)
		p.Description = extractDescriptionFromField(field)
	}

	return parameters
}

func buildPostParameters(api *requestApi) []spec.Parameter {
	bodySchema := createSchemaForType(reflect.TypeOf(api.requestObject))
	bodyParam := *spec.BodyParam(bodyKey, bodySchema)
	bodyParam.Required = true
	return []spec.Parameter{bodyParam}
}

func createSchemaForType(tpe reflect.Type) *spec.Schema {
	if tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
	}

	schema := &spec.Schema{}
	switch tpe.Kind() {
	case reflect.String:
		schema.Type = []string{"string"}
	case reflect.Bool:
		schema.Type = []string{"boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = []string{"integer"}
	case reflect.Float32, reflect.Float64:
		schema.Type = []string{"number"}
	case reflect.Slice, reflect.Array:
		elemType := tpe.Elem()
		itemSchema := createSchemaForType(elemType)
		schema.Type = []string{"array"}
		schema.Items = &spec.SchemaOrArray{Schema: itemSchema}
	case reflect.Map:
		keyType := tpe.Key()
		if keyType.Kind() != reflect.String {
			panic("Map keys must be strings in OpenAPI schemas.")
		}
		valueType := tpe.Elem()
		valueSchema := createSchemaForType(valueType)
		schema.Type = []string{"object"}
		schema.AdditionalProperties = &spec.SchemaOrBool{
			Allows: true,
			Schema: valueSchema,
		}
	case reflect.Struct:
		schema.Properties = make(map[string]spec.Schema)
		schema.Required = make([]string, 0)
		cnt := tpe.NumField()

		for i := 0; i < cnt; i++ {
			field := tpe.Field(i)
			property := createSchemaForType(field.Type)
			property.Title = extractNameFromField(field)
			property.Description = extractDescriptionFromField(field)
			fieldName := extractNameFromField(field)
			schema.Properties[fieldName] = *property

			if extractRequiredFlagFromField(field) {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	default:
		fmt.Printf("Unsupported field type: %s\n", tpe.Kind())
	}

	return schema
}

func buildResponses(api *requestApi) *spec.Responses {
	return &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: map[int]spec.Response{
				responseOk: {
					ResponseProps: spec.ResponseProps{
						Description: "成功",
						Schema:      createSchemaForType(reflect.TypeOf(api.responseObject)),
					},
				},
			},
		},
	}
}

func saveSwagger(swaggerProps spec.SwaggerProps, filename string) {
	swaggerJSON, err := json.MarshalIndent(swaggerProps, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filename, swaggerJSON, 0644)
	if err != nil {
		panic(err)
	}
}

func extractNameFromField(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		return jsonTag
	} else {
		return field.Name
	}
}

func extractRequiredFlagFromField(field reflect.StructField) bool {
	bindingTag := field.Tag.Get("binding")
	return bindingTag != "" && strings.Contains(bindingTag, "required")
}

func extractDescriptionFromField(field reflect.StructField) string {
	return field.Tag.Get("remark")
}
