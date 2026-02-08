package analyzer

import (
	"fmt"
	"testing"

	"spec-recon/internal/model"
)

// TestRecursiveGenerics verifies that List<T> is correctly unwrapped and its inner schema T
// is recursively resolved and added to the fields list.
func TestRecursiveGenerics(t *testing.T) {
	// 1. Setup Mock Data
	membersFieldType := "List<MemberDTO>"

	// ClassMap simulates the linker's pool of classes
	classMap := map[string]*model.Node{
		"TeamDTO": {
			ID:   "TeamDTO",
			Type: model.NodeTypeUtil, // Placeholder
		},
		"MemberDTO": {
			ID:   "MemberDTO",
			Type: model.NodeTypeUtil, // Placeholder
		},
	}

	// FieldTypeMap simulates the field definitions
	fieldTypeMap := map[string]map[string]string{
		"TeamDTO": {
			"members":  membersFieldType,
			"teamName": "String",
		},
		"MemberDTO": {
			"name": "String",
			"role": "String",
		},
	}

	// 2. Execute Resolution
	// We resolve schema for "TeamDTO". Expecting depth 1 for children.
	fields := resolveSchema("TeamDTO", classMap, fieldTypeMap)

	// 3. Verification
	fmt.Println("Resolution Results:")
	foundMembers := false
	foundMemberName := false

	for _, f := range fields {
		indent := ""
		for i := 0; i < f.Depth; i++ {
			indent += "  "
		}
		fmt.Printf("%s└ %s (%s) [Depth: %d]\n", indent, f.Name, f.Type, f.Depth)

		if f.Name == "members" {
			foundMembers = true
			if f.Type != "List<MemberDTO>" {
				t.Errorf("Expected members type 'List<MemberDTO>', got '%s'", f.Type)
			}
		}

		// Check for flattened children
		if f.Name == "name" && f.Depth == 2 {
			foundMemberName = true
		}
	}

	if !foundMembers {
		t.Error("Failed to find 'members' field in TeamDTO")
	}

	if !foundMemberName {
		t.Error("Failed to find 'name' field (child of MemberDTO) flattened in the list")
	}
}
