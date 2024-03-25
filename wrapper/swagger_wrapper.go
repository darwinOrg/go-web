package wrapper

import (
	"encoding/json"
	"fmt"
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/go-openapi/spec"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	contentTypeJson = "application/json"
)
const (
	resultSchemaCode    = "code"
	resultSchemaMessage = "message"
	resultSchemaSuccess = "success"
	resultSchemaData    = "data"
)

var (
	requestApis []*requestApi
)

type requestApi struct {
	method         string
	basePath       string
	relativePath   string
	remark         string
	requestObject  any
	responseObject any
}

type ExportSwaggerFileRequest struct {
	ServiceName string
	Title       string
	Description string
	OutDir      string
	Version     string
}

func appendRequestApi[T any, V any](rh *RequestHolder[T, V], method string) {
	if !dgsys.IsQa() && !dgsys.IsProd() {
		requestApis = append(requestApis, &requestApi{
			method:         method,
			basePath:       rh.BasePath(),
			relativePath:   rh.RelativePath,
			remark:         rh.Remark,
			requestObject:  new(T),
			responseObject: new(V),
		})
	}
}

func ExportSwaggerFile(req *ExportSwaggerFileRequest) {
	if len(requestApis) == 0 {
		panic("没有需要导出的接口定义")
	}
	if req.ServiceName == "" {
		panic("服务名不能为空")
	}

	if req.Title == "" {
		req.Title = "接口文档"
	}
	if req.Description == "" {
		req.Description = "接口描述"
	}
	if req.OutDir == "" {
		req.OutDir = "openapi/v1"
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
				Title:       req.Title,
				Description: req.Description,
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
	for tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
	}
	cnt := tpe.NumField()
	var parameters []spec.Parameter

	for i := 0; i < cnt; i++ {
		field := tpe.Field(i)
		p := *spec.QueryParam(extractNameFromField(field))

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
		case reflect.Map:
			continue
		default:
			fmt.Printf("Unsupported field type: %s\n", field.Type.Kind())
		}

		p.Required = extractRequiredFlagFromField(field)
		p.Description = extractDescriptionFromField(field)

		parameters = append(parameters, p)
	}

	return parameters
}

func buildPostParameters(api *requestApi) []spec.Parameter {
	schema := createSchemaForType(reflect.TypeOf(api.requestObject))
	bodyParam := *spec.BodyParam("body", schema)
	bodyParam.Required = true
	return []spec.Parameter{bodyParam}
}

func createSchemaForType(tpe reflect.Type) *spec.Schema {
	for tpe.Kind() == reflect.Pointer {
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

			if strings.Contains(tpe.String(), "result.Result") && field.Name == "Data" {
				rt := reflect.New(tpe).Elem().Interface()
				dataType := reflect.ValueOf(rt).Field(i).Type()
				for dataType.Kind() == reflect.Pointer {
					dataType = dataType.Elem()
				}
				field.Type = dataType
			}

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
				http.StatusOK: {
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

	dirPath := filepath.Dir(filename)
	if err = os.MkdirAll(dirPath, os.ModePerm); err != nil {
		panic(err)
	}

	_, err = os.Create(filename)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filename, swaggerJSON, os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func extractNameFromField(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		return jsonTag
	} else {
		if len(field.Name) == 1 {
			return strings.ToLower(field.Name)
		}

		return strings.ToLower(field.Name[0:1]) + field.Name[1:]
	}
}

func extractRequiredFlagFromField(field reflect.StructField) bool {
	bindingTag := field.Tag.Get("binding")
	return bindingTag != "" && strings.Contains(bindingTag, "required")
}

func extractDescriptionFromField(field reflect.StructField) string {
	return field.Tag.Get("remark")
}
