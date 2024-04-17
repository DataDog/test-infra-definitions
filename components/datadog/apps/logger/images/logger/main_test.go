package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPostHandler(t *testing.T) {

	tests := []struct {
		name     string
		jsonData []byte
		expected []string
	}{
		{
			name: "simple",
			jsonData: []byte(`{
				"data": [
				  {
					"message": "some text"
				  }
				]
			  }`),
			expected: []string{"some text"},
		},
		{
			name: "two messages",
			jsonData: []byte(`{
				"data": [
				  {
					"message": "message ONE"
				  },
				  {
					"message": "message TWO"
				  }
				]
			  }`),
			expected: []string{"message ONE", "message TWO"},
		}, {
			name: "encoded",
			jsonData: []byte(`{
				"data": [
				  {
					"message": "c29tZSB0ZXh0",
					"encoded": true
				  }
				]
			  }`),
			expected: []string{"some text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := NewWriterMock()
			stderr := NewWriterMock()
			l := NewLoggerHandler(stdout, stderr)
			svr := httptest.NewServer(http.HandlerFunc(l.handleRequest))
			defer svr.Close()

			res, err := http.Post(svr.URL, "application/json", bytes.NewBuffer(tt.jsonData))
			if err != nil {
				t.Fatalf("could not send POST request: %v", err)
			}
			io.ReadAll(res.Body)
			res.Body.Close()

			stdout.mu.Lock()
			defer stdout.mu.Unlock()
			if len(stdout.writen) != len(tt.expected) {
				t.Fatalf("expected %d write, got %d", len(tt.expected), (stdout.writen))
			}
			for i, expected := range tt.expected {
				if string(stdout.writen[i]) != expected {
					t.Fatalf("expected '%s', got %s", expected, (stdout.writen[i]))
				}
			}
			if len(stderr.writen) != 0 {
				t.Fatalf("expected 0 write, got %d", len(stderr.writen))
			}
		})
	}
}

func TestPostHandler_stderr(t *testing.T) {
	stdout := NewWriterMock()
	stderr := NewWriterMock()
	l := NewLoggerHandler(stdout, stderr)
	svr := httptest.NewServer(http.HandlerFunc(l.handleRequest))
	defer svr.Close()
	jsonData := []byte(`{
		"data": [
		  {
			"message": "some text",
			"output": "stderr"
		  }
		]
	  }`)
	res, err := http.Post(svr.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("could not send POST request: %v", err)
	}
	io.ReadAll(res.Body)
	res.Body.Close()

	stdout.mu.Lock()
	defer stdout.mu.Unlock()
	if len(stderr.writen) != 1 {
		t.Fatalf("expected 1 write, got %d", len(stderr.writen))
	}
	if string(stderr.writen[0]) != "some text" {
		t.Fatalf("expected 'some text', got %s", (stderr.writen[0]))
	}
	if len(stdout.writen) != 0 {
		t.Fatalf("expected 0 write, got %d", len(stdout.writen))
	}

}

type WriterMock struct {
	mu     sync.Mutex
	writen [][]byte
}

func (w *WriterMock) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writen = append(w.writen, p)
	return len(p), nil
}

func NewWriterMock() *WriterMock {
	return &WriterMock{
		writen: make([][]byte, 0),
	}
}
