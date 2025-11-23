package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	VectorDim      = 128
	NumGroups      = 10
	VectorsPerGroup = 10
	NumNoiseVectors = 100
	NumTestQueries  = 10
	TopK           = 10
)

type Vector []float32

type TrainData struct {
	ID        []int64     `json:"id"`
	GroupID   []int64     `json:"group_id"`
	Category  []string    `json:"category"`
	Embedding []Vector    `json:"embedding"`
}

type TestData struct {
	QueryID   []int       `json:"query_id"`
	GroupID   []int       `json:"group_id"`
	Embedding []Vector    `json:"embedding"`
}

type Neighbors struct {
	QueryID   int     `json:"query_id"`
	GroupID   int     `json:"group_id"`
	Neighbors []int64 `json:"neighbors"`
	Distances []float32 `json:"distances"` // Actual L2 distances
}

// VectorDistance represents a vector with its ID and distance to query
type VectorDistance struct {
	ID       int64
	GroupID  int64
	Distance float32
}

// Generate a base vector with reproducible pattern
func generateBaseVector(dim int, seed int) Vector {
	vector := make(Vector, dim)
	for i := 0; i < dim; i++ {
		vector[i] = float32(math.Sin(float64(seed+i)*0.1)*0.5 + 0.5)
	}
	return vector
}

// Generate a similar vector by adding small noise
func generateSimilarVector(baseVector Vector, noise float64, rng *rand.Rand) Vector {
	vector := make(Vector, len(baseVector))
	for i, val := range baseVector {
		vector[i] = float32(float64(val) + (rng.Float64()-0.5)*noise)
	}
	return vector
}

// Generate a dissimilar vector
func generateDissimilarVector(dim int, seed int) Vector {
	vector := make(Vector, dim)
	for i := 0; i < dim; i++ {
		vector[i] = float32(math.Cos(float64(seed*2+i)*0.3)*0.5 + 0.5)
	}
	return vector
}

// calculateL2Distance calculates L2 (Euclidean) distance between two vectors
// L2 distance = sqrt(sum((a[i] - b[i])^2))
func calculateL2Distance(v1, v2 Vector) float32 {
	if len(v1) != len(v2) {
		panic("vectors must have the same dimension")
	}

	var sum float32
	for i := 0; i < len(v1); i++ {
		diff := v1[i] - v2[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// calculateCosineDistance calculates cosine distance between two vectors
// Cosine distance = 1 - cosine similarity
// Cosine similarity = dot_product / (norm1 * norm2)
func calculateCosineDistance(v1, v2 Vector) float32 {
	if len(v1) != len(v2) {
		panic("vectors must have the same dimension")
	}

	var dotProduct, norm1, norm2 float32
	for i := 0; i < len(v1); i++ {
		dotProduct += v1[i] * v2[i]
		norm1 += v1[i] * v1[i]
		norm2 += v2[i] * v2[i]
	}

	norm1 = float32(math.Sqrt(float64(norm1)))
	norm2 = float32(math.Sqrt(float64(norm2)))

	if norm1 == 0 || norm2 == 0 {
		return 1.0 // Maximum distance for zero vectors
	}

	cosineSimilarity := dotProduct / (norm1 * norm2)
	return 1.0 - cosineSimilarity
}

// findTrueNeighbors finds the actual top-K nearest neighbors by calculating real distances
func findTrueNeighbors(queryVector Vector, trainVectors []Vector, trainIDs []int64, trainGroupIDs []int64, topK int, metricType string) ([]int64, []float32) {
	distances := make([]VectorDistance, len(trainVectors))

	// Calculate distance to each training vector
	for i, trainVector := range trainVectors {
		var distance float32
		if metricType == "L2" {
			distance = calculateL2Distance(queryVector, trainVector)
		} else if metricType == "COSINE" {
			distance = calculateCosineDistance(queryVector, trainVector)
		} else {
			distance = calculateL2Distance(queryVector, trainVector) // default to L2
		}

		distances[i] = VectorDistance{
			ID:       trainIDs[i],
			GroupID:  trainGroupIDs[i],
			Distance: distance,
		}
	}

	// Sort by distance (ascending - smaller distance = more similar)
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})

	// Take top-K
	k := topK
	if k > len(distances) {
		k = len(distances)
	}

	neighborIDs := make([]int64, k)
	neighborDistances := make([]float32, k)
	for i := 0; i < k; i++ {
		neighborIDs[i] = distances[i].ID
		neighborDistances[i] = distances[i].Distance
	}

	return neighborIDs, neighborDistances
}

func main() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("Generating recall validation data...")
	fmt.Printf("- Vector dimension: %d\n", VectorDim)
	fmt.Printf("- Groups: %d\n", NumGroups)
	fmt.Printf("- Vectors per group: %d\n", VectorsPerGroup)
	fmt.Printf("- Noise vectors: %d\n", NumNoiseVectors)
	fmt.Printf("- Test queries: %d\n", NumTestQueries)

	// Generate train data
	trainData := TrainData{
		ID:        make([]int64, 0),
		GroupID:   make([]int64, 0),
		Category:  make([]string, 0),
		Embedding: make([]Vector, 0),
	}

	currentID := int64(1)

	// Create groups of similar vectors
	for groupID := 0; groupID < NumGroups; groupID++ {
		baseVector := generateBaseVector(VectorDim, groupID*100)

		for i := 0; i < VectorsPerGroup; i++ {
			var vector Vector
			if i == 0 {
				vector = baseVector
			} else {
				vector = generateSimilarVector(baseVector, 0.05, rng)
			}

			trainData.ID = append(trainData.ID, currentID)
			trainData.GroupID = append(trainData.GroupID, int64(groupID))
			trainData.Category = append(trainData.Category, fmt.Sprintf("Group%d", groupID))
			trainData.Embedding = append(trainData.Embedding, vector)
			currentID++
		}
	}

	// Add noise vectors
	for i := 0; i < NumNoiseVectors; i++ {
		vector := generateDissimilarVector(VectorDim, i*50)
		trainData.ID = append(trainData.ID, currentID)
		trainData.GroupID = append(trainData.GroupID, -1)
		trainData.Category = append(trainData.Category, "Noise")
		trainData.Embedding = append(trainData.Embedding, vector)
		currentID++
	}

	fmt.Printf("Total train vectors: %d\n", len(trainData.ID))

	// Generate test queries (one per group)
	testData := TestData{
		QueryID:   make([]int, 0),
		GroupID:   make([]int, 0),
		Embedding: make([]Vector, 0),
	}

	neighbors := make([]Neighbors, 0)

	// Use L2 distance metric (same as used in Milvus)
	metricType := "L2"
	fmt.Printf("Using metric type: %s\n", metricType)

	for groupID := 0; groupID < NumGroups; groupID++ {
		// Use the base vector as query
		queryVector := generateBaseVector(VectorDim, groupID*100)

		testData.QueryID = append(testData.QueryID, groupID)
		testData.GroupID = append(testData.GroupID, groupID)
		testData.Embedding = append(testData.Embedding, queryVector)

		// Calculate TRUE ground truth by computing actual distances
		fmt.Printf("Computing ground truth for query %d... ", groupID)
		neighborIDs, distances := findTrueNeighbors(
			queryVector,
			trainData.Embedding,
			trainData.ID,
			trainData.GroupID,
			TopK,
			metricType,
		)

		// Verify how many neighbors are from the expected group
		expectedGroupCount := 0
		for i, id := range neighborIDs {
			// Find the group ID of this neighbor
			for j, trainID := range trainData.ID {
				if trainID == id {
					if trainData.GroupID[j] == int64(groupID) {
						expectedGroupCount++
					}
					break
				}
			}
			if i < 3 {
				fmt.Printf("ID=%d(dist=%.4f) ", id, distances[i])
			}
		}
		fmt.Printf("... %d/%d from expected group\n", expectedGroupCount, len(neighborIDs))

		neighbors = append(neighbors, Neighbors{
			QueryID:   groupID,
			GroupID:   groupID,
			Neighbors: neighborIDs,
			Distances: distances,
		})
	}

	// Create output directory if not exists
	if err := os.MkdirAll("examples/recall-data", 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write train data
	if err := writeJSON("examples/recall-data/train.json", trainData); err != nil {
		fmt.Printf("Error writing train.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Generated examples/recall-data/train.json")

	// Write test data
	if err := writeJSON("examples/recall-data/test.json", testData); err != nil {
		fmt.Printf("Error writing test.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Generated examples/recall-data/test.json")

	// Write neighbors data
	if err := writeJSON("examples/recall-data/neighbors.json", neighbors); err != nil {
		fmt.Printf("Error writing neighbors.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Generated examples/recall-data/neighbors.json")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Data generation complete!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total train vectors: %d\n", len(trainData.ID))
	fmt.Printf("Total test queries: %d\n", len(testData.QueryID))
	fmt.Printf("Metric type: %s\n", metricType)
	fmt.Printf("Ground truth computed using ACTUAL distance calculations\n")
	fmt.Println(strings.Repeat("=", 60))
}

func writeJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
