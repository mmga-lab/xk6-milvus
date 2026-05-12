package milvus

import (
	"encoding/json"
	"fmt"
	"time"
)

// ListCollections lists all collections via REST API
func (rc *RestClient) ListCollections(dbName ...string) interface{} {
	body := map[string]interface{}{}
	if len(dbName) > 0 && dbName[0] != "" {
		body["dbName"] = dbName[0]
	} else if rc.dbName != "" {
		body["dbName"] = rc.dbName
	}

	data, elapsed, err := rc.post("/collections/list", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// convertSchemaToRest converts our Schema format to REST API format
func convertSchemaToRest(schema Schema) map[string]interface{} {
	body := map[string]interface{}{
		"collectionName": schema.Name,
	}

	if len(schema.Fields) > 0 {
		restFields := make([]map[string]interface{}, 0, len(schema.Fields))

		for _, f := range schema.Fields {
			field := map[string]interface{}{
				"fieldName": f.Name,
				"dataType":  f.DataType,
			}

			if f.IsPrimaryKey {
				field["isPrimary"] = true
			}
			if f.IsAutoID {
				field["autoId"] = true
			}
			if f.Description != "" {
				field["description"] = f.Description
			}
			if f.EnableAnalyzer {
				field["enableAnalyzer"] = true
			}
			if f.EnableMatch {
				field["enableMatch"] = true
			}
			if f.AnalyzerParams != nil {
				field["analyzerParams"] = f.AnalyzerParams
			}
			if f.Nullable != nil {
				field["nullable"] = *f.Nullable
			}
			if f.ElementType != "" {
				field["elementDataType"] = f.ElementType
			}
			if f.ElementType == "Struct" && len(f.StructFields) > 0 {
				structFields := make([]map[string]interface{}, 0, len(f.StructFields))
				for _, sf := range f.StructFields {
					sfMap := map[string]interface{}{
						"fieldName": sf.Name,
						"dataType":  sf.DataType,
					}
					sfParams := map[string]interface{}{}
					if sf.Dimension > 0 {
						sfParams["dim"] = fmt.Sprintf("%d", sf.Dimension)
					}
					if sf.MaxLength > 0 {
						sfParams["max_length"] = fmt.Sprintf("%d", sf.MaxLength)
					}
					if len(sfParams) > 0 {
						sfMap["elementTypeParams"] = sfParams
					}
					structFields = append(structFields, sfMap)
				}
				field["structFields"] = structFields
			}

			typeParams := map[string]interface{}{}
			if f.Dimension > 0 {
				typeParams["dim"] = fmt.Sprintf("%d", f.Dimension)
			}
			if f.MaxLength > 0 {
				typeParams["max_length"] = fmt.Sprintf("%d", f.MaxLength)
			}
			if f.MaxCapacity > 0 {
				typeParams["max_capacity"] = fmt.Sprintf("%d", f.MaxCapacity)
			}
			if len(typeParams) > 0 {
				field["elementTypeParams"] = typeParams
			}

			restFields = append(restFields, field)
		}

		restSchema := map[string]interface{}{
			"fields": restFields,
		}

		// Check if any field has autoId
		for _, f := range schema.Fields {
			if f.IsAutoID {
				restSchema["autoId"] = true
				break
			}
		}

		// Add functions
		if len(schema.Functions) > 0 {
			fns := make([]map[string]interface{}, 0, len(schema.Functions))
			for _, fn := range schema.Functions {
				fnMap := map[string]interface{}{
					"name":             fn.Name,
					"type":             fn.FunctionType,
					"inputFieldNames":  fn.InputFieldNames,
					"outputFieldNames": fn.OutputFieldNames,
				}
				if fn.Params != nil {
					fnMap["params"] = fn.Params
				}
				fns = append(fns, fnMap)
			}
			restSchema["functions"] = fns
		}

		body["schema"] = restSchema
	}

	if schema.NumShards > 0 {
		body["numShards"] = schema.NumShards
	}

	return body
}

// CreateCollection creates a collection via REST API
func (rc *RestClient) CreateCollection(schemaInput interface{}) interface{} {
	start := time.Now()

	// Convert input to Schema
	var schema Schema
	schemaBytes, err := json.Marshal(schemaInput)
	if err != nil {
		return errorResult(float64(time.Since(start).Milliseconds()), fmt.Sprintf("failed to marshal schema: %v", err))
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return errorResult(float64(time.Since(start).Milliseconds()), fmt.Sprintf("failed to unmarshal schema: %v", err))
	}

	body := convertSchemaToRest(schema)
	if rc.dbName != "" {
		body["dbName"] = rc.dbName
	}

	data, elapsed, err := rc.post("/collections/create", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	_ = data
	return successResult(elapsed, map[string]interface{}{"collection": schema.Name})
}

// CreateCollectionFromJSON creates a collection from a JSON schema string
func (rc *RestClient) CreateCollectionFromJSON(schemaJSON string) interface{} {
	start := time.Now()

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return errorResult(float64(time.Since(start).Milliseconds()), fmt.Sprintf("failed to parse schema JSON: %v", err))
	}

	return rc.CreateCollection(schema)
}

// DescribeCollection describes a collection via REST API
func (rc *RestClient) DescribeCollection(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	data, elapsed, err := rc.post("/collections/describe", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// DropCollection drops a collection via REST API
func (rc *RestClient) DropCollection(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	_, elapsed, err := rc.post("/collections/drop", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"collection": name})
}

// HasCollection checks if a collection exists via REST API
func (rc *RestClient) HasCollection(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	data, elapsed, err := rc.post("/collections/has", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var hasResult map[string]interface{}
	json.Unmarshal(data, &hasResult)

	// Normalize to match gRPC client format: {exists: true/false}
	has := false
	if v, ok := hasResult["has"]; ok {
		has, _ = v.(bool)
	}

	return successResult(elapsed, map[string]interface{}{"exists": has})
}

// LoadCollection loads a collection into memory via REST API
func (rc *RestClient) LoadCollection(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	_, elapsed, err := rc.post("/collections/load", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"collection": name})
}

// ReleaseCollection releases a collection from memory via REST API
func (rc *RestClient) ReleaseCollection(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	_, elapsed, err := rc.post("/collections/release", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"collection": name})
}

// GetLoadState gets the load state of a collection via REST API
func (rc *RestClient) GetLoadState(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	data, elapsed, err := rc.post("/collections/get_load_state", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// GetCollectionStats gets statistics of a collection via REST API
func (rc *RestClient) GetCollectionStats(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	data, elapsed, err := rc.post("/collections/get_stats", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// Flush flushes a collection via REST API
func (rc *RestClient) Flush(collectionName ...string) interface{} {
	name := rc.getCollectionName(collectionName...)
	if name == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	_, elapsed, err := rc.post("/collections/flush", rc.baseBody(name))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"collection": name})
}

// RenameCollection renames a collection via REST API
func (rc *RestClient) RenameCollection(collectionName, newCollectionName string) interface{} {
	body := rc.baseBody(collectionName)
	body["newCollectionName"] = newCollectionName

	_, elapsed, err := rc.post("/collections/rename", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{
		"collection":    collectionName,
		"newCollection": newCollectionName,
	})
}
