package milvus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchParamMap(t *testing.T) {
	params := map[string]interface{}{
		"vectorField":  "structA[embedding]",
		"metricType":   "COSINE",
		"groupByField": "id",
		"radius":       0.5,
		"params": map[string]interface{}{
			"ef":    float64(64),
			"range": "strict",
		},
	}

	got := searchParamMap(params)

	assert.Equal(t, float64(64), got["ef"])
	assert.Equal(t, "strict", got["range"])
	assert.Equal(t, 0.5, got["radius"])
	assert.NotContains(t, got, "vectorField")
	assert.NotContains(t, got, "metricType")
	assert.NotContains(t, got, "groupByField")
	assert.NotContains(t, got, "params")
}

func TestSearchParamValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "string", value: "COSINE", want: "COSINE"},
		{name: "bool", value: true, want: "true"},
		{name: "int", value: 64, want: "64"},
		{name: "float", value: 0.25, want: "0.25"},
		{name: "object", value: map[string]interface{}{"level": 1}, want: `{"level":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, searchParamValue(tt.value))
		})
	}
}

func TestOptionHelpers(t *testing.T) {
	options := map[string]interface{}{
		"limit":           float64(10),
		"offset":          "5",
		"strictGroupSize": "true",
		"groupByField":    123,
	}

	limit, ok := intOption(options, "limit")
	assert.True(t, ok)
	assert.Equal(t, 10, limit)

	offset, ok := intOption(options, "offset")
	assert.True(t, ok)
	assert.Equal(t, 5, offset)

	strict, ok := boolOption(options, "strictGroupSize")
	assert.True(t, ok)
	assert.True(t, strict)

	groupBy, ok := stringOption(options, "groupByField")
	assert.True(t, ok)
	assert.Equal(t, "123", groupBy)
}

func TestParseQueryArgs(t *testing.T) {
	client := &Client{defaultCollection: "default_collection"}

	coll, options := client.parseQueryArgs(map[string]interface{}{
		"collectionName": "custom_collection",
		"limit":          float64(20),
	})
	assert.Equal(t, "custom_collection", coll)
	assert.Equal(t, float64(20), options["limit"])

	coll, options = client.parseQueryArgs("string_collection")
	assert.Equal(t, "string_collection", coll)
	assert.Empty(t, options)
}
