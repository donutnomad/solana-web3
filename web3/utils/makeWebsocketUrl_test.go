package utils

import (
	"fmt"
	"testing"
)

func TestUrl(t *testing.T) {
	endpoint := "https://example.com:8080/some/path"
	result, err := MakeWebsocketURL(endpoint)
	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Println(result)
}
