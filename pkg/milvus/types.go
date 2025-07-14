// Package milvus provides a k6 extension for load testing Milvus vector databases.
// This file contains type definitions and data structures.
package milvus

// Field represents a field definition for a Milvus collection schema.
// It defines the structure and properties of data fields in a collection.
type Field struct {
	Name         string `json:"name"`                   // Field name
	DataType     string `json:"dataType"`               // Data type (e.g., "Int64", "Float", "FloatVector")
	IsPrimaryKey bool   `json:"isPrimaryKey,omitempty"` // Whether this field is the primary key
	IsAutoID     bool   `json:"isAutoID,omitempty"`     // Whether to auto-generate IDs for this field
	Dimension    int64  `json:"dimension,omitempty"`    // Vector dimension (required for vector fields)
	Description  string `json:"description,omitempty"`  // Field description
	MaxLength    int64  `json:"maxLength,omitempty"`    // Maximum length (required for VarChar fields)
}

// Schema represents a Milvus collection schema.
// It defines the structure of a collection including all its fields.
type Schema struct {
	Name        string  `json:"name"`        // Collection name
	Description string  `json:"description"` // Collection description
	Fields      []Field `json:"fields"`      // List of fields in the collection
}

// SearchResult represents a single search result from a vector search operation.
// Contains the matched entity's ID, similarity score, and optional field values.
type SearchResult struct {
	ID     int64                  `json:"id"`               // Entity ID of the matched result
	Score  float32                `json:"score"`            // Similarity score (higher means more similar)
	Fields map[string]interface{} `json:"fields,omitempty"` // Additional fields returned with the result
}