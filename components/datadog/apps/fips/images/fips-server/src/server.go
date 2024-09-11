package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	gliderssh "github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	fips_mode      bool
	is_ssh         bool
	tlsMin, tlsMax string
	port           int
	serverCiphers  []string
	serverCert     string

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
	ServerCmd.PersistentFlags().BoolVarP(&is_ssh, "ssh", "s", false, "Toggle the use of SSH over HTTP")
	ServerCmd.PersistentFlags().BoolVarP(&fips_mode, "fips", "f", false, "Toggle FIPS mode to allow only FIPS ciphers")
}

type ServerInterface interface {
	Listen() error
	Shutdown(ctx context.Context) error
}

type Server struct {
	port    int
	ciphers []uint16
	cert    string
	srv     ServerInterface

	sync.RWMutex
}

type HttpServerWrapper struct {
	httpServer *http.Server
}

func (h *HttpServerWrapper) Listen() error {
	return h.httpServer.ListenAndServeTLS("", "")
}

func (h *HttpServerWrapper) Shutdown(ctx context.Context) error {
	return h.httpServer.Shutdown(ctx)
}

type SshServerWrapper struct {
	sshServer *gliderssh.Server
}

func (s *SshServerWrapper) Listen() error {
	return s.sshServer.ListenAndServe()
}

func (s *SshServerWrapper) Shutdown(ctx context.Context) error {
	return s.sshServer.Shutdown(ctx)
}

func loadPrivateKeyFromFile(keyPath string) (ssh.Signer, error) {
	// Read the private key file
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	// Parse the private key
	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, nil
}

func findFirstKeyFileInHome() (string, error) {
	var err error
	var keyFilePath string

	// Walk through the home directory to find the first .key file
	err = filepath.Walk("/build", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the file has a .key extension
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".key") {
			keyFilePath = path
			return filepath.SkipDir // Stop after finding the first .key file
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking the path: %v", err)
	}

	if keyFilePath == "" {
		return "", fmt.Errorf("no .key file found in the home directory")
	}

	return keyFilePath, nil
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

	if is_ssh {
		log.Print("SSH server starting")
		var allowed_ciphers []string
		var allowed_keyExchanges []string
		// Ciphers and keys gotten from https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.27.0:ssh/common.go;l=27
		if fips_mode {
			allowed_ciphers = []string{"aes128-gcm@openssh.com"}
			allowed_keyExchanges = []string{"ecdh-sha2-nistp256"}
		} else {
			allowed_ciphers = []string{"aes128-cbc"}
			allowed_keyExchanges = []string{"ecdh-sha2-nistp256"}
		}
		keyPath, err := findFirstKeyFileInHome()
		if err != nil {
			log.Fatalf("Failed to find .key file: %v", err)
		}

		privateKey, err := loadPrivateKeyFromFile(keyPath)
		if err != nil {
			log.Fatalf("Failed to load private key: %v", err)
		}
		sshConfig := &ssh.ServerConfig{
			Config: ssh.Config{
				Ciphers:      allowed_ciphers, // Set allowed ciphers
				KeyExchanges: allowed_keyExchanges,
			},
		}
		s.srv = &SshServerWrapper{
			sshServer: &gliderssh.Server{
				Addr: ":443",
				PublicKeyHandler: func(ctx gliderssh.Context, key gliderssh.PublicKey) bool {
					log.Print("New connection attempt...")
					return true
				},
				PasswordHandler: func(ctx gliderssh.Context, password string) bool {
					log.Print("New connection attempt...")
					return true
				},
				HostSigners: []gliderssh.Signer{privateKey},
				ServerConfigCallback: func(ctx gliderssh.Context) *ssh.ServerConfig {
					return sshConfig
				},
			},
		}
	} else {
		log.Print("HTTP server starting")
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
			MinVersion:               tlsMin,
			MaxVersion:               tlsMax,
			Certificates:             []tls.Certificate{pair},
			CipherSuites:             s.ciphers,
			PreferServerCipherSuites: true,
		}

		s.srv = &HttpServerWrapper{
			httpServer: &http.Server{
				Addr:         fmt.Sprintf(":%d", port),
				Handler:      mux,
				TLSConfig:    cfg,
				TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
			},
		}
	}
	log.Printf("Server Starting...")
	go func() {
		s.srv.Listen()
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
		if is_ssh {
			log.Printf("HTTP server Shutdown: %v", err)
		} else {
			log.Printf("SSH server Shutdown: %v", err)
		}
	}
	log.Printf("Server shutdown...")

	return err

}
