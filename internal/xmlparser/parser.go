package xmlparser

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// SQL represents a single SQL statement in MyBatis mapper
type SQL struct {
	ID      string // SQL statement ID (e.g., "selectUserCount")
	Type    string // Type: select, insert, update, delete
	Content string // SQL query content
}

// MapperXML represents a parsed MyBatis XML mapper file
type MapperXML struct {
	Namespace string // Mapper namespace (matches Java interface)
	SQLs      []SQL  // List of SQL statements
}

// RawMapper is used for XML unmarshaling
type RawMapper struct {
	XMLName   xml.Name `xml:"mapper"`
	Namespace string   `xml:"namespace,attr"`
	Selects   []RawSQL `xml:"select"`
	Inserts   []RawSQL `xml:"insert"`
	Updates   []RawSQL `xml:"update"`
	Deletes   []RawSQL `xml:"delete"`
}

// RawSQL represents a SQL element in XML
type RawSQL struct {
	ID      string `xml:"id,attr"`
	Content string `xml:",chardata"`
}

// ParseXMLFile parses a MyBatis XML mapper file
func ParseXMLFile(content string) (*MapperXML, error) {
	// Parse XML
	var rawMapper RawMapper
	err := xml.Unmarshal([]byte(content), &rawMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	mapper := &MapperXML{
		Namespace: rawMapper.Namespace,
		SQLs:      []SQL{},
	}

	// Extract SELECT statements
	for _, sel := range rawMapper.Selects {
		mapper.SQLs = append(mapper.SQLs, SQL{
			ID:      sel.ID,
			Type:    "select",
			Content: cleanSQLContent(sel.Content),
		})
	}

	// Extract INSERT statements
	for _, ins := range rawMapper.Inserts {
		mapper.SQLs = append(mapper.SQLs, SQL{
			ID:      ins.ID,
			Type:    "insert",
			Content: cleanSQLContent(ins.Content),
		})
	}

	// Extract UPDATE statements
	for _, upd := range rawMapper.Updates {
		mapper.SQLs = append(mapper.SQLs, SQL{
			ID:      upd.ID,
			Type:    "update",
			Content: cleanSQLContent(upd.Content),
		})
	}

	// Extract DELETE statements
	for _, del := range rawMapper.Deletes {
		mapper.SQLs = append(mapper.SQLs, SQL{
			ID:      del.ID,
			Type:    "delete",
			Content: cleanSQLContent(del.Content),
		})
	}

	return mapper, nil
}

// cleanSQLContent cleans up SQL content by removing excessive whitespace
func cleanSQLContent(sql string) string {
	// Remove leading/trailing whitespace
	sql = strings.TrimSpace(sql)

	// Normalize whitespace (multiple spaces -> single space)
	sql = normalizeWhitespace(sql)

	return sql
}

// normalizeWhitespace reduces multiple consecutive whitespace to single space
func normalizeWhitespace(s string) string {
	wsRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(wsRegex.ReplaceAllString(s, " "))
}

// GetSQLByID retrieves a SQL statement by its ID
func (m *MapperXML) GetSQLByID(id string) *SQL {
	for i := range m.SQLs {
		if m.SQLs[i].ID == id {
			return &m.SQLs[i]
		}
	}
	return nil
}

// GetSQLsByType returns all SQL statements of a specific type
func (m *MapperXML) GetSQLsByType(sqlType string) []SQL {
	result := []SQL{}
	for _, sql := range m.SQLs {
		if strings.EqualFold(sql.Type, sqlType) {
			result = append(result, sql)
		}
	}
	return result
}

// GetNamespaceName returns the simple class name from the namespace
// Example: "com.company.legacy.UserMapper" -> "UserMapper"
func (m *MapperXML) GetNamespaceName() string {
	parts := strings.Split(m.Namespace, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return m.Namespace
}

// CountStatements returns the total number of SQL statements
func (m *MapperXML) CountStatements() int {
	return len(m.SQLs)
}

// CountByType returns the count of statements by type
func (m *MapperXML) CountByType() map[string]int {
	counts := map[string]int{
		"select": 0,
		"insert": 0,
		"update": 0,
		"delete": 0,
	}

	for _, sql := range m.SQLs {
		counts[strings.ToLower(sql.Type)]++
	}

	return counts
}

// HasNamespace checks if the mapper has a namespace
func (m *MapperXML) HasNamespace() bool {
	return m.Namespace != ""
}

// MatchesJavaInterface checks if this mapper matches a Java interface name
func (m *MapperXML) MatchesJavaInterface(javaClassName string) bool {
	namespaceName := m.GetNamespaceName()
	return strings.EqualFold(namespaceName, javaClassName)
}
