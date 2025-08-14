package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	logFilePath = "app.log"
	maxLogSize  = 1 * 1024 * 1024 // 1MB
	maxLogs     = 100
)

var (
	apiKey               = getEnv("API_KEY", "mysecretkey")
	maxRequestsPerMinute = getEnvAsInt("MAX_REQUESTS_PER_MINUTE", 10)
	rateLimiter          = make(map[string]rateLimit)
	mu                   sync.Mutex
	logFile              *os.File
	logs                 []LogEntry
)

type rateLimit struct {
	count     int
	startTime time.Time
}

type AttackData struct {
	URL      string `json:"url"`
	Workers  int    `json:"workers"`
	PostSize int    `json:"post_size"`
}

type StatusData struct {
	Running       bool   `json:"running"`
	TotalRequests int    `json:"total_requests"`
	TargetURL     string `json:"target_url"`
	Workers       int    `json:"workers"`
	PostSize      int    `json:"post_size"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

func initLogs() {
	// Create the log file if it doesn't exist, or open it in append mode
	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// If the file is new, initialize it as a JSON array
	fileInfo, err := logFile.Stat()
	if err != nil {
		log.Fatalf("Failed to get log file info: %v", err)
	}
	if fileInfo.Size() == 0 {
		logFile.WriteString("[]")
	}

	// Read existing logs on start
	readLogs()
}

func readLogs() {
	fileContent, err := os.ReadFile(logFilePath)
	if err != nil {
		log.Fatalf("Failed to read log file: %v", err)
	}

	err = json.Unmarshal(fileContent, &logs)
	if err != nil {
		log.Fatalf("Failed to unmarshal log file: %v", err)
	}
}

func writeLog(level, message string) {
	mu.Lock()
	defer mu.Unlock()
	// Check file size before writing
	fileInfo, err := logFile.Stat()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return
	}
	if fileInfo.Size() > maxLogSize {
		rotateLog()
	}

	now := time.Now().Format(time.RFC3339)
	entry := LogEntry{
		Timestamp: now,
		Level:     level,
		Message:   message,
	}
	logs = append(logs, entry)
	if len(logs) > maxLogs {
		logs = logs[1:]
	}

	jsonData, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	err = os.WriteFile(logFilePath, jsonData, 0644)
	if err != nil {
		log.Printf("Failed to write to log file: %v", err)
	}
}

func getLogs() []LogEntry {
	mu.Lock()
	defer mu.Unlock()

	if len(logs) < maxLogs {
		return logs
	} else {
		return logs[len(logs)-maxLogs:]
	}
}

func clearLogs() {
	mu.Lock()
	defer mu.Unlock()
	logs = []LogEntry{}
	logFile.Truncate(0)
	logFile.Seek(0, 0)
	logFile.WriteString("[]")
}

func rotateLog() {
	mu.Lock()
	defer mu.Unlock()

	// Close the current log file
	if logFile != nil {
		logFile.Close()
	}

	// Rename the current log file
	err := os.Rename(logFilePath, logFilePath+".old")
	if err != nil {
		log.Printf("Failed to rename log file: %v", err)
		return
	}

	// Open a new log file
	logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Clear the logs array
	logs = []LogEntry{}

	// Add the json []
	logFile.WriteString("[]")
}

func main() {
	// Print environment variables
	fmt.Println("Environment Variables:")
	fmt.Println("API_KEY:", apiKey)
	fmt.Println("MAX_REQUESTS_PER_MINUTE:", maxRequestsPerMinute)

	initLogs()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Closing log file...")
		if logFile != nil {
			logFile.Close()
		}
		os.Exit(1)
	}()

	http.HandleFunc("/api/attack", apiKeyAuth(attackHandler))
	http.HandleFunc("/api/status", apiKeyAuth(statusHandler))
	http.HandleFunc("/api/logs", apiKeyAuth(logsHandler))

	log.Println("Server listening on https://localhost:8080")
	log.Fatal(http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil))
}

func apiKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAPIKey := r.Header.Get("X-API-Key")
		if reqAPIKey == "" || reqAPIKey != apiKey {
			writeLog("ERROR", fmt.Sprintf("Request unauthorized: invalid API key: %v", reqAPIKey))
			http.Error(w, "Request unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkRateLimit(apiKey string) bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	limit, exists := rateLimiter[apiKey]
	if !exists || now.Sub(limit.startTime) >= time.Minute {
		rateLimiter[apiKey] = rateLimit{count: 1, startTime: now}
		return true
	}

	if limit.count >= maxRequestsPerMinute {
		writeLog("ERROR", fmt.Sprintf("Rate limit reached for apikey %v", apiKey))
		return false
	}

	limit.count++
	rateLimiter[apiKey] = limit
	return true
}

func attackHandler(w http.ResponseWriter, r *http.Request) {
	if !checkRateLimit(r.Header.Get("X-API-Key")) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	switch r.Method {
	case http.MethodPost:
		// Start attack
		writeLog("INFO", "Starting attack")
		var data AttackData
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			writeLog("ERROR", fmt.Sprintf("Error decoding JSON: %v", err))
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}

		//Validate parameters
		if data.URL == "" || !regexp.MustCompile(`^https?://`).MatchString(data.URL) {
			writeLog("ERROR", fmt.Sprintf("Invalid URL: %v", data.URL))
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}
		if data.Workers <= 0 || data.Workers > 1000 {
			writeLog("ERROR", fmt.Sprintf("Invalid workers: %v", data.Workers))
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}
		if data.PostSize <= 0 || data.PostSize > 10000000 {
			writeLog("ERROR", fmt.Sprintf("Invalid post size: %v", data.PostSize))
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}

		writeLog("INFO", fmt.Sprintf("Starting attack to %v with %v workers and %v post size", data.URL, data.Workers, data.PostSize))
		//Simulate the start attack
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Attack started successfully."))
	case http.MethodDelete:
		// Stop attack
		writeLog("INFO", "Stopping attack")
		//Simulate the stop attack
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Attack stopped successfully."))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if !checkRateLimit(r.Header.Get("X-API-Key")) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	writeLog("INFO", "Getting status")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//Simulate the status
	status := StatusData{
		Running:       false,
		TotalRequests: 100,
		TargetURL:     "https://www.example.com",
		Workers:       10,
		PostSize:      1024,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if !checkRateLimit(r.Header.Get("X-API-Key")) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		writeLog("INFO", "Getting logs")
		logsToReturn := getLogs()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logsToReturn)
	} else if r.Method == http.MethodDelete {
		if !checkRateLimit(r.Header.Get("X-API-Key")) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		writeLog("INFO", "Clearing logs")
		clearLogs()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logs cleared"))

	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}
