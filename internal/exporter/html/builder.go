package html

import (
	"html/template"
	"os"
	"sort"
	"strings"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/model"
)

type HTMLExporter struct{}

func NewHTMLExporter() *HTMLExporter {
	return &HTMLExporter{}
}

// Data structures for API Documentation Template
type APIReportData struct {
	AnalysisDate     string
	TotalEndpoints   int
	TotalControllers int
	Endpoints        []model.EndpointDef
}

func (e *HTMLExporter) Export(summary *model.Summary, tree []*model.Node, cfg *config.Config) error {
	// Extract API endpoints (Swagger-style)
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

	// Calculate stats from VISIBLE endpoints (not global counts)
	// This ensures consistency between overview and content
	totalEndpoints := len(endpoints)
	controllerSet := make(map[string]bool)
	for _, ep := range endpoints {
		controllerSet[ep.ControllerName] = true
	}
	totalControllers := len(controllerSet)

	// Prepare Data with calculated stats
	data := APIReportData{
		AnalysisDate:     summary.AnalysisDate,
		TotalEndpoints:   totalEndpoints,
		TotalControllers: totalControllers,
		Endpoints:        endpoints,
	}

	// Create Output
	outputFile := strings.TrimSuffix(cfg.GetOutputPath(), ".xlsx") + ".html"
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Use API documentation template
	tmpl, err := template.New("api-report").Funcs(template.FuncMap{
		"methodColor": getMethodColor,
		"methodBadge": getMethodBadge,
		"mul": func(a, b int) int {
			return a * b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}).Parse(APIReportTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(f, data)
}

// getMethodColor returns CSS color class for HTTP method
func getMethodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "method-get"
	case "POST":
		return "method-post"
	case "PUT":
		return "method-put"
	case "DELETE":
		return "method-delete"
	case "PATCH":
		return "method-patch"
	default:
		return "method-default"
	}
}

// getMethodBadge returns badge text for HTTP method
func getMethodBadge(method string) string {
	return strings.ToUpper(method)
}
