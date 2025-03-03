package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func forwardToRunner(req CodeRequest) (CodeResponse, error) {
	// Map language to runner URL
	runnerURLs := map[string]string{
		"go":     "http://golang-runner:8001/run",
		"js":     "http://javascript-runner:8002/run",
		"ts":     "http://typescript-runner:8003/run",
		"python": "http://python-runner:8004/run",
	}

	runnerURL, ok := runnerURLs[req.Language]
	if !ok {
		err := fmt.Errorf("unsupported language: %s", req.Language)
		log.Printf("Error: %v", err)
		return CodeResponse{Error: "unsupported language"}, err
	}

	// Create HTTP client with timeout
	client := &http.Client{}

	var codeObj Code
	codeObj.Code = req.Code

	// Encode to base64
	//codeObj.Code = base64.StdEncoding.EncodeToString([]byte(codeObj.Code))

	// Marshal request to JSON
	reqBody, err := json.Marshal(codeObj)
	if err != nil {
		log.Printf("Error marshalling request: %v", err)
		return CodeResponse{Error: err.Error()}, err
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", runnerURL, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return CodeResponse{Error: err.Error()}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Error sending request to runner: %v", err)
		return CodeResponse{Error: err.Error()}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	// Handle error response status code
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("runner returned non-200 status: %d", resp.StatusCode)
		log.Printf("Error: %v", err)
		return CodeResponse{Error: fmt.Sprintf("runner error (status: %d)", resp.StatusCode)}, err
	}

	// Parse response
	var result CodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response: %v", err)
		return CodeResponse{Error: err.Error()}, err
	}

	return result, nil
}
