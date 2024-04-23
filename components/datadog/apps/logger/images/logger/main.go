package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	// Set flags
	flag.Int("port", 3333, "port to listen on")
	flag.Bool("udp", false, "send logs via UDP")
	flag.Bool("tcp", false, "send logs via TCP")
	flag.String("target", "", "if sending logs via UDP or TCP, specify the target host:port")
	flag.String("data", "", "path to JSON data file with messages to log")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Get flags
	port := viper.GetInt("port")
	useUDP := viper.GetBool("udp")
	useTCP := viper.GetBool("tcp")
	target := viper.GetString("target")
	data := viper.GetString("data")

	// Create a channel to listen for OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var stdout, stderr io.Writer
	if useUDP {
		// Create a UDP sender
		addr, err := net.ResolveUDPAddr("udp", target)
		if err != nil {
			slog.Error("Error resolving UDP address", err)
			os.Exit(1)
		}
		c, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			slog.Error("Error dialing UDP", err)
			os.Exit(1)
		}
		defer c.Close()
		stdout = c
		stderr = c
	} else if useTCP {
		// Create a TCP sender
		addr, err := net.ResolveTCPAddr("tcp", target)
		if err != nil {
			slog.Error("Error resolving TCP address", err)
			os.Exit(1)
		}
		c, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			slog.Error("Error dialing TCP", err)
			os.Exit(1)
		}
		defer c.Close()
		stdout = c
		stderr = c
	} else {
		stdout = os.Stdout
		stderr = os.Stderr
	}

	l := NewLoggerHandler(stdout, stderr)
	// Create an HTTP server
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(l.handleRequest))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start the server in a separate goroutine
	go func() {
		slog.Info("Starting server", "port", port)
		if data != "" {
			go logData(data, port)
		}
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

func logData(pathToData string, port int) {
	slog.Info("Logging data", "data", pathToData)
	// Read the JSON data file
	f, err := os.Open(pathToData)
	if err != nil {
		slog.Error("Error opening data file", "error", err.Error())
		return
	}
	defer f.Close()
	bs, err := io.ReadAll(f)
	if err != nil {
		slog.Error("Error reading data file", "error", err.Error())
		return
	}
	buf := bytes.NewBuffer(bs)
	resp, err := http.Post(fmt.Sprintf("http://localhost:%v", port), "application/json", buf)
	if err != nil {
		slog.Error("Error sending POST request", "error", err.Error())
		return
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
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
		switch message.Output {
		case "stderr":
			io.WriteString(l.stderr, output+"\n")
		default:
			io.WriteString(l.stdout, output+"\n")
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
