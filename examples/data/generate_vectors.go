// Sample vector data generator
// This utility generates test vector data for xk6-milvus examples
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type SampleData struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	Category string    `json:"category"`
	Price    float64   `json:"price"`
	Vector   []float32 `json:"vector"`
}

var (
	titles = []string{
		"Laptop Computer", "Wireless Mouse", "Mechanical Keyboard",
		"USB-C Cable", "Monitor Stand", "Webcam HD",
		"Gaming Headset", "External SSD", "Laptop Bag",
		"Phone Charger", "Bluetooth Speaker", "Smart Watch",
	}

	categories = []string{
		"Electronics", "Accessories", "Computing", "Audio", "Storage",
	}
)

func generateVector(dim int) []float32 {
	vector := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vector[i] = rand.Float32()
	}
	return vector
}

func generateSampleData(count, dim int) []SampleData {
	rand.Seed(time.Now().UnixNano())

	data := make([]SampleData, count)
	for i := 0; i < count; i++ {
		data[i] = SampleData{
			ID:       int64(i + 1),
			Title:    titles[rand.Intn(len(titles))],
			Category: categories[rand.Intn(len(categories))],
			Price:    float64(rand.Intn(200)) + rand.Float64()*100,
			Vector:   generateVector(dim),
		}
	}

	return data
}

func main() {
	count := 1000      // Number of samples
	dimension := 128   // Vector dimension

	fmt.Printf("Generating %d sample vectors with dimension %d...\n", count, dimension)

	data := generateSampleData(count, dimension)

	// Write to JSON file
	file, err := os.Create("sample_data.json")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return
	}

	fmt.Printf("✓ Generated sample_data.json with %d entries\n", count)
	fmt.Printf("✓ Vector dimension: %d\n", dimension)
	fmt.Printf("✓ File size: %d bytes\n", getFileSize("sample_data.json"))
}

func getFileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}
