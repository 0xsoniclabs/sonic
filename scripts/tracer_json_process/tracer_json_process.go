package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// Define structs to match the JSON structure:
// { "data": [ { "spans": [ { "duration": 123 } ] } ] }

type Span struct {
	Duration int `json:"duration"`
}

type TraceData struct {
	Spans []Span `json:"spans"`
}

type Root struct {
	Data []TraceData `json:"data"`
}

func main() {
	// 1. Read command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename.json>")
		os.Exit(1)
	}
	filename := os.Args[1]

	// 2. Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 3. Decode JSON
	var root Root
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&root); err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	// 4. Flatten all durations into a single slice of floats
	var durations []float64
	for _, item := range root.Data {
		for _, span := range item.Spans {
			durations = append(durations, float64(span.Duration))
		}
	}

	// Handle empty data case
	if len(durations) == 0 {
		fmt.Println("No duration data found.")
		return
	}

	// 5. Calculate Statistics

	// -- Average (Mean) --
	var sum float64
	for _, d := range durations {
		sum += d
	}
	mean := sum / float64(len(durations))

	// -- Standard Deviation --
	// Formula: sqrt( sum((x - mean)^2) / N )
	var varianceSum float64
	for _, d := range durations {
		varianceSum += math.Pow(d-mean, 2)
	}
	stdDev := math.Sqrt(varianceSum / float64(len(durations)))

	// -- 95th Percentile --
	// We must sort the data to find the percentile
	sort.Float64s(durations)
	// Calculate index: 0.95 * N
	p95Index := int(math.Ceil(0.95*float64(len(durations)))) - 1
	// Safety check for index boundaries
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	} else if p95Index < 0 {
		p95Index = 0
	}
	p95 := durations[p95Index]

	// 6. Output Results
	fmt.Println("--- Statistics ---")
	fmt.Printf("Count:   %d\n", len(durations))
	fmt.Printf("Average: %.2f\n", mean)
	fmt.Printf("P95:     %.2f\n", p95)
	fmt.Printf("StdDev:  %.2f\n", stdDev)
}
