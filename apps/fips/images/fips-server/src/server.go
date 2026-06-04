package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	fipsOnly          bool
	tlsMin, tlsMax    string
	port              int
	serverCiphers     []string
	serverCert        string

	ServerCmd = &cobra.Command{
		Use:          "server",
		Short:        "A server to receive dummy HTTP requests",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := RunServer(port, serverCiphers, serverCert); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	ServerCmd.PersistentFlags().StringSliceVarP(&serverCiphers, "ciphers", "c", GetAvailableTLSCiphersStrings(), "TLS cipher names")
	ServerCmd.PersistentFlags().StringVarP(&serverCert, "cert", "C", "server.pem", "Server TLS certificate (should match selected ciphersuite)")
	ServerCmd.PersistentFlags().StringVarP(&tlsMin, "tls-min", "t", "1.2", "Minimum allowed TLS version (1.0, 1.1, 1.2, 1.3)")
	ServerCmd.PersistentFlags().StringVarP(&tlsMax, "tls-max", "T", "1.2", "Maximum allowed TLS version (1.0, 1.1, 1.2, 1.3)")
	ServerCmd.PersistentFlags().IntVarP(&port, "port", "p", 443, "Port for sever to listen on")

}

type Server struct {
	port    int
	ciphers []uint16
	cert    string
	srv     *http.Server

	sync.RWMutex
}

func RunServer(port int, ciphers []string, cert string) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)

	server := NewServer(port, ciphers, cert)


	if err := server.Start(); err != nil {
        return err
    }

     func() {
		<-sigs
		log.Print("signal received\n")
		if err := server.Shutdown(); err != nil {
			log.Printf("error shutting down: %s", err)
		}
	}()
	log.Printf("Done!")

	return nil
}


func NewServer(port int, ciphers []string, cert string) Server {
    suites := filterCiphers(ciphers)

	return Server{
		port:    port,
		ciphers: suites,
		cert:    cert,
	}
}

func (s *Server) Start() error {
	s.Lock()
	defer s.Unlock()

	pair, err := tls.LoadX509KeyPair(s.cert, s.cert)
	if err != nil {
		return fmt.Errorf("failed to load `%s`: %w", s.cert, err)
	}
	log.Printf("Loaded certificate pair from `%s`.", s.cert)

    tlsMin, tlsMax := verifyTLSVersion(tlsMin, tlsMax)
    displayTLSInfo(tlsMin, tlsMax, s.ciphers)


	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Printf("Received request...")
		err := VerifyTLSInfo(req.TLS)
		if err != nil {
			log.Printf("error: %v", err)
		}

		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte("This is a dummy server.\n"))
	})

	cfg := &tls.Config{
		MinVersion:   tlsMin,
		MaxVersion:   tlsMax,
		Certificates: []tls.Certificate{pair},
		CipherSuites: s.ciphers,
		PreferServerCipherSuites: true,
	}

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	log.Printf("Server Starting...")
    go func() {
	    s.srv.ListenAndServeTLS("", "")
    }()

	return nil
}


func (s *Server) Shutdown() error {
	s.Lock()
	defer s.Unlock()

	if s.srv == nil {
		return fmt.Errorf("no active server configured, nothing to close")
	}
	defer func() { s.srv = nil }()

	var err error
	if err = s.srv.Shutdown(context.Background()); err != nil {
		// Error from closing listeners, or context timeout:
		log.Printf("HTTP server Shutdown: %v", err)
	}
	log.Printf("Server shutdown...")

	return err

}
