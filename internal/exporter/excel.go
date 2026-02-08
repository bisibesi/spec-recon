package exporter

import (
	"fmt"
	"sort"
	"strings"

	"spec-recon/internal/config"
	"spec-recon/internal/exporter/common"
	"spec-recon/internal/model"
	"spec-recon/internal/utils"

	"github.com/xuri/excelize/v2"
)

// ExcelExporter handles the Excel generation
type ExcelExporter struct {
	// Stateless
}

// NewExcelExporter creates a new ExcelExporter
func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{}
}

// Export generates the Excel report
func (e *ExcelExporter) Export(summary *model.Summary, tree []*model.Node, cfg *config.Config) error {
	outputFile := cfg.GetOutputPath()
	f := excelize.NewFile()
	styler, err := NewStyler(f)
	if err != nil {
		return err
	}

	// 1. Create Overview Sheet
	if err := e.writeOverview(f, styler, summary, tree); err != nil {
		return err
	}

	// 2. Create Spec Detail Sheet
	if err := e.writeSpecDetail(f, styler, tree); err != nil {
		return err
	}

	// Remove default "Sheet1"
	if idx, err := f.GetSheetIndex("Sheet1"); err == nil && idx != -1 {
		f.DeleteSheet("Sheet1")
	}

	// Save
	if err := f.SaveAs(outputFile); err != nil {
		return err
	}

	return nil
}

// --- Overview Sheet Logic ---

func (e *ExcelExporter) writeOverview(f *excelize.File, s *Styler, summary *model.Summary, controllers []*model.Node) error {
	sheet := "Overview"
	f.NewSheet(sheet)

	// Section A: System Summary
	headers := []string{"Metric", "Count"}

	row := 1
	e.writeRow(f, sheet, row, headers, s.HeaderStyle)
	row++

	metrics := []struct {
		Key string
		Val int
	}{
		{"Total Controllers", summary.TotalControllers},
		{"Total Services", summary.TotalServices},
		{"Total Mappers", summary.TotalMappers},
		{"Total SQL Queries", summary.TotalSQLs},
		{"Total Utils", summary.TotalUtils},
	}

	for _, m := range metrics {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), m.Key)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.Val)
		row++
	}

	row += 2 // Spacer

	// Section B: Controller Complexity
	headersB := []string{"No", "Controller Name", "Total Methods", "API Count", "View Count", "Note"}
	e.writeRow(f, sheet, row, headersB, s.HeaderStyle)
	row++

	// Sort controllers by complexity (method count)
	sort.Slice(controllers, func(i, j int) bool {
		return len(controllers[i].Children) > len(controllers[j].Children)
	})

	listIndex := 1
	for _, ctrl := range controllers {
		// FILTER 1: Strict Model Class Filter (DTOs, VOs, Entities)
		if model.IsModelClass(ctrl.ID) {
			continue
		}

		// FILTER 2: Empty Node Filter
		// If these are "Controllers" (or any root node), they must have methods to be listed.
		if len(ctrl.Children) == 0 {
			continue
		}

		// Calculate stats
		methodCount := len(ctrl.Children) // Direct children are methods
		apiCount := 0
		viewCount := 0
		for _, child := range ctrl.Children {
			ret := strings.ToLower(child.ReturnDetail)
			if strings.Contains(ret, "responseentity") || strings.Contains(ret, "dto") || strings.Contains(ret, "json") {
				apiCount++
			} else {
				viewCount++
			}
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), listIndex)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), getSimpleClassName(ctrl.ID))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), methodCount)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), apiCount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), viewCount)

		if methodCount > 20 {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "Complex")
		}

		row++
		listIndex++
	}

	// Adjust column widths
	f.SetColWidth(sheet, "B", "C", 30)

	return nil
}

// --- Spec Detail Sheet Logic ---

func (e *ExcelExporter) writeSpecDetail(f *excelize.File, s *Styler, controllers []*model.Node) error {
	sheet := "Spec Detail"
	f.NewSheet(sheet)

	headers := []string{"Type", "Package/File", "Method/ID", "URL", "Params (Input)", "Return/Detail (Output)", "Comment"}
	e.writeRow(f, sheet, 1, headers, s.HeaderStyle)

	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	row := 2
	sort.Slice(controllers, func(i, j int) bool {
		return controllers[i].ID < controllers[j].ID
	})

	for _, ctrl := range controllers {
		// Use Shared Sorter Logic
		mainStream, utilStream := common.SortNodes(ctrl)

		// Pre-filter stream inputs to identify valid content
		var validMain []*model.Node
		for _, node := range mainStream {
			if isExportable(node) {
				validMain = append(validMain, node)
			}
		}

		var validUtil []*model.Node
		for _, node := range utilStream {
			if isExportable(node) {
				validUtil = append(validUtil, node)
			}
		}

		// SKIP CONTROLLER: If it has no valid children (empty shell)
		// We do not write the blue header row if there are no methods to list.
		if len(validMain) == 0 && len(validUtil) == 0 {
			continue
		}

		// 1. Write Controller Node (Root)
		e.writeControllerRow(f, sheet, row, ctrl, s)
		row++

		// 3. Write Main Stream (Business Logic)
		for _, node := range validMain {
			e.writeNodeRow(f, sheet, row, node, s)
			row++
		}

		// 4. Separator & Util Stream
		if len(validUtil) > 0 {
			// Separator Row (Only if main stream had content, optional but cleaner)
			if len(validMain) > 0 {
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), s.DefaultStyle)
				row++
			}

			for _, node := range validUtil {
				e.writeNodeRow(f, sheet, row, node, s)
				row++
			}
		}

		// 5. End Spacer (Removed for compact view)
		// row++
	}

	// Auto width
	f.SetColWidth(sheet, "B", "B", 40) // Package
	f.SetColWidth(sheet, "C", "C", 40) // Method
	f.SetColWidth(sheet, "D", "D", 40) // URL
	f.SetColWidth(sheet, "E", "F", 30) // Params/Return
	f.SetColWidth(sheet, "G", "G", 50) // Comment

	return nil
}

func (e *ExcelExporter) writeControllerRow(f *excelize.File, sheet string, row int, node *model.Node, s *Styler) {
	typeLabel := fmt.Sprintf("[%s]", node.Type)

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), typeLabel)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), node.Package)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), node.Method)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), node.URL)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), node.Params)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), node.ReturnDetail)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), node.Comment)

	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), s.ControllerStyle)
}

func (e *ExcelExporter) writeNodeRow(f *excelize.File, sheet string, row int, node *model.Node, s *Styler) {
	typeLabel := fmt.Sprintf("[%s]", node.Type)

	style := s.DefaultStyle
	switch node.Type {
	case model.NodeTypeService:
		style = s.ServiceStyle
	case model.NodeTypeMapper:
		style = s.MapperStyle
	case model.NodeTypeSQL:
		style = s.SQLStyle
	case model.NodeTypeUtil:
		style = s.UtilStyle
	}

	// Column A: Type (keep the tag)
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), typeLabel)

	// Column B: Package/File - CLEAN OUTPUT (no indentation prefixes)
	// Use Package field if available, otherwise use File
	packageOrFile := node.Package
	if packageOrFile == "" {
		packageOrFile = node.File
	}
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), packageOrFile)

	// Column C: Method/ID - CLEAN OUTPUT (no indentation prefixes)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), node.Method)

	// Column D: URL
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), node.URL)

	// Column E: Params
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), node.Params)

	// Column F: Return Detail
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), node.ReturnDetail)

	// Column G: Comment
	comment := node.Comment
	if node.Type == model.NodeTypeUtil && strings.TrimSpace(comment) == "" {
		comment = "[Ref] Used in this flow"
	}
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), comment)

	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("G%d", row), style)
}

func (e *ExcelExporter) writeRow(f *excelize.File, sheet string, row int, values []string, style int) {
	for i, val := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, val)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

// shouldSkipEmptyRow determines if a node should be skipped in Excel output
// KEEP: CONTROLLER headers (section markers)
// SKIP: SERVICE, MAPPER, SQL, UTIL with empty/whitespace-only Method names
func shouldSkipEmptyRow(node *model.Node) bool {
	// Always keep CONTROLLER headers (they're section markers)
	if node.Type == model.NodeTypeController {
		return false
	}

	// For all other types, skip if Method is empty/whitespace
	methodName := strings.TrimSpace(node.Method)
	return methodName == ""
}

func getSimpleClassName(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}

// isExportable determines if a node should be included in the report
func isExportable(node *model.Node) bool {
	// 1. Empty Name check
	if len(strings.TrimSpace(node.Method)) == 0 {
		return false
	}

	// 2. Local Excel logic check
	if shouldSkipEmptyRow(node) {
		return false
	}

	// 3. Noise check (keywords, etc)
	if utils.IsNoise(node.Method) {
		return false
	}

	// 4. Model Class check (DTOs/VOs) using Shared Model Logic
	if model.IsModelClass(node.ID) {
		return false
	}

	// 5. Empty Util check
	if node.Type == model.NodeTypeUtil && len(node.Children) == 0 {
		return false
	}

	return true
}
