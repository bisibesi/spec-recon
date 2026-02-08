package analyzer

import (
	"spec-recon/internal/model"
)

// Analyzer is the main interface for analyzing Java/XML source code
type Analyzer interface {
	// Analyze scans the given root directory and returns:
	// - nodes: All parsed nodes (controllers, services, mappers, SQL queries, utils)
	// - summary: System-level statistics for the dashboard
	// - error: Any error that occurred during analysis
	Analyze(rootDir string) (nodes []*model.Node, summary *model.Summary, err error)
}

// Parser is responsible for parsing individual files
type Parser interface {
	// CanParse checks if this parser can handle the given file
	CanParse(filePath string) bool

	// Parse extracts nodes from the given file
	Parse(filePath string) ([]*model.Node, error)
}

// JavaParser parses Java source files (.java)
type JavaParser interface {
	Parser

	// ParseController parses a controller class and extracts endpoints
	ParseController(filePath string) ([]*model.Node, error)

	// ParseService parses a service class and extracts business logic methods
	ParseService(filePath string) ([]*model.Node, error)

	// ParseMapper parses a MyBatis mapper interface
	ParseMapper(filePath string) ([]*model.Node, error)
}

// XMLParser parses MyBatis XML mapper files
type XMLParser interface {
	Parser

	// ParseMapperXML parses a MyBatis XML file and extracts SQL queries
	ParseMapperXML(filePath string) ([]*model.Node, error)
}

// Linker is responsible for connecting nodes into call chains
type Linker interface {
	// Link establishes relationships between nodes based on heuristic matching
	// (Variable name/type matching for Ctrl->Svc->Mapper, namespace+id for Mapper->XML)
	Link(nodes []*model.Node) error
}

// Filter determines which files/classes should be excluded from analysis
type Filter interface {
	// ShouldExclude returns true if the file/class should be filtered out
	// Examples: *Util.java, *DTO.java, *VO.java, *Constant.java
	ShouldExclude(filePath string, className string) bool
}

// SummaryBuilder builds system-level statistics from parsed nodes
type SummaryBuilder interface {
	// BuildSummary generates a Summary from the list of nodes
	BuildSummary(nodes []*model.Node) (*model.Summary, error)
}

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
	// RootDir is the root directory to analyze
	RootDir string

	// ExcludePatterns are glob patterns for files/directories to exclude
	ExcludePatterns []string

	// IncludeUtils determines whether utility classes should be included
	IncludeUtils bool

	// EncodingHints provides encoding hints for charset detection
	// (e.g., "euc-kr", "ms949", "utf-8")
	EncodingHints []string
}

// DefaultConfig returns the default analyzer configuration
func DefaultConfig(rootDir string) *AnalyzerConfig {
	return &AnalyzerConfig{
		RootDir: rootDir,
		ExcludePatterns: []string{
			"**/test/**",
			"**/target/**",
			"**/build/**",
			"**/.git/**",
		},
		IncludeUtils: false, // Filter out utility classes by default
		EncodingHints: []string{
			"utf-8",
			"euc-kr",
			"ms949",
		},
	}
}
