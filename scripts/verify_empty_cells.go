package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	// Check which file to verify
	filename := "output/e2e_report.xlsx"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	// Open the Excel file
	f, err := excelize.OpenFile(filename)
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

	fmt.Printf("=== ZERO TOLERANCE CHECK: %s ===\n", filename)
	fmt.Printf("Checking sheet: %s\n", sheetName)
	fmt.Printf("Total rows: %d\n\n", len(rows))

	emptyCount := 0
	checkedCount := 0

	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}

		// Check if this row has a Type in Column A (index 0)
		// If yes, it MUST have a Method/ID in Column C (index 2)
		if len(row) > 0 {
			typeCell := strings.TrimSpace(row[0])

			// If there's a type marker (e.g., [SERVICE], [MAPPER], etc.)
			if typeCell != "" && strings.HasPrefix(typeCell, "[") {
				checkedCount++

				// Check Column C (Method/ID) - index 2
				methodCell := ""
				if len(row) > 2 {
					methodCell = strings.TrimSpace(row[2])
				}

				if methodCell == "" {
					fmt.Printf("❌ EMPTY METHOD at row %d: Type=%s, Method=EMPTY\n", i+1, typeCell)
					emptyCount++
				}
			}
		}
	}

	fmt.Printf("\nChecked %d rows with type markers\n", checkedCount)

	if emptyCount > 0 {
		fmt.Printf("❌ FAILED: Found %d empty Method cells!\n", emptyCount)
		os.Exit(1)
	} else {
		fmt.Printf("✅ PASSED: No empty Method cells found!\n")
		os.Exit(0)
	}
}
