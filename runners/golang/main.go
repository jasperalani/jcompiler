package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type CodeRequest struct {
	Code    string            `json:"code"`
	Timeout int               `json:"timeout"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type CodeResponse struct {
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	ExitCode      int    `json:"exitCode"`
	ExecutionTime int64  `json:"executionTime"`
	Error         string `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("POST /run", handleRunCode)
	http.HandleFunc("GET /health", handleHealth)

	port := "8081"
	log.Printf("Starting Go runner service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRunCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	response, err := executeGoCode(req)
	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func executeGoCode(req CodeRequest) (CodeResponse, error) {
	start := time.Now()
	response := CodeResponse{}

	// Create temporary directory

	var tmpDir = "tmp"

	_, err := os.Stat(tmpDir)
	if err != nil {
		//tmpDir, err := os.MkdirTemp("./", "goexec")
		err := os.Mkdir(tmpDir, 0644)
		if err != nil {
			return response, fmt.Errorf("failed to create temp directory: %v", err)
		}
		/*	defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {

			}
		}(tmpDir)*/
	}

	// Remove escaping from json
	log.Print(req.Code)

	// Create temporary file for the code
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(req.Code), 0644); err != nil {
		return response, fmt.Errorf("failed to write code to file: %v", err)
	}

	// Create go.mod file
	modFile := filepath.Join(tmpDir, "go.mod")
	//modContent := "module runner\n\ngo 1.20\n"
	modContent := "module main\n\ngo 1.20\n"
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		return response, fmt.Errorf("failed to create go.mod: %v", err)
	}

	//exec.Command("gofmt", "-s", "-w", ".")

	// Get max execution time from environment or use default
	maxExecTime := 5
	if maxTimeEnv := os.Getenv("MAX_EXECUTION_TIME"); maxTimeEnv != "" {
		if val, err := strconv.Atoi(maxTimeEnv); err == nil {
			maxExecTime = val
		}
	}

	// Override with request timeout if specified and not exceeding maximum
	if req.Timeout > 0 && req.Timeout < maxExecTime {
		maxExecTime = req.Timeout
	}

	// Build the code
	//bu := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "runner"), mainFile)
	//err = bu.Run()
	//if err != nil {
	//	return CodeResponse{}, err
	//}
	//buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "runner")+".exe", mainFile) // windows
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "runner"), mainFile) // linux
	//buildCmd.Dir = tmpDir
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		response.Stderr = buildStderr.String()
		response.ExitCode = 1
		response.ExecutionTime = time.Since(start).Milliseconds()
		return response, fmt.Errorf("build failed: %v", err)
	}

	// Run the compiled code
	//runCmd := exec.Command(filepath.Join(tmpDir, "runner") + ".exe") // windows
	runCmd := exec.Command("./" + filepath.Join(tmpDir, "runner")) // linux
	//runCmd.Dir = tmpDir

	// Set custom environment variables if provided
	if len(req.Env) > 0 {
		runCmd.Env = os.Environ()
		for k, v := range req.Env {
			runCmd.Env = append(runCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set command arguments if provided
	if len(req.Args) > 0 {
		runCmd.Args = append(runCmd.Args, req.Args...)
	}

	var stdout, stderr bytes.Buffer
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr

	// Set timeout
	timeout := time.Duration(maxExecTime) * time.Second

	// Create channel for command completion
	done := make(chan error, 1)
	go func() {
		done <- runCmd.Run()
	}()

	// Wait for completion or timeout
	select {
	case <-time.After(timeout):
		if runCmd.Process != nil {
			err := runCmd.Process.Kill()
			if err != nil {
				return CodeResponse{
					Error: err.Error(),
				}, err
			}
		}
		response.Error = "execution timed out"
		response.ExitCode = -1
	case err := <-done:
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				response.ExitCode = exitErr.ExitCode()
			}
		} else {
			response.ExitCode = 0
		}
	}

	response.Stdout = stdout.String()
	response.Stderr = stderr.String()
	response.ExecutionTime = time.Since(start).Milliseconds()

	return response, nil
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}
