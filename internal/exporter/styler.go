package exporter

import (
	"github.com/xuri/excelize/v2"
)

// Styler handles Excel styling
type Styler struct {
	File *excelize.File

	// Pre-defined styles
	HeaderStyle     int
	ControllerStyle int
	ServiceStyle    int
	MapperStyle     int
	SQLStyle        int
	UtilStyle       int
	DefaultStyle    int
}

// NewStyler creates a new Styler and explicitly registers styles
func NewStyler(f *excelize.File) (*Styler, error) {
	s := &Styler{File: f}
	var err error

	// Header Style: Bold, Gray Background, Center Aligned
	s.HeaderStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#000000"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// Controller Style: Blue Text (Entry Point)
	s.ControllerStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#0000FF"}, // Blue
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// Service Style: Default Black
	s.ServiceStyle, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// Mapper Style: Default Black
	s.MapperStyle, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// SQL Style: Red Text (Database Interaction)
	s.SQLStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "#D32F2F"}, // Red
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// Util Style: Gray Italic (Helper)
	s.UtilStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "#757575", Italic: true}, // Gray
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	// Default Style
	s.DefaultStyle, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    createBorder(),
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func createBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "D4D4D4", Style: 1},
		{Type: "top", Color: "D4D4D4", Style: 1},
		{Type: "bottom", Color: "D4D4D4", Style: 1},
		{Type: "right", Color: "D4D4D4", Style: 1},
	}
}
