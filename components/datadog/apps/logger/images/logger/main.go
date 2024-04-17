package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Message struct {
	Message string `json:"message"`
	Encoded bool   `json:"encoded,omitempty"`
	Output  string `json:"output,omitempty"`
}

type Data struct {
	Data []Message `json:"data"`
}

func main() {

	// Set port
	port := flag.Int("port", 3333, "port to listen on")

	// Create a channel to listen for OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	l := NewLoggerHandler(os.Stdout, os.Stderr)
	// Create an HTTP server
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(l.handleRequest))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Start the server in a separate goroutine
	go func() {
		slog.Info("Starting server", "port", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %s\n", err)
		}
	}()

	// Wait for SIGTERM signal
	sig := <-sigCh

	// Gracefully shut down the server with a timeout
	slog.Info("Received operating system signal. Shutting down server...", "signal", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down server: %s\n", err)
	} else {
		slog.Info("Server shut down gracefully")
	}
}

type loggerHandler struct {
	stdout, stderr io.Writer
}

func (l *loggerHandler) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		slog.Warn("Method not allowed", "method", r.Method)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		slog.Error("Error reading request body", "error", err.Error())
		return
	}

	// Unmarshal JSON
	var data Data
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}
	// Process data
	for _, message := range data.Data {
		var output string
		if message.Encoded {
			decoded, err := decodeBase64(message.Message)
			if err != nil {
				http.Error(w, "Error decoding base64", http.StatusBadRequest)
				slog.Error("Error decoding base64", "error", err.Error())
				return
			}
			output = decoded
		} else {
			output = message.Message
		}
		fmt.Println(output)
		switch message.Output {
		case "stderr":
			io.WriteString(l.stderr, output)
		default:
			io.WriteString(l.stdout, output)
		}
	}
}

func NewLoggerHandler(stdout, stderr io.Writer) *loggerHandler {
	return &loggerHandler{stdout: stdout, stderr: stderr}
}

func decodeBase64(encoded string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}
