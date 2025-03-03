package main

import (
	"bytes"
	"encoding/base64"
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
	Error         string `json:"error"`
}

func main() {
	http.HandleFunc("POST /run", handleRunCode)
	http.HandleFunc("GET /health", handleHealth)

	port := "8001"
	log.Printf("Starting Go runner service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRunCode(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for %s", r.URL.Path)

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
	response.Error = ""

	decodedBytes, err := base64.StdEncoding.DecodeString(req.Code)
	if err != nil {
		return CodeResponse{}, err
	}

	// Convert bytes to string and print
	req.Code = string(decodedBytes)

	// Create tmp dir, main.go and go.mod files and write code
	mainFile, tmpDir, err := enqueueCode(req.Code)
	if err != nil {
		return CodeResponse{}, err
	}

	var executable = filepath.Join(tmpDir, "runner")

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
	//buildCmd := exec.Command("go", "build", "-o", executable+".exe", mainFile) // windows
	buildCmd := exec.Command("go", "build", "-o", executable, mainFile) // linux

	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		response.Stderr = buildStderr.String()
		response.ExitCode = 1
		response.ExecutionTime = time.Since(start).Milliseconds()
		return response, fmt.Errorf("build failed: %v", err)
	}

	// Run the compiled code
	//runCmd := exec.Command(executable + ".exe") // windows
	runCmd := exec.Command("./" + executable) // linux

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

	// Trim output string
	//var explode1 = strings.Split(response.Stderr, ":")
	//response.Stderr = explode1[2][3:]
	//response.Stderr = response.Stderr[:len(response.Stderr)-1]
	//
	//response.Stdout = response.Stderr
	//response.Stderr = ""

	return response, nil
}

func enqueueCode(code string) (string, string, error) {
	dirName := "tmp"
	err := os.Mkdir(dirName, 0777)
	if err != nil {
		if os.IsExist(err) {
			//fmt.Printf("Directory '%s' already exists\n", dirName)
		} else {
			fmt.Printf("Error creating directory: %v\n", err)
			return "", "", nil
		}
	} else {
		fmt.Printf("Created directory '%s' with full permissions\n", dirName)
	}

	// Create file with full permissions (0666)
	mainFileName := dirName + "/main.go"
	mainFile, err := os.OpenFile(mainFileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return "", "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}(mainFile)

	_, err = mainFile.WriteString(code)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return "", "", err
	}

	// Create file with full permissions (0666)
	modFileName := dirName + "/go.mod"
	modFile, err := os.OpenFile(modFileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return "", "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}(modFile)

	_, err = modFile.WriteString("module main\n\ngo 1.20\n")
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return "", "", err
	}

	return mainFileName, dirName, nil
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}
