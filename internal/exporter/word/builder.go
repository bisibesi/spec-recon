package word

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/model"

	"github.com/nguyenthenguyen/docx"
)

//go:embed template.docx
var templateFS embed.FS

type WordExporter struct{}

func NewWordExporter() *WordExporter {
	return &WordExporter{}
}

func (e *WordExporter) Export(summary *model.Summary, tree []*model.Node, cfg *config.Config) error {
	// 1. Extract embedded template to temp file
	templateBytes, err := templateFS.ReadFile("template.docx")
	if err != nil {
		return fmt.Errorf("failed to read embedded template: %w", err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "spec-recon-template-*.docx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up

	if _, err := tmpFile.Write(templateBytes); err != nil {
		return fmt.Errorf("failed to write template to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Open docx from temp path
	r, err := docx.ReadDocxFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read docx from temp file: %w", err)
	}
	defer r.Close()

	doc := r.Editable()

	// 1. Extract API Endpoints (Swagger-style)
	// Use ClassMap and FieldTypeMap from summary for deep schema extraction
	classMap := summary.ClassMap
	fieldTypeMap := summary.FieldTypeMap
	if classMap == nil {
		classMap = make(map[string]*model.Node)
	}
	if fieldTypeMap == nil {
		fieldTypeMap = make(map[string]map[string]string)
	}

	endpoints := analyzer.ExtractEndpoints(tree, classMap, fieldTypeMap)

	// Sort endpoints by path
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Path < endpoints[j].Path
	})

	// 2. Calculate stats from VISIBLE endpoints (not global counts)
	// This ensures consistency between overview and content
	totalEndpoints := len(endpoints)
	controllerSet := make(map[string]bool)
	for _, ep := range endpoints {
		controllerSet[ep.ControllerName] = true
	}
	totalControllers := len(controllerSet)

	// 3. Replace Summary Placeholders
	doc.Replace("{{Date}}", summary.AnalysisDate, -1)
	doc.Replace("{{TotalEndpoints}}", fmt.Sprintf("%d", totalEndpoints), -1)
	doc.Replace("{{TotalControllers}}", fmt.Sprintf("%d", totalControllers), -1)

	// 4. Generate API Documentation Content as Plain Text
	// The docx library will handle the XML encoding
	var contentBuilder strings.Builder

	contentBuilder.WriteString("API SPECIFICATION\n\n")
	contentBuilder.WriteString("Summary Overview:\n")
	contentBuilder.WriteString(fmt.Sprintf("  • Total Endpoints: %d\n", totalEndpoints))
	contentBuilder.WriteString(fmt.Sprintf("  • Controllers: %d\n\n", totalControllers))
	contentBuilder.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Generate documentation for each endpoint
	for i, endpoint := range endpoints {
		buildEndpointText(&contentBuilder, &endpoint)

		// Add separator between endpoints
		if i < len(endpoints)-1 {
			contentBuilder.WriteString("\n" + strings.Repeat("-", 80) + "\n\n")
		}
	}

	// Inject content (the library handles XML encoding)
	doc.Replace("{{Content}}", contentBuilder.String(), -1)

	outFile := strings.TrimSuffix(cfg.GetOutputPath(), ".xlsx") + ".docx"
	if err := doc.WriteToFile(outFile); err != nil {
		return fmt.Errorf("failed to write Word document: %w", err)
	}

	return nil
}

// buildEndpointText builds plain text documentation for a single API endpoint
func buildEndpointText(sb *strings.Builder, endpoint *model.EndpointDef) {
	// Endpoint Header
	sb.WriteString(fmt.Sprintf("[%s] %s\n", endpoint.Method, endpoint.Path))
	sb.WriteString(fmt.Sprintf("Controller: %s.%s\n", endpoint.ControllerName, endpoint.MethodName))

	if endpoint.Summary != "" {
		sb.WriteString(fmt.Sprintf("Summary: %s\n", endpoint.Summary))
	}
	sb.WriteString("\n")

	// Request Parameters
	if len(endpoint.Params) > 0 {
		sb.WriteString("REQUEST PARAMETERS:\n")
		sb.WriteString(fmt.Sprintf("%-25s %-20s %-10s %-10s %s\n", "Name", "Type", "In", "Required", "Description"))
		sb.WriteString(strings.Repeat("-", 100) + "\n")

		for _, param := range endpoint.Params {
			required := "No"
			if param.Required {
				required = "Yes"
			}

			// Calculate indentation based on depth
			indent := strings.Repeat("  ", param.Depth)
			marker := ""
			if param.Depth > 0 {
				marker = "└ "
			}
			paramName := indent + marker + param.Name

			// Write parameter row
			sb.WriteString(fmt.Sprintf("%-25s %-20s %-10s %-10s %s\n",
				truncate(paramName, 25),
				truncate(param.Type, 20),
				truncate(param.In, 10),
				required,
				param.Description))

			// Nested fields are now flattened in the Fields slice with proper Depth
			// No need for separate child iteration - they're already in the main list
			if len(param.Fields) > 0 {
				for _, field := range param.Fields {
					// Calculate indentation for nested field
					fieldIndent := strings.Repeat("  ", field.Depth)
					fieldMarker := ""
					if field.Depth > 0 {
						fieldMarker = "└ "
					}
					fieldName := fieldIndent + fieldMarker + field.Name

					sb.WriteString(fmt.Sprintf("%-25s %-20s %-10s %-10s %s\n",
						truncate(fieldName, 25),
						truncate(field.Type, 20),
						"-", // No "In" for nested fields
						"-", // No "Required" for nested fields
						field.Description))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Response
	sb.WriteString("RESPONSE:\n")
	sb.WriteString(fmt.Sprintf("%-15s %-25s %s\n", "Status Code", "Type", "Description"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("%-15d %-25s %s\n",
		endpoint.Response.StatusCode,
		truncate(endpoint.Response.Type, 25),
		endpoint.Response.Description))

	// Response nested fields (DTO schema)
	if len(endpoint.Response.Fields) > 0 {
		sb.WriteString("\nResponse Fields:\n")
		sb.WriteString(fmt.Sprintf("%-30s %-20s %s\n", "Field Name", "Type", "Description"))
		sb.WriteString(strings.Repeat("-", 80) + "\n")

		for _, field := range endpoint.Response.Fields {
			// Calculate indentation based on depth
			indent := strings.Repeat("  ", field.Depth)
			marker := ""
			if field.Depth > 0 {
				marker = "└ "
			}
			fieldName := indent + marker + field.Name

			sb.WriteString(fmt.Sprintf("%-30s %-20s %s\n",
				truncate(fieldName, 30),
				truncate(field.Type, 20),
				field.Description))
		}
	}

	sb.WriteString("\n")
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
