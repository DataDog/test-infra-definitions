package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
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
			expected: []string{"some text\n"},
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
			expected: []string{"message ONE\n", "message TWO\n"},
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
			expected: []string{"some text\n"},
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
	if string(stderr.writen[0]) != "some text\n" {
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

func TestUDPSender(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not resolve UDP address: %v", err)
	}

	l, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatalf("could not listen on UDP: %v", err)
	}
	l.SetDeadline(time.Now().Add(1 * time.Second))
	defer l.Close()

	stop := make(chan struct{})
	output := make(chan string, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		t.Log("Starting listener")
		defer wg.Done()
		for {
			select {
			case <-stop:
				t.Log("Stopped listener")
				return
			default:
				t.Log("Listening for messages")
				buf := make([]byte, 1024)
				n, _, err := l.ReadFromUDP(buf)
				if err != nil {
					return
				}
				output <- string(buf[:n])
			}
		}
	}()

	c, err := net.DialUDP("udp", nil, l.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("could not dial UDP: %v", err)
	}
	defer c.Close()

	lh := NewLoggerHandler(c, c)
	svr := httptest.NewServer(http.HandlerFunc(lh.handleRequest))
	defer svr.Close()

	jsonData := []byte(`{
		"data": [
		  {
			"message": "message ONE"
		  },
		  {
			"message": "message TWO"
		  }
		]
	  }`)
	t.Log("Stopping listener")
	res, err := http.Post(svr.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("could not send POST request: %v", err)
	}
	io.ReadAll(res.Body)
	res.Body.Close()

	actual := <-output

	if actual != "message ONE\n" {
		t.Fatalf("expected 'message ONE\n', got '%s'", actual)
	}

	actual = <-output

	if actual != "message TWO\n" {
		t.Fatalf("expected 'message TWO\n', got '%s'", actual)
	}
	close(stop)
	wg.Wait()
}
