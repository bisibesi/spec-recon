package analyzer

import (
	"fmt"
	"testing"

	"spec-recon/internal/model"
)

func TestInferMapSchema(t *testing.T) {
	// Setup Dummy maps
	classMap := make(map[string]*model.Node)
	fieldTypeMap := make(map[string]map[string]string)

	t.Run("Scenario 1: Standard Map Usage", func(t *testing.T) {
		mockBody := `{
			Map<String, Object> resultMap = new HashMap<>();
			resultMap.put("userInfo", new UserDto());
			resultMap.put("isSuccess", true);
			return resultMap;
		}`

		node := &model.Node{
			Method: "getUserInfo",
			Body:   mockBody,
		}

		results := inferMapSchema(node, classMap, fieldTypeMap)

		if len(results) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(results))
		}

		fieldMap := make(map[string]string)
		for _, f := range results {
			fieldMap[f.Name] = f.Type
			fmt.Printf("Field: %s, Type: %s\n", f.Name, f.Type)
		}

		if fieldMap["userInfo"] != "UserDto" {
			t.Errorf("Expected userInfo type 'UserDto', got '%s'", fieldMap["userInfo"])
		}

		// "true" literal infers "boolean"
		if fieldMap["isSuccess"] != "boolean" {
			t.Errorf("Expected isSuccess type 'boolean', got '%s'", fieldMap["isSuccess"])
		}
	})

	t.Run("Scenario 2: Wrapped Return", func(t *testing.T) {
		mockBody := `{
			Map<String, Object> data = new HashMap<>();
			data.put("schoolList", new ArrayList<SchoolDto>());
			return new ResponseDto(data);
		}`

		node := &model.Node{
			Method: "getSchoolList",
			Body:   mockBody,
		}

		results := inferMapSchema(node, classMap, fieldTypeMap)

		if len(results) != 1 {
			t.Errorf("Expected 1 field, got %d", len(results))
		}

		if len(results) > 0 {
			f := results[0]
			fmt.Printf("Field: %s, Type: %s\n", f.Name, f.Type)

			if f.Name != "schoolList" {
				t.Errorf("Expected field 'schoolList', got '%s'", f.Name)
			}

			// Note: Current regex might only capture 'ArrayList' without generics
			// If we want 'ArrayList<SchoolDto>', we need to update the regex.
			// Checking contains "ArrayList"
			if f.Type != "ArrayList" && f.Type != "ArrayList<SchoolDto>" && f.Type != "List<SchoolDto>" {
				t.Errorf("Expected Type to contain 'ArrayList' or 'List', got '%s'", f.Type)
			}
		}
	})

	t.Run("Scenario 3: Static Factory", func(t *testing.T) {
		mockBody := `{
			Map<String, Object> map = new HashMap<>();
			map.put("id", 123);
			return Response.ok(map);
		}`

		node := &model.Node{
			Method: "getId",
			Body:   mockBody,
		}

		results := inferMapSchema(node, classMap, fieldTypeMap)

		if len(results) != 1 {
			t.Errorf("Expected 1 field, got %d", len(results))
		}

		if len(results) > 0 {
			if results[0].Name != "id" {
				t.Errorf("Expected field 'id', got '%s'", results[0].Name)
			}
			if results[0].Type != "int" {
				t.Errorf("Expected type 'int', got '%s'", results[0].Type)
			}
		}
	})
}

func TestServiceHop(t *testing.T) {
	// Setup Dummy maps
	classMap := make(map[string]*model.Node)
	fieldTypeMap := make(map[string]map[string]string)

	// Create Service Node and Method
	serviceBody := `{
        Map<String, Object> map = new HashMap<>();
        map.put("serviceKey", "serviceValue");
        return map;
    }`

	serviceNode := &model.Node{
		ID:   "com.example.MyService",
		Type: model.NodeTypeService,
		Children: []*model.Node{
			{
				Method: "getData",
				Body:   serviceBody,
			},
		},
	}

	// Add to classMap using simple name used in cleanTypeName
	classMap["MyService"] = serviceNode

	// Create Controller Node
	controllerNode := &model.Node{
		ID:   "com.example.MyController",
		Type: model.NodeTypeController,
	}

	// Setup FieldTypeMap: Controller has field "myService" of type "MyService"
	fieldTypeMap["com.example.MyController"] = map[string]string{
		"myService": "MyService",
	}

	// Create Controller Method Node which calls service
	controllerMethodBody := `return new ResponseDto(myService.getData());`
	methodNode := &model.Node{
		Method: "endpoint",
		Body:   controllerMethodBody,
		Parent: controllerNode, // Crucial for Service Hop
	}

	// Execute
	results := inferMapSchema(methodNode, classMap, fieldTypeMap)

	// Verify
	if len(results) != 1 {
		t.Errorf("Expected 1 field from service hop, got %d", len(results))
	}
	if len(results) > 0 {
		fmt.Printf("Service Hop Result Field: %s\n", results[0].Name)
		if results[0].Name != "serviceKey" {
			t.Errorf("Expected 'serviceKey', got '%s'", results[0].Name)
		}
	}
}
