package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/model"
)

// OpenAPI Root Object
type OpenAPI struct {
	OpenAPI string              `json:"openapi"`
	Info    Info                `json:"info"`
	Paths   map[string]PathItem `json:"paths"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type PathItem map[string]Operation // Key is method: "get", "post", etc.

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // "query", "path", "header"
	Required    bool   `json:"required,omitempty"`
	Schema      Schema `json:"schema"`
	Description string `json:"description,omitempty"`
}

type RequestBody struct {
	Content  map[string]MediaType `json:"content"`
	Required bool                 `json:"required,omitempty"`
}

type MediaType struct {
	Schema interface{} `json:"schema"` // Use interface{} for flexible schema
}

type Schema struct {
	Type string `json:"type"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// OpenAPIExporter constructs OpenAPI spec
type OpenAPIExporter struct {
	// Stateless
}

func NewOpenAPIExporter() *OpenAPIExporter {
	return &OpenAPIExporter{}
}

func (b *OpenAPIExporter) Export(summary *model.Summary, tree []*model.Node, cfg *config.Config) error {
	spec := OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Spec Recon API",
			Version: "1.0.0",
		},
		Paths: make(map[string]PathItem),
	}

	// 1. Leverage Analyzer to get High-Fidelity Endpoints (with schemas)
	endpoints := analyzer.ExtractEndpoints(tree, summary.ClassMap, summary.FieldTypeMap)

	// 2. Build Paths
	for _, endpoint := range endpoints {
		b.processEndpoint(&spec, endpoint)
	}

	// Determine output file
	dir := filepath.Dir(cfg.GetOutputPath())
	outputFile := filepath.Join(dir, "openapi.json")

	// Write to file
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(spec)
}

func (b *OpenAPIExporter) processEndpoint(spec *OpenAPI, endpoint model.EndpointDef) {
	fullPath := endpoint.Path
	if fullPath == "" {
		return
	}

	// Ensure path starts with /
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	method := strings.ToLower(endpoint.Method)
	if method == "" {
		method = "get"
	}

	// Initialize PathItem
	if _, ok := spec.Paths[fullPath]; !ok {
		spec.Paths[fullPath] = make(PathItem)
	}

	// Build Operation
	op := Operation{
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		OperationID: endpoint.ControllerName + "_" + endpoint.MethodName,
		Responses:   make(map[string]Response),
	}
	if op.Summary == "" {
		op.Summary = endpoint.MethodName
	}

	// 1. Process Parameters (Query, Path, Header, Body)
	for _, param := range endpoint.Params {
		lowerIn := strings.ToLower(param.In)

		if lowerIn == "body" {
			// Request Body
			schema := b.buildParamSchema(param)
			// Handle Flattened Fields if complex
			if len(param.Fields) > 0 {
				schema = b.buildComplexSchema(param.Fields)
			}

			op.RequestBody = &RequestBody{
				Content: map[string]MediaType{
					"application/json": {
						Schema: schema,
					},
				},
				Required: param.Required,
			}
		} else {
			// Standard Parameter
			inType := "query" // default
			if lowerIn == "path" {
				inType = "path"
			} else if lowerIn == "header" {
				inType = "header"
			}

			op.Parameters = append(op.Parameters, Parameter{
				Name:        param.Name,
				In:          inType,
				Required:    param.Required,
				Schema:      Schema{Type: b.mapType(param.Type)},
				Description: param.Description,
			})
		}
	}

	// 2. Process Response
	responseDesc := endpoint.Response.Description
	if responseDesc == "" {
		responseDesc = "Successful response"
	}

	respObj := Response{
		Description: responseDesc,
	}

	// Build properties for response schema
	if len(endpoint.Response.Fields) > 0 {
		schema := b.buildComplexSchema(endpoint.Response.Fields)
		respObj.Content = map[string]MediaType{
			"application/json": {
				Schema: schema,
			},
		}
	} else if endpoint.Response.Type != "void" {
		// Simple response type (String, Integer)
		respObj.Content = map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"type": b.mapType(endpoint.Response.Type),
				},
			},
		}
	}

	statusCode := "200"
	if endpoint.Response.StatusCode != 0 {
		statusCode = strings.TrimSpace(strings.Split(strings.Trim(string(endpoint.Response.StatusCode), "[]"), " ")[0]) // Just in case, but integer to string
		// Wait, StatusCode is int
		if endpoint.Response.StatusCode == 204 {
			statusCode = "204"
		} else {
			statusCode = "200" // Simplify for now
		}
	}

	op.Responses[statusCode] = respObj

	spec.Paths[fullPath][method] = op
}

// buildComplexSchema reconstructs the JSON schema from a flattened list of depth-aware ParamDefs
func (b *OpenAPIExporter) buildComplexSchema(fields []model.ParamDef) map[string]interface{} {
	// Root Schema (Object)
	rootProps := make(map[string]interface{})
	rootSchema := map[string]interface{}{
		"type":       "object",
		"properties": rootProps,
	}

	// pathMap tracks the Schema Object at each depth
	// pathMap[0] = rootSchema
	// pathMap[1] = a field at depth 1 (property of root)
	pathMap := make(map[int]map[string]interface{})
	pathMap[0] = rootSchema

	for _, field := range fields {
		if field.Depth < 1 {
			continue
		}

		// Find Parent Schema
		parentSchema, ok := pathMap[field.Depth-1]
		if !ok {
			// Fallback: attach to root if parent level missing (should not happen in valid DFS)
			parentSchema = rootSchema
		}

		// Determine where to attach this field (properties vs items)
		var targetProps map[string]interface{}

		parentType, _ := parentSchema["type"].(string)

		if parentType == "array" {
			// Parent is Array -> Properties belong to "items" (which must be an object)
			if _, hasItems := parentSchema["items"]; !hasItems {
				parentSchema["items"] = map[string]interface{}{
					"type":       "object",
					"properties": make(map[string]interface{}),
				}
			}
			itemsSchema := parentSchema["items"].(map[string]interface{})
			// Ensure items is object-like to accept properties
			if itemsSchema["type"] != "object" {
				itemsSchema["type"] = "object"
				itemsSchema["properties"] = make(map[string]interface{})
			}
			if _, hasProps := itemsSchema["properties"]; !hasProps {
				itemsSchema["properties"] = make(map[string]interface{})
			}
			targetProps = itemsSchema["properties"].(map[string]interface{})
		} else {
			// Parent is Object -> Attach to "properties"
			if _, hasProps := parentSchema["properties"]; !hasProps {
				parentSchema["properties"] = make(map[string]interface{})
			}
			targetProps = parentSchema["properties"].(map[string]interface{})
		}

		// Create Schema for Current Field
		fieldType := b.mapType(field.Type)
		fieldSchema := map[string]interface{}{
			"type": fieldType,
		}
		if field.Description != "" {
			fieldSchema["description"] = field.Description
		}

		// If Array, initialize default items (String)
		// This will be overwritten if children are added later
		if fieldType == "array" {
			fieldSchema["items"] = map[string]interface{}{
				"type": "string",
			}
		}

		// Attach
		targetProps[field.Name] = fieldSchema

		// Update Path Map for next depth
		pathMap[field.Depth] = fieldSchema
	}

	return rootSchema
}

// isCollection checks if type string implies a List/Array
func (b *OpenAPIExporter) isCollection(typeName string) bool {
	lower := strings.ToLower(typeName)
	return strings.Contains(lower, "list") || strings.Contains(lower, "set") || strings.Contains(lower, "[]") || strings.Contains(lower, "page")
}

// buildParamSchema builds a simple schema for a top-level parameter
func (b *OpenAPIExporter) buildParamSchema(param model.ParamDef) map[string]interface{} {
	return map[string]interface{}{
		"type":        b.mapType(param.Type),
		"description": param.Description,
	}
}

// mapType maps Java types to JSON Schema types
func (b *OpenAPIExporter) mapType(javaType string) string {
	lower := strings.ToLower(javaType)

	if strings.Contains(lower, "int") || strings.Contains(lower, "long") || strings.Contains(lower, "double") || strings.Contains(lower, "float") {
		return "integer" // Simplified
	}
	if strings.Contains(lower, "boolean") {
		return "boolean"
	}
	if strings.Contains(lower, "list") || strings.Contains(lower, "set") || strings.Contains(lower, "[]") {
		return "array"
	}
	if strings.Contains(lower, "map") || strings.Contains(lower, "dto") || strings.Contains(lower, "vo") || strings.Contains(lower, "entity") || strings.Contains(lower, "object") {
		return "object"
	}
	if strings.Contains(lower, "date") || strings.Contains(lower, "time") {
		return "string" // string format date
	}

	return "string"
}
