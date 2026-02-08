package model

import (
	"fmt"
	"strings"
)

// NodeType represents the layer type in the call chain
type NodeType string

const (
	NodeTypeController NodeType = "CONTROLLER"
	NodeTypeService    NodeType = "SERVICE"
	NodeTypeMapper     NodeType = "MAPPER"
	NodeTypeSQL        NodeType = "SQL"
	NodeTypeUtil       NodeType = "UTIL"
)

// Node represents a unified code element (Controller, Service, Mapper, SQL, or Util)
type Node struct {
	// Identity
	ID   string   // Unique identifier: "package.ClassName.methodName"
	Type NodeType // Node type (CONTROLLER, SERVICE, MAPPER, SQL, UTIL)

	// Location
	Package string // Java package name (e.g., "com.company.legacy")
	File    string // File path relative to source root
	Line    int    // Line number where method/query is defined

	// Method/Query Details
	Method       string // Method name or SQL query ID
	Params       string // Input parameters (formatted string)
	ReturnDetail string // Return type or result description
	Body         string // Method body content (for analysis)
	Comment      string // JavaDoc summary or query description

	// Linking (for building call chains)
	Children []*Node // Direct downstream nodes
	Parent   *Node   // Direct upstream node

	// Metadata
	Annotation string // Primary annotation (@Controller, @Service, etc.)
	URL        string // Request mapping URL (for controllers only)
}

// NewNode creates a new Node with the given type
func NewNode(nodeType NodeType) *Node {
	return &Node{
		Type:     nodeType,
		Children: make([]*Node, 0),
	}
}

// AddChild adds a child node to the current node
func (n *Node) AddChild(child *Node) {
	if child == nil {
		return
	}

	// GATEKEEPER 1: Reject nodes with empty or whitespace-only names
	// This prevents empty rows from appearing in Excel output
	trimmedName := strings.TrimSpace(child.Method)
	if len(trimmedName) == 0 {
		// Log and skip - do not add this child
		fmt.Printf("[LINKER SKIP] Empty node name detected (Type: %s, Parent: %s)\n", child.Type, n.Method)
		return
	}

	n.Children = append(n.Children, child)
	child.Parent = n
}

// IsController checks if this node is a controller
func (n *Node) IsController() bool {
	return n.Type == NodeTypeController
}

// IsService checks if this node is a service
func (n *Node) IsService() bool {
	return n.Type == NodeTypeService
}

// IsMapper checks if this node is a mapper
func (n *Node) IsMapper() bool {
	return n.Type == NodeTypeMapper
}

// IsSQL checks if this node is a SQL query
func (n *Node) IsSQL() bool {
	return n.Type == NodeTypeSQL
}

// IsUtil checks if this node is a utility
func (n *Node) IsUtil() bool {
	return n.Type == NodeTypeUtil
}

// String returns a human-readable representation of the node
func (n *Node) String() string {
	return fmt.Sprintf("[%s] %s.%s", n.Type, n.Package, n.Method)
}

// Summary represents the system-level statistics for the Overview sheet
type Summary struct {
	// System Scale
	TotalControllers int
	TotalServices    int
	TotalMappers     int
	TotalSQLs        int
	TotalUtils       int
	AnalysisDate     string

	// Legacy/Detailed Stats
	ControllerStats []ControllerStat

	// Pool Data for Deep Schema Extraction (API Documentation)
	ClassMap     map[string]*Node
	FieldTypeMap map[string]map[string]string
}

// ControllerStat represents statistics for a single controller
type ControllerStat struct {
	Name        string // Controller class name
	Package     string // Package name
	BaseURL     string // Base request mapping URL
	MethodCount int    // Total number of methods in this controller
	ApiCount    int    // Number of @RestController methods (returns JSON/data)
	ViewCount   int    // Number of @Controller methods (returns views)
}

// NewSummary creates a new Summary instance
func NewSummary() *Summary {
	return &Summary{
		ControllerStats: make([]ControllerStat, 0),
	}
}

// AddControllerStat adds a controller statistic to the summary
func (s *Summary) AddControllerStat(stat ControllerStat) {
	s.ControllerStats = append(s.ControllerStats, stat)
}

// IsModelClass checks if the node represents a DTO, VO, or Entity based on its ID/Name
func IsModelClass(id string) bool {
	// Extract class name from ID (package.ClassName.methodName)
	parts := strings.Split(id, ".")
	if len(parts) == 0 {
		return false
	}

	var candidates []string
	if len(parts) >= 2 {
		candidates = append(candidates, parts[len(parts)-2]) // ClassName
		candidates = append(candidates, parts[len(parts)-1]) // MethodName
	} else {
		candidates = append(candidates, parts[0])
	}

	suffixes := []string{"dto", "vo", "entity", "request", "response", "projection", "exception"}

	for _, name := range candidates {
		lower := strings.ToLower(name)
		for _, suffix := range suffixes {
			if strings.HasSuffix(lower, suffix) {
				return true
			}
		}
	}
	return false
}
