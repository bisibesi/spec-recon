package exporter

import (
	"strings"

	"spec-recon/internal/exporter/html"
	"spec-recon/internal/exporter/openapi"
	"spec-recon/internal/exporter/word"
)

// GetExporters returns a list of Exporters based on requested formats
func GetExporters(formats []string) []Exporter {
	exporters := []Exporter{}
	seen := make(map[string]bool)

	for _, fmtStr := range formats {
		fmtStr = strings.ToLower(strings.TrimSpace(fmtStr))
		if seen[fmtStr] {
			continue
		}
		seen[fmtStr] = true

		switch fmtStr {
		case "excel", "xlsx":
			exporters = append(exporters, NewExcelExporter())
		case "html":
			exporters = append(exporters, html.NewHTMLExporter())
		case "word", "docx":
			exporters = append(exporters, word.NewWordExporter())
		case "openapi", "swagger", "json":
			exporters = append(exporters, openapi.NewOpenAPIExporter())
		}
	}

	// Default to Excel if nothing valid specified?
	// Or maybe the caller handles defaults.
	// We'll leave it empty if no match.

	return exporters
}
