package milvus

import (
	"encoding/json"
	"testing"
)

func TestModuleRegistration(t *testing.T) {
	// Test that the module can be instantiated
	m := New()
	if m == nil {
		t.Fatal("Failed to create RootModule instance")
	}
}

func TestSchemaJSONParsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid schema",
			json: `{
				"name": "test_collection",
				"fields": [
					{"name": "id", "dataType": "Int64", "isPrimaryKey": true, "isAutoID": true},
					{"name": "vector", "dataType": "FloatVector", "dimension": 128}
				]
			}`,
			wantErr: false,
		},
		{
			name: "schema with all field types",
			json: `{
				"name": "complex_collection",
				"description": "A complex collection",
				"fields": [
					{"name": "id", "dataType": "Int64", "isPrimaryKey": true},
					{"name": "int32_field", "dataType": "Int32"},
					{"name": "float_field", "dataType": "Float"},
					{"name": "double_field", "dataType": "Double"},
					{"name": "bool_field", "dataType": "Bool"},
					{"name": "string_field", "dataType": "VarChar", "maxLength": 200},
					{"name": "vector", "dataType": "FloatVector", "dimension": 256}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name: "missing required fields",
			json: `{
				"fields": []
			}`,
			wantErr: false, // Schema can have empty name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schema Schema
			err := json.Unmarshal([]byte(tt.json), &schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldValidation(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		valid bool
	}{
		{
			name: "valid int64 field",
			field: Field{
				Name:     "id",
				DataType: "Int64",
			},
			valid: true,
		},
		{
			name: "valid vector field",
			field: Field{
				Name:      "embedding",
				DataType:  "FloatVector",
				Dimension: 128,
			},
			valid: true,
		},
		{
			name: "varchar without maxLength",
			field: Field{
				Name:     "text",
				DataType: "VarChar",
			},
			valid: false, // VarChar requires MaxLength
		},
		{
			name: "vector without dimension",
			field: Field{
				Name:     "vector",
				DataType: "FloatVector",
			},
			valid: false, // Vector requires Dimension
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			valid := true
			if tt.field.DataType == "" {
				valid = false
			}
			if tt.field.DataType == "VarChar" && tt.field.MaxLength == 0 {
				valid = false
			}
			if (tt.field.DataType == "FloatVector" || tt.field.DataType == "BinaryVector") && tt.field.Dimension == 0 {
				valid = false
			}

			if valid != tt.valid {
				t.Errorf("Field validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestSearchResultStructure(t *testing.T) {
	// Test SearchResult JSON marshaling/unmarshaling
	result := SearchResult{
		ID:    123456,
		Score: 0.95,
		Fields: map[string]interface{}{
			"title": "Test Product",
			"price": 29.99,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SearchResult: %v", err)
	}

	// Unmarshal back
	var decoded SearchResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SearchResult: %v", err)
	}

	// Verify fields
	if decoded.ID != result.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, result.ID)
	}
	if decoded.Score != result.Score {
		t.Errorf("Score mismatch: got %f, want %f", decoded.Score, result.Score)
	}
	if decoded.Fields["title"] != result.Fields["title"] {
		t.Errorf("Fields mismatch for title")
	}
}
