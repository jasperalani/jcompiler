package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// processRequest simulates processing the request
// In a real application, this could be a more complex operation
func processRequest(req Request) Response {

	err := os.Remove("output.txt")
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error deleting file:", err)
	}

	// Create a new file
	file, err := os.Create("output.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
	}
	defer file.Close()

	cmd := exec.Command(`docker run --rm -v "$(pwd):/app" -w /app node:18 node -e "` + req.Code + `" > output.txt`)

	if cmd.Err != nil {
		return Response{
			Error: "" + cmd.Err.Error(),
		}
	}

	data, err := os.ReadFile("output.txt")
	if err != nil {
		return Response{
			Error: "Failed to read output.txt: " + err.Error(),
		}
	}

	return Response{
		Output:    string(data),
		Timestamp: time.Now().Format(time.RFC3339),
		Cached:    false,
	}
}
