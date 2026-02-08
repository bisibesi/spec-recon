package model

// EndpointDef represents a REST API endpoint definition optimized for documentation
type EndpointDef struct {
	// HTTP Method (GET, POST, PUT, DELETE, etc.)
	Method string

	// Full URL path (e.g., "/api/v1/users")
	Path string

	// Controller class name
	ControllerName string

	// Method name in the controller
	MethodName string

	// Summary from JavaDoc or annotation
	Summary string

	// Detailed description from comments
	Description string

	// Request parameters
	Params []ParamDef

	// Response definition
	Response ResponseDef
}

// ParamDef represents a parameter in the API request
type ParamDef struct {
	// Parameter name
	Name string

	// Parameter type (String, Integer, User, etc.)
	Type string

	// Where the parameter comes from (Query, Body, Path, Header)
	In string

	// Whether the parameter is required
	Required bool

	// Parameter description
	Description string

	// Nesting depth (0=Root, 1=Child, 2=Grandchild, etc.)
	// Used for indentation in documentation output
	Depth int

	// Nested fields (for complex types like DTOs)
	// If this parameter is a DTO, Fields contains its properties
	Fields []ParamDef
}

// ResponseDef represents the API response
type ResponseDef struct {
	// Response type (String, User, List<Product>, etc.)
	Type string

	// Response description
	Description string

	// HTTP status code (optional)
	StatusCode int

	// Nested fields (for complex response types like DTOs)
	// If the response is a DTO, Fields contains its properties
	Fields []ParamDef
}

// NewEndpointDef creates a new endpoint definition
func NewEndpointDef() *EndpointDef {
	return &EndpointDef{
		Params: make([]ParamDef, 0),
	}
}
