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
	filename := "output/spec-recon-report.xlsx"
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

	// Noise keywords to check for
	noiseKeywords := []string{"if", "else", "new", "throw", "try", "catch", "switch", "for", "while", "return"}
	systemTypes := []string{"ModelAndView", "model", "void"}

	fmt.Printf("=== EXCEL NOISE CHECK: %s ===\n", filename)
	fmt.Printf("Checking sheet: %s\n", sheetName)
	fmt.Printf("Total rows: %d\n\n", len(rows))

	foundNoise := false
	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}

		// Check column C (Method/ID) - index 2
		if len(row) > 2 {
			methodName := strings.TrimSpace(row[2])
			if methodName == "" {
				continue
			}

			// Check if it's noise keyword
			for _, noise := range noiseKeywords {
				if strings.EqualFold(methodName, noise) {
					fmt.Printf("❌ NOISE FOUND at row %d: '%s' (keyword)\n", i+1, methodName)
					foundNoise = true
				}
			}

			// Check if it's system type
			for _, sysType := range systemTypes {
				if strings.EqualFold(methodName, sysType) {
					fmt.Printf("❌ NOISE FOUND at row %d: '%s' (system type)\n", i+1, methodName)
					foundNoise = true
				}
			}

			// Check if it ends with Exception
			if strings.HasSuffix(methodName, "Exception") {
				fmt.Printf("❌ NOISE FOUND at row %d: '%s' (Exception type)\n", i+1, methodName)
				foundNoise = true
			}
		}
	}

	fmt.Println()
	if !foundNoise {
		fmt.Println("✅ Excel Noise Removed: YES")
		fmt.Println("No noise keywords (if, new, Exception, ModelAndView, etc.) found!")
	} else {
		fmt.Println("❌ Excel Noise Removed: NO")
		fmt.Println("Found noise keywords that should have been filtered!")
	}
}
