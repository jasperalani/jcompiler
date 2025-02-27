package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func forwardToRunner(req CodeRequest) (CodeResponse, error) {
	// Map language to runner URL
	runnerURLs := map[string]string{
		"go":     "http://golang-runner:8081/run",
		"js":     "http://javascript-runner:8082/run",
		"ts":     "http://typescript-runner:8083/run",
		"python": "http://python-runner:8084/run",
	}

	runnerURL, ok := runnerURLs[req.Language]
	if !ok {
		return CodeResponse{Stdout: "1", Error: "unsupported language"}, fmt.Errorf("unsupported language: %s", req.Language)
	}

	// Create HTTP client with timeout
	client := &http.Client{}

	var CodeObj Code
	CodeObj.Code = req.Code

	// Marshal request to JSON
	reqBody, err := json.Marshal(CodeObj)
	if err != nil {
		return CodeResponse{Stdout: "2", Error: err.Error()}, err
	}

	log.Print("code")
	log.Print(string(reqBody))

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", runnerURL, bytes.NewReader(reqBody))
	if err != nil {
		return CodeResponse{Stdout: "3", Error: err.Error()}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Body = http.NoBody

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return CodeResponse{Stdout: "4", Error: err.Error()}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	log.Print(resp.Body)

	// Parse response
	var result CodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CodeResponse{Stdout: "5", Error: err.Error()}, err
	}

	// Handle error response
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("runner returned non-200 status: %d", resp.StatusCode)
	}

	return result, nil
}
