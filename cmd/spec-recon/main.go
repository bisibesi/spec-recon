package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spec-recon/internal/analyzer"
	"spec-recon/internal/config"
	"spec-recon/internal/exporter"
	"spec-recon/internal/javaparser"
	"spec-recon/internal/linker"
	"spec-recon/internal/logger"
	"spec-recon/internal/model"
	"spec-recon/internal/ui"
	"spec-recon/internal/xmlparser"
)

const (
	appName    = "Spec Recon"
	appVersion = "1.0.0"
	appDesc    = "A Pure Go static analysis tool for Legacy Spring (Java/XML) codebases"
)

var (
	configPath  string
	verbose     bool
	showVersion bool
	outputDir   string
	formats     string
)

func init() {
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.StringVar(&configPath, "c", "config.yaml", "Path to configuration file (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging (DEBUG level)")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.StringVar(&outputDir, "output", "", "Override output directory from config")
	flag.StringVar(&formats, "format", "excel,html,word,json", "Comma-separated output formats (excel,html,word,json)")
}

func main() {
	// CRITICAL: Ensure "Press Enter to Exit" runs even on panic or error
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n❌ PANIC: %v\n", r)
		}
		waitForEnter()
	}()

	// Run the actual application logic
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s\n%s\n", appName, appVersion, appDesc)
		return 0
	}

	printBanner()

	// 1. Initialize
	logger.Info("Loading configuration...")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		return 1
	}

	if outputDir != "" {
		cfg.Output.Dir = outputDir
		cfg.EnsureOutputDir()
	}

	logPath := filepath.Join(cfg.Output.Dir, "spec_recon.log")
	if err := logger.Init(os.Stdout, logPath, verbose); err != nil {
		fmt.Printf("❌ Failed to initialize logger: %v\n", err)
		return 1
	}
	defer logger.Close()

	if err := runAnalysis(cfg); err != nil {
		logger.Error("Analysis failed: %v", err)
		return 1
	}

	logger.Info("✅ Analysis Complete. Check [%s] directory.", cfg.Output.Dir)
	return 0
}

// waitForEnter pauses execution and waits for user to press Enter
// This prevents the console window from closing immediately when double-clicked
func waitForEnter() {
	fmt.Println("\n==========================================")
	fmt.Println("Execution Finished. Press 'Enter' to exit.")
	fmt.Println("==========================================")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func runAnalysis(cfg *config.Config) error {
	pipeline := ui.NewPipeline([]ui.Phase{
		ui.PhaseScanning,
		ui.PhaseLinking, // Scanning + Parsing + Linking combined conceptually for user
		ui.PhaseGenerating,
	})

	// --- Phase 1: Scanning & Parsing ---
	logger.Info("Phase 1: Scanning & Parsing...")
	scanBar := pipeline.NextPhase(100)

	// Scan files
	files, err := analyzer.ScanDirectory(cfg.Project.RootDir, cfg.Analysis.ExcludeDirs)
	if err != nil {
		return err
	}
	scanBar.SetTotal(len(files))

	pool := linker.NewComponentPool()

	for _, path := range files {
		content, err := analyzer.ReadFile(path)
		if err != nil {
			logger.Warn("Failed to read file %s: %v", path, err)
			scanBar.Increment()
			continue
		}

		if strings.HasSuffix(path, ".java") {
			cls, err := javaparser.ParseJavaFile(content)
			if err == nil {
				pool.AddJavaClass(cls, content)
			}
		} else if strings.HasSuffix(path, ".xml") {
			mapper, err := xmlparser.ParseXMLFile(content)
			if err == nil {
				pool.AddMapperXML(mapper)
			}
		}
		scanBar.Increment()
	}
	scanBar.Finish()

	// --- Phase 2: Linking ---
	logger.Info("Phase 2: Linking Components...")
	linkBar := pipeline.NextPhase(50) // Arbitrary steps for linking

	mainLinker := linker.NewLinker(pool)
	tree := mainLinker.BuildCallGraph()
	linkBar.Finish()

	// Build Summary
	summary := buildSummary(pool, tree)

	// Extract API Endpoints (for HTML/Word/JSON reports)
	endpoints := analyzer.ExtractEndpoints(tree, summary.ClassMap, summary.FieldTypeMap)
	logger.Info("Extracted %d API endpoints", len(endpoints))

	// --- Phase 3: Reporting ---
	logger.Info("Phase 3: Generating Reports...")
	targetFormats := strings.Split(formats, ",")
	exporters := exporter.GetExporters(targetFormats)

	genBar := pipeline.NextPhase(len(exporters))

	var exportErrors []error
	for _, exp := range exporters {
		if err := exp.Export(summary, tree, cfg); err != nil {
			logger.Error("Export failed: %v", err)
			exportErrors = append(exportErrors, err)
		}
		genBar.Increment()
	}
	genBar.Finish()

	pipeline.Finish()

	// Return error if any exports failed
	if len(exportErrors) > 0 {
		return fmt.Errorf("one or more exports failed: %d errors", len(exportErrors))
	}

	return nil
}

func buildSummary(pool *linker.ComponentPool, tree []*model.Node) *model.Summary {
	s := model.NewSummary()
	s.AnalysisDate = time.Now().Format("2006-01-02")

	// Count from Pool (Source of Truth for Totals)
	for _, node := range pool.ClassMap {
		// FILTER 1: Strict Model Class Filter (DTOs, VOs, Entities)
		if model.IsModelClass(node.ID) {
			continue
		}

		// FILTER 2: Empty Class Filter (Skip files with no methods)
		// This ensures stats match the "Spec Detail" visual report
		if len(node.Children) == 0 {
			continue
		}

		switch node.Type {
		case model.NodeTypeController:
			s.TotalControllers++
		case model.NodeTypeService:
			s.TotalServices++
		case model.NodeTypeMapper:
			s.TotalMappers++
		case model.NodeTypeUtil:
			s.TotalUtils++
		}
	}
	s.TotalSQLs = len(pool.SQLMap)

	// CRITICAL: Wire ClassMap and FieldTypeMap for deep schema extraction
	// This enables HTML/Word exporters to resolve nested DTO fields
	s.ClassMap = pool.ClassMap
	s.FieldTypeMap = pool.FieldTypeMap

	// Log schema extraction capabilities
	logger.Info("Schema Pool: %d classes, %d field mappings", len(s.ClassMap), len(s.FieldTypeMap))

	return s
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                      SPEC RECON v1.0.0                    ║
║        Static Analysis for Legacy Spring Projects         ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
