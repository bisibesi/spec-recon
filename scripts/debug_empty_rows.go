package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	// Open the Excel file
	f, err := excelize.OpenFile("output/e2e_report.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Check "Spec Detail" sheet
	sheetName := "Spec Detail"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== DEBUGGING EMPTY ROWS ===")

	// Check specific rows that were reported as empty
	emptyRows := []int{2, 17, 23, 36, 42, 50, 66, 74}

	for _, rowNum := range emptyRows {
		if rowNum-1 < len(rows) {
			row := rows[rowNum-1]
			fmt.Printf("\nRow %d:\n", rowNum)
			for i, cell := range row {
				if strings.TrimSpace(cell) != "" {
					fmt.Printf("  Col %d: '%s'\n", i, cell)
				}
			}
		}
	}
}
