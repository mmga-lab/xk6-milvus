// Package milvus provides a k6 extension for load testing Milvus vector databases.
// It wraps the official Milvus Go SDK to provide k6-friendly methods for vector operations.
//
// This package is organized into multiple files for better maintainability:
//   - module.go: k6 module initialization and metrics management
//   - types.go: type definitions and data structures  
//   - client.go: client implementation and data operations
//   - search.go: search operations and recall calculation
//
// The extension provides comprehensive metrics tracking, flexible schema support,
// and recall calculation for vector search quality assessment.
package milvus