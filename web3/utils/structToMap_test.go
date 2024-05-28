package utils

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Example struct with json tags
type MyStruct struct {
	Field1  string
	Field2  int    `json:"field22"`
	Field21 *int   `json:"field21"`
	Field3  bool   `json:"field33,omitempty"`
	Field4  []byte `json:"field44,omitempty"`
}

func TestStructToMap(t *testing.T) {
	// Create an instance of the struct
	var z int = 1
	instance := MyStruct{"value1", 42, &z, true, []byte{1, 2, 3}}

	// Convert the struct to a map
	resultMap := StructToMap(instance)

	// Convert the map to JSON for display
	resultJSON, _ := json.MarshalIndent(resultMap, "", "  ")
	fmt.Println(string(resultJSON))

	resultJSON2, _ := json.Marshal(instance)
	fmt.Println(string(resultJSON2))
}
